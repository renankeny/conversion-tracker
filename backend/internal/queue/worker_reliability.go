package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// recoverDispatchPanic evita que um pânico em goroutine de canal derrube o worker inteiro
// (o asynq só recupera pânicos na goroutine do handler).
func recoverDispatchPanic(channel string, in domain.SendEventInput) {
	if rec := recover(); rec != nil {
		slog.Error("dispatch goroutine panic",
			"channel", channel, "event", in.EventName, "event_id", in.EventID,
			"pixel_name", in.PixelName, "error", rec, "stack", string(debug.Stack()))
	}
}

// metaRateLimitCodes são códigos da Graph API que indicam limite de taxa/transiente,
// mesmo quando retornados com HTTP 400/403. Devem ser retentados.
var metaRateLimitCodes = map[int]bool{1: true, 2: true, 4: true, 17: true, 32: true, 341: true, 613: true, 80004: true}

// isPermanentResult identifica erros definitivos (400/401/403) que não devem ser retentados,
// exceto quando o corpo indica rate limit/erro transiente da Meta.
func isPermanentResult(r dispatchResult) bool {
	if !isPermanentHTTPStatus(r.StatusCode) {
		return false
	}
	var body struct {
		Error struct {
			Code        int  `json:"code"`
			IsTransient bool `json:"is_transient"`
		} `json:"error"`
	}
	if json.Unmarshal([]byte(r.Body), &body) == nil {
		if body.Error.IsTransient || metaRateLimitCodes[body.Error.Code] {
			return false
		}
	}
	return true
}

const firstAttemptFailedTTL = 72 * time.Hour

func firstAttemptFailedKey(pixelName, eventID string) string {
	return "counter:first_failed:" + pixelName + ":" + eventID
}

// markFirstAttemptFailed registra que a 1ª tentativa foi contabilizada como Failed.
func markFirstAttemptFailed(ctx context.Context, rdb *redis.Client, pixelName, eventID string) {
	if rdb == nil || eventID == "" {
		return
	}
	if err := rdb.Set(ctx, firstAttemptFailedKey(pixelName, eventID), 1, firstAttemptFailedTTL).Err(); err != nil {
		slog.Warn("mark first attempt failed", "pixel_name", pixelName, "event_id", eventID, "error", err)
	}
}

// consumeFirstAttemptFailed retorna true apenas uma vez, se a 1ª tentativa foi contada como Failed.
// Garante que a correção failed→sent nos contadores aconteça só na transição real.
func consumeFirstAttemptFailed(ctx context.Context, rdb *redis.Client, pixelName, eventID string) bool {
	if rdb == nil || eventID == "" {
		return false
	}
	n, err := rdb.Del(ctx, firstAttemptFailedKey(pixelName, eventID)).Result()
	return err == nil && n == 1
}

// applyCounters atualiza event_counters respeitando retries e replays manuais.
func applyCounters(ctx context.Context, deps Deps, pixel *domain.Pixel, in domain.SendEventInput, eventTime time.Time, retryCount int, hasErr, hasSuccess bool) {
	if deps.Counters == nil {
		return
	}
	failedOnly := hasErr && !hasSuccess
	if retryCount == 0 && !in.IsReplay {
		delta := domain.CounterDelta{Total: 1, Sent: 1}
		if failedOnly {
			delta = domain.CounterDelta{Total: 1, Failed: 1}
			markFirstAttemptFailed(ctx, deps.Redis, pixel.Name, in.EventID)
		}
		if err := deps.Counters.Increment(ctx, pixel.AccountID, pixel.Name, in.EventName, eventTime, delta); err != nil {
			slog.Warn("counter increment failed", "pixel_name", pixel.Name, "event_id", in.EventID, "error", err)
		}
		return
	}
	if !hasErr && hasSuccess && consumeFirstAttemptFailed(ctx, deps.Redis, pixel.Name, in.EventID) {
		if err := deps.Counters.Increment(ctx, pixel.AccountID, pixel.Name, in.EventName, eventTime, domain.CounterDelta{Sent: 1, Failed: -1}); err != nil {
			slog.Warn("counter correction failed", "pixel_name", pixel.Name, "event_id", in.EventID, "error", err)
		}
	}
}

