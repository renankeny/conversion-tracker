package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// claimAggregatedSession move atomicamente a sessão para uma chave de processamento (RENAME).
// Itens que chegarem durante o dispatch criam uma NOVA sessão (e nova task), em vez de serem
// apagados pelo Unlink final. Em retries, a chave de processamento já existente é reutilizada.
func claimAggregatedSession(ctx context.Context, rdb *redis.Client, sessionKey string) (string, []byte, error) {
	// Chave por task: retries da mesma task reutilizam; tasks de sessões novas não colidem.
	taskID, _ := asynq.GetTaskID(ctx)
	procKey := sessionKey + ":processing:" + taskID
	raw, err := rdb.Get(ctx, procKey).Bytes()
	if err == nil && len(raw) > 0 {
		return procKey, raw, nil
	}
	if err != nil && !errors.Is(err, redis.Nil) {
		return procKey, nil, fmt.Errorf("get aggregated processing session: %w", err)
	}
	if rerr := rdb.Rename(ctx, sessionKey, procKey).Err(); rerr != nil {
		if strings.Contains(rerr.Error(), "no such key") {
			return procKey, nil, nil
		}
		return procKey, nil, fmt.Errorf("claim aggregated session: %w", rerr)
	}
	raw, err = rdb.Get(ctx, procKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return procKey, nil, nil
	}
	if err != nil {
		return procKey, nil, fmt.Errorf("get aggregated session from redis: %w", err)
	}
	return procKey, raw, nil
}

func releaseAggregatedSession(ctx context.Context, rdb *redis.Client, procKey string) {
	if err := rdb.Unlink(ctx, procKey).Err(); err != nil {
		slog.Warn("release aggregated session failed", "key", procKey, "error", err)
	}
}

func makeProcessAggregatedPurchaseHandler(deps Deps) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var payload AggregatedPurchasePayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("unmarshal aggregated purchase payload: %w", err)
		}
		if deps.Redis == nil {
			slog.Warn("redis not configured in worker, cannot process aggregated purchase", "key", payload.SessionKey)
			return nil
		}

		procKey, raw, err := claimAggregatedSession(ctx, deps.Redis, payload.SessionKey)
		if err != nil {
			return err
		}
		if len(raw) == 0 {
			return nil
		}

		var session AggregatedSession
		if err := json.Unmarshal(raw, &session); err != nil {
			return fmt.Errorf("unmarshal aggregated session: %w", err)
		}

		totalVal := session.TotalValue
		numItems := session.Count
		in := domain.SendEventInput{
			PixelName:             session.PixelName,
			EventName:             domain.EventPurchase,
			EventID:               session.FirstEventID,
			EventTime:             session.FirstEventTime,
			SourceURL:             session.SourceURL,
			Value:                 &totalVal,
			Currency:              session.Currency,
			OrderID:               strings.Join(session.OrderIDs, ","),
			ContentID:             strings.Join(session.ContentIDs, ","),
			ContentName:           strings.Join(session.ContentNames, ", "),
			ContentType:           "product",
			NumItems:              &numItems,
			ExternalID:            session.ExternalID,
			IDWpp:                 session.IDWpp,
			ClientIP:              session.ClientIP,
			ClientUserAgent:       session.ClientUserAgent,
			FBP:                   session.FBP,
			FBC:                   session.FBC,
			Email:                 session.Email,
			Phone:                 session.Phone,
			FirstName:             session.FirstName,
			LastName:              session.LastName,
			Country:               session.Country,
			State:                 session.State,
			City:                  session.City,
			PostalCode:            session.PostalCode,
			TTClid:                session.TTClid,
			TTP:                   session.TTP,
			GCLID:                 session.GCLID,
			WBraid:                session.WBraid,
			GBraid:                session.GBraid,
			GA4ClientID:           session.GA4ClientID,
			GA4SessionID:          session.GA4SessionID,
			OverrideGadsAccountID: session.OverrideGadsAccountID,
			OverrideGadsActionID:  session.OverrideGadsActionID,
			AggregatedCount:       session.Count,
			AggregatedItems:       session.Items,
		}

		slog.Info("processing aggregated purchase dispatch",
			"pixel", in.PixelName,
			"event_id", in.EventID,
			"items_count", in.AggregatedCount,
			"total_value", totalVal,
		)
		if err := dispatchSendEventInput(ctx, deps, in); err != nil {
			if errors.Is(err, asynq.SkipRetry) {
				releaseAggregatedSession(ctx, deps.Redis, procKey)
			}
			return err
		}
		releaseAggregatedSession(ctx, deps.Redis, procKey)
		return nil
	}
}
