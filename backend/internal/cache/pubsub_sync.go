package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const InvalidateChannel = "cache:invalidate"

type InvalidateEvent struct {
	Kind      string   `json:"kind"`
	Keys      []string `json:"keys,omitempty"`
	AccountID int64    `json:"account_id,omitempty"`
	UserID    int64    `json:"user_id,omitempty"`
}

// PublishInvalidate transmite um evento de invalidação via Pub/Sub para sincronizar réplicas.
func PublishInvalidate(ctx context.Context, rdb *redis.Client, ev InvalidateEvent) {
	if rdb == nil || ev.Kind == "" {
		return
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_ = rdb.Publish(ctx, InvalidateChannel, raw).Err()
}

// RunInvalidationSubscriber escuta invalidações no canal Redis com reconexão automática.
func RunInvalidationSubscriber(ctx context.Context, rdb *redis.Client, onEvent func(context.Context, InvalidateEvent)) {
	if rdb == nil || onEvent == nil {
		return
	}
	for {
		if ctx.Err() != nil {
			return
		}
		consumeInvalidationsOnce(ctx, rdb, onEvent)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func consumeInvalidationsOnce(ctx context.Context, rdb *redis.Client, onEvent func(context.Context, InvalidateEvent)) {
	sub := rdb.Subscribe(ctx, InvalidateChannel)
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		if ctx.Err() == nil {
			slog.Warn("cache pubsub: subscribe failed, retrying", "err", err)
		}
		return
	}
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var ev InvalidateEvent
			if err := json.Unmarshal([]byte(msg.Payload), &ev); err == nil && ev.Kind != "" {
				evCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				onEvent(evCtx, ev)
				cancel()
			}
		}
	}
}

// QueryCache aplica singleflight + cache Redis curto para consultas caras (Meta Spend, Dashboard pública).
type QueryCache struct {
	rdb *redis.Client
	sf  singleflight.Group
}

func NewQueryCache(rdb *redis.Client) *QueryCache {
	return &QueryCache{rdb: rdb}
}

func HashQueryKey(prefix string, parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return prefix + ":" + hex.EncodeToString(h.Sum(nil)[:12])
}

func (qc *QueryCache) GetOrLoadBytes(ctx context.Context, key string, ttl time.Duration, loader func(context.Context) ([]byte, error)) ([]byte, error) {
	if qc != nil && qc.rdb != nil && key != "" {
		if cached, err := qc.rdb.Get(ctx, key).Bytes(); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}
	if qc == nil {
		return loader(ctx)
	}
	val, err, _ := qc.sf.Do(key, func() (any, error) {
		b, lerr := loader(ctx)
		if lerr != nil {
			return nil, lerr
		}
		if qc.rdb != nil && key != "" && len(b) > 0 && ttl > 0 {
			_ = qc.rdb.Set(ctx, key, b, ttl).Err()
		}
		return b, nil
	})
	if err != nil {
		return nil, err
	}
	out, _ := val.([]byte)
	return out, nil
}

