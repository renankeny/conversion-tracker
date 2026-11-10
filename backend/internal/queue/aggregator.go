package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const (
	TypeProcessAggregatedPurchase = "meta:process_aggregated_purchase"
)

type AggregatedPurchasePayload struct {
	SessionKey string `json:"session_key"`
	PixelName  string `json:"pixel_name"`
}

type AggregatedSession struct {
	PixelName       string                  `json:"pixel_name"`
	EventName       string                  `json:"event_name"`
	FirstEventID    string                  `json:"first_event_id"`
	FirstEventTime  int64                   `json:"first_event_time"`
	SourceURL       string                  `json:"source_url"`
	TotalValue      float64                 `json:"total_value"`
	Currency        string                  `json:"currency"`
	Count           int                     `json:"count"`
	OrderIDs        []string                `json:"order_ids"`
	ContentIDs      []string                `json:"content_ids"`
	ContentNames    []string                `json:"content_names"`
	Items           []domain.AggregatedItem `json:"items"`
	ExternalID      string                  `json:"external_id"`
	IDWpp           string                  `json:"id_wpp"`
	ClientIP        string                  `json:"client_ip"`
	ClientUserAgent string                  `json:"client_user_agent"`
	FBP             string                  `json:"fbp"`
	FBC             string                  `json:"fbc"`
	Email           string                  `json:"email"`
	Phone           string                  `json:"phone"`
	FirstName       string                  `json:"first_name"`
	LastName        string                  `json:"last_name"`
	Country         string                  `json:"country"`
	State           string                  `json:"state"`
	City            string                  `json:"city"`
	PostalCode      string                  `json:"postal_code"`
	TTClid          string                  `json:"ttclid"`
	TTP             string                  `json:"ttp"`
	GCLID           string                  `json:"gclid"`
	WBraid          string                  `json:"wbraid"`
	GBraid          string                  `json:"gbraid"`
	GA4ClientID     string                  `json:"ga4_client_id"`
	GA4SessionID    string                  `json:"ga4_session_id"`

	OverrideGadsAccountID *int64 `json:"override_gads_account_id,omitempty"`
	OverrideGadsActionID  string `json:"override_gads_action_id,omitempty"`
}

func NormalizeCustomerKey(in domain.SendEventInput) string {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email != "" {
		return "em:" + email
	}
	phone := util.NormalizePhone(in.Phone)
	if phone != "" {
		return "ph:" + phone
	}
	if in.ExternalID != "" {
		return "ext:" + in.ExternalID
	}
	if in.OrderID != "" {
		return "ord:" + in.OrderID
	}
	return "evt:" + in.EventID
}

func SessionKey(pixelName string, in domain.SendEventInput) string {
	customerKey := NormalizeCustomerKey(in)
	hash := sha256.Sum256([]byte(customerKey))
	return fmt.Sprintf("agg_purchase:%s:%s", pixelName, hex.EncodeToString(hash[:8]))
}

func (e *Enqueuer) AggregateOrEnqueue(ctx context.Context, pixel *domain.Pixel, in domain.SendEventInput, rdb *redis.Client) (bool, error) {
	if pixel == nil || !pixel.AggregationEnabled || in.EventName != domain.EventPurchase || rdb == nil {
		return false, e.EnqueueSendEvent(ctx, in)
	}

	windowMin := pixel.AggregationWindowMinutes
	if windowMin < 1 {
		windowMin = 1
	} else if windowMin > 30 {
		windowMin = 30
	}

	key := SessionKey(pixel.Name, in)
	val := 0.0
	if in.Value != nil {
		val = *in.Value
	}
	currentItem := buildAggregatedItem(in, val)
	ttl := time.Duration(windowMin+60) * time.Minute

	isFirst, ignoredDup, session, err := evalAggregatePurchaseLua(ctx, rdb, key, pixel.Name, in, val, currentItem, ttl)
	if err != nil {
		slog.Warn("lua purchase aggregation failed, falling back to direct enqueue", "err", err, "key", key)
		return false, e.EnqueueSendEvent(ctx, in)
	}
	if ignoredDup {
		slog.Info("ignoring duplicate webhook for same order_id in aggregation session", "pixel", pixel.Name, "order_id", in.OrderID, "gateway", in.Gateway)
		return true, nil
	}

	if isFirst {
		taskPayload, _ := json.Marshal(AggregatedPurchasePayload{SessionKey: key, PixelName: pixel.Name})
		task := asynq.NewTask(TypeProcessAggregatedPurchase, taskPayload)
		delay := time.Duration(windowMin) * time.Minute
		_, err = e.client.EnqueueContext(ctx, task, asynq.Queue(QueueDefault), asynq.ProcessIn(delay), asynq.MaxRetry(3))
		if err != nil {
			slog.Error("failed to schedule aggregated purchase task", "err", err, "key", key)
			_ = rdb.Unlink(ctx, key).Err()
			return false, e.EnqueueSendEvent(ctx, in)
		}
		slog.Info("scheduled aggregated purchase window", "pixel", pixel.Name, "customer_key", NormalizeCustomerKey(in), "window_minutes", windowMin, "event_id", in.EventID)
	} else {
		slog.Info("added item to existing aggregated purchase buffer", "pixel", pixel.Name, "customer_key", NormalizeCustomerKey(in), "new_count", session.Count, "new_total_value", session.TotalValue)
	}

	return true, nil
}
