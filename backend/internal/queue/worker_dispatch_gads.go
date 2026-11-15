package queue

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
)

func gadsDigits(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func dispatchGoogleAds(ctx context.Context, deps Deps, pixel *domain.Pixel, in domain.SendEventInput, doneTargets map[string]bool) []googleads.SendResult {
	accID := pixel.GoogleAdsAccountID
	if in.OverrideGadsAccountID != nil {
		accID = in.OverrideGadsAccountID
	}
	if accID == nil {
		return nil
	}
	account, ferr := deps.GoogleAdsAcc.FindByID(ctx, *accID)
	if ferr != nil {
		slog.Warn("google ads account lookup failed", "err", ferr, "pixel", pixel.Name, "account_id", *accID)
		return nil
	}
	if in.OverrideGadsAccountID == nil {
		if pxCid := gadsDigits(pixel.GoogleAdsCustomerID); pxCid != "" && pxCid != gadsDigits(account.CustomerID) {
			var matched *domain.GoogleAdsAccount
			if list, lerr := deps.GoogleAdsAcc.ListByAccount(ctx, pixel.AccountID); lerr == nil {
				for i := range list {
					if !list[i].Enabled || !list[i].HasCredentials() || gadsDigits(list[i].CustomerID) != pxCid {
						continue
					}
					if matched == nil || list[i].HasActionForAnyEvent() {
						matched = &list[i]
					}
				}
			}
			if matched == nil {
				return nil
			}
			account = matched
		}
	}
	if !account.Enabled || !account.HasCredentials() {
		return nil
	}

	var actions []*domain.GoogleAdsConversionAction
	if in.OverrideGadsActionID != "" {
		actions = append(actions, &domain.GoogleAdsConversionAction{
			ConversionActionID: in.OverrideGadsActionID,
			EventType:          in.EventName,
			Enabled:            true,
		})
	} else {
		actions = account.FindActionsForEvent(in.EventName)
	}
	if len(actions) == 0 {
		return nil
	}

	var (
		gaWg    sync.Mutex
		inner   sync.WaitGroup
		results []googleads.SendResult
	)
	for _, action := range actions {
		action := action
		targetKey := "google_ads:" + account.CustomerID + ":" + action.ConversionActionID
		if doneTargets[targetKey] {
			continue
		}
		inner.Add(1)
		go func() {
			defer inner.Done()
			defer recoverDispatchPanic("google_ads", in)
			res := deps.GoogleAds.Send(ctx, account, action, in)
			gaWg.Lock()
			results = append(results, res...)
			gaWg.Unlock()
		}()
	}
	inner.Wait()
	return results
}
