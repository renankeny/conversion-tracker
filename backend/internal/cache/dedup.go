package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	EventDedupTTL      = 24 * time.Hour
	RecentActionTTL    = 3 * time.Second
	WebhookDedupTTL    = 24 * time.Hour
	DispatchTargetsTTL = 48 * time.Hour
)

// ClaimEventDedup registra atomicamente o event_id no Redis (SET NX EX).
// Retorna (claimed, redisOK). Se redisOK == false, o chamador faz fallback p/ banco/memória.
func ClaimEventDedup(ctx context.Context, rdb *redis.Client, accountID int64, pixelName, eventID string, ttl time.Duration) (bool, bool) {
	if rdb == nil || eventID == "" || pixelName == "" {
		return false, false
	}
	if ttl <= 0 {
		ttl = EventDedupTTL
	}
	key := fmt.Sprintf("dedup:ev:%d:%s:%s", accountID, pixelName, eventID)
	claimed, err := rdb.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, false
	}
	return claimed, true
}

// ReleaseEventDedup libera a trava de deduplicação caso o enfileiramento falhe.
func ReleaseEventDedup(ctx context.Context, rdb *redis.Client, accountID int64, pixelName, eventID string) {
	if rdb == nil || eventID == "" || pixelName == "" {
		return
	}
	key := fmt.Sprintf("dedup:ev:%d:%s:%s", accountID, pixelName, eventID)
	_ = rdb.Del(ctx, key).Err()
}

// ClaimRecentAction evita disparos repetidos do mesmo evento/usuário em janela curta (ex: 3s).
func ClaimRecentAction(ctx context.Context, rdb *redis.Client, pixelName, externalID, eventType string, ttl time.Duration) (bool, bool) {
	if rdb == nil || pixelName == "" || externalID == "" || eventType == "" {
		return false, false
	}
	if ttl <= 0 {
		ttl = RecentActionTTL
	}
	key := fmt.Sprintf("dedup:recent:%s:%s:%s", pixelName, externalID, eventType)
	claimed, err := rdb.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, false
	}
	return claimed, true
}

// ClaimWebhookDedup evita que retries de gateways (Hotmart, Kiwify, Cakto, etc.)
// re-enfileirem o mesmo pedido/item. Preserva order_bumps/upsells usando itemKey distinto.
func ClaimWebhookDedup(ctx context.Context, rdb *redis.Client, accountID int64, gateway, orderID, eventName, itemKey string) (bool, bool) {
	if rdb == nil || strings.TrimSpace(orderID) == "" {
		return false, false
	}
	gw := strings.ToLower(strings.TrimSpace(gateway))
	ev := strings.TrimSpace(eventName)
	ik := strings.TrimSpace(itemKey)
	if ik == "" {
		ik = "main"
	}
	key := fmt.Sprintf("dedup:wh:%d:%s:%s:%s:%s", accountID, gw, strings.TrimSpace(orderID), ev, ik)
	claimed, err := rdb.SetNX(ctx, key, "1", WebhookDedupTTL).Result()
	if err != nil {
		return false, false
	}
	return claimed, true
}

func ReleaseWebhookDedup(ctx context.Context, rdb *redis.Client, accountID int64, gateway, orderID, eventName, itemKey string) {
	if rdb == nil || strings.TrimSpace(orderID) == "" {
		return
	}
	ik := strings.TrimSpace(itemKey)
	if ik == "" {
		ik = "main"
	}
	key := fmt.Sprintf("dedup:wh:%d:%s:%s:%s:%s", accountID, strings.ToLower(strings.TrimSpace(gateway)), strings.TrimSpace(orderID), strings.TrimSpace(eventName), ik)
	_ = rdb.Del(ctx, key).Err()
}

func dispatchDoneKey(pixelName, eventID string) string {
	return fmt.Sprintf("dispatch:done:%s:%s", pixelName, eventID)
}

// LoadDoneTargets retorna todos os sub-destinos (<channel>:<target_id>) que já receberam
// com sucesso este event_id em tentativas anteriores do worker.
func LoadDoneTargets(ctx context.Context, rdb *redis.Client, pixelName, eventID string) map[string]bool {
	out := make(map[string]bool)
	if rdb == nil || pixelName == "" || eventID == "" {
		return out
	}
	members, err := rdb.SMembers(ctx, dispatchDoneKey(pixelName, eventID)).Result()
	if err != nil {
		return out
	}
	for _, m := range members {
		if m != "" {
			out[m] = true
		}
	}
	return out
}

// MarkTargetsDone grava os sub-destinos concluídos com sucesso no Redis (TTL 48h).
func MarkTargetsDone(ctx context.Context, rdb *redis.Client, pixelName, eventID string, targets []string) {
	if rdb == nil || pixelName == "" || eventID == "" || len(targets) == 0 {
		return
	}
	args := make([]any, 0, len(targets))
	for _, t := range targets {
		if t != "" {
			args = append(args, t)
		}
	}
	if len(args) == 0 {
		return
	}
	key := dispatchDoneKey(pixelName, eventID)
	pipe := rdb.Pipeline()
	pipe.SAdd(ctx, key, args...)
	pipe.Expire(ctx, key, DispatchTargetsTTL)
	_, _ = pipe.Exec(ctx)
}

