package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hibiken/asynq"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/ga4"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/service"
	"github.com/renankeny/conversion-tracker/backend/internal/tiktok"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func dispatchSendEventInput(ctx context.Context, deps Deps, in domain.SendEventInput) error {
	pixel, err := deps.Pixels.FindByName(ctx, in.PixelName)
	if err != nil {
		return fmt.Errorf("pixel %q: %w", in.PixelName, err)
	}
	if !pixel.Enabled {
		slog.Warn("pixel disabled, skipping", "pixel", pixel.Name)
		return nil
	}

	service.EnrichSendInput(ctx, deps.Users, pixel.AccountID, &in)

	var (
		metaResults      []meta.SendResult
		tiktokResults    []tiktok.SendResult
		googleAdsResults []googleads.SendResult
		ga4Result        ga4.SendResult
		hasGA4           bool
		wg               sync.WaitGroup
	)
	alreadySent := make(map[string]bool)
	if deps.SentEvents != nil && in.EventID != "" {
		if sentMap, err := deps.SentEvents.GetSuccessfulChannels(ctx, in.EventID, pixel.Name); err == nil {
			alreadySent = sentMap
		}
	}
	doneTargets := cache.LoadDoneTargets(ctx, deps.Redis, pixel.Name, in.EventID)

	if len(pixel.PixelIDs) > 0 && deps.Meta != nil && !alreadySent["meta"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer recoverDispatchPanic("meta", in)
			metaResults = dispatchMetaWithOAuthFallback(ctx, deps, pixel, in, func(pid string) bool { return doneTargets["meta:"+pid] })
		}()
	}
	if pixel.HasTikTok() && deps.TikTok != nil && !alreadySent["tiktok"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer recoverDispatchPanic("tiktok", in)
			tiktokResults = deps.TikTok.SendFiltered(ctx, pixel, in, func(pid string) bool { return doneTargets["tiktok:"+pid] })
		}()
	}
	if (pixel.HasGoogleAds() || in.OverrideGadsAccountID != nil || in.OverrideGadsActionID != "") &&
		deps.GoogleAds != nil && deps.GoogleAdsAcc != nil && !alreadySent["google_ads"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer recoverDispatchPanic("google_ads", in)
			googleAdsResults = dispatchGoogleAds(ctx, deps, pixel, in, doneTargets)
		}()
	}
	if pixel.GA4MeasurementID != "" && pixel.GA4APISecret != "" && in.GA4ClientID != "" &&
		deps.GA4 != nil && !alreadySent["ga4"] && !doneTargets["ga4:"+pixel.GA4MeasurementID] {
		hasGA4 = true
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer recoverDispatchPanic("ga4", in)
			ga4Result = deps.GA4.SendEvents(ctx, pixel.GA4APISecret, pixel.GA4MeasurementID, &in)
		}()
	}
	wg.Wait()

	results := append(fromMetaResults(metaResults), fromTikTokResults(tiktokResults)...)
	results = append(results, fromGoogleAdsResults(googleAdsResults)...)
	if hasGA4 {
		results = append(results, fromGA4Result(ga4Result, pixel.GA4MeasurementID)...)
	}
	if len(results) > 0 && in.EventID != "" {
		newlyDone := make([]string, 0, len(results))
		for _, r := range results {
			if r.Err == nil && r.PixelID != "" {
				newlyDone = append(newlyDone, r.Channel+":"+r.PixelID)
			}
		}
		cache.MarkTargetsDone(ctx, deps.Redis, pixel.Name, in.EventID, newlyDone)
	}

	hasErr := false
	hasRetryableErr := false
	hasSuccess := false
	eventTime := time.Unix(in.EventTime, 0)
	if in.EventTime <= 0 {
		eventTime = time.Now()
	}
	storeFullPayload := domain.ShouldStoreFullPayload(in.EventName)
	retryCount, _ := asynq.GetRetryCount(ctx)

	if len(results) == 0 && len(alreadySent) == 0 && len(doneTargets) == 0 && storeFullPayload {
		results = append(results, dispatchResult{
			Channel:    "internal",
			PixelID:    pixel.Name,
			StatusCode: 204,
			Body:       `{"tracked_internally":true}`,
		})
	}

	for _, r := range collapseChannelResults(results) {
		if r.Err != nil {
			hasErr = true
			if !isPermanentResult(r) {
				hasRetryableErr = true
			}
			slog.Error("send failed", "channel", r.Channel, "pixel_id", r.PixelID, "event", in.EventName, "event_id", in.EventID, "status", r.StatusCode, "err", util.RedactSecrets(r.Err.Error()))
		} else {
			hasSuccess = true
			slog.Info("send ok", "channel", r.Channel, "pixel_id", r.PixelID, "event", in.EventName, "event_id", in.EventID, "status", r.StatusCode)
		}
		if storeFullPayload || r.Err != nil || retryCount > 0 || in.IsReplay {
			persistResult(ctx, deps.Pool, pixel.AccountID, in, r)
		}
	}

	if len(results) > 0 || len(alreadySent) == 0 {
		applyCounters(ctx, deps, pixel, in, eventTime, retryCount, hasErr, hasSuccess)
	}

	if deps.Redis != nil && storeFullPayload {
		cache.BumpDashboardVersion(ctx, deps.Redis, pixel.AccountID)
	}

	if deps.Redis != nil && len(results) > 0 && (retryCount == 0 || !hasErr) {
		status := "sent"
		if hasErr {
			status = "failed"
		}
		wshub.Publish(ctx, deps.Redis, wshub.LiveEvent{
			AccountID:  pixel.AccountID,
			EventID:    in.EventID,
			EventName:  in.EventName,
			PixelName:  in.PixelName,
			ExternalID: in.ExternalID,
			Channels:   summarizeChannels(results),
			Status:     status,
			Ts:         time.Now().Format(time.RFC3339),
			Value:      purchaseValue(in),
			City:       in.City,
			State:      in.State,
			Country:    in.Country,
		})
	}

	if hasErr {
		if !hasRetryableErr {
			return fmt.Errorf("permanent channel error (4xx): %w", asynq.SkipRetry)
		}
		return fmt.Errorf("at least one channel/pixel failed")
	}
	return nil
}
