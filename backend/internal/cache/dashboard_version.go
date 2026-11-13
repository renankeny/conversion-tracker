package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const dashVersionPrefix = "dash:ver:"

func dashVersionKey(accountID int64) string {
	return dashVersionPrefix + strconv.FormatInt(accountID, 10)
}

// BumpDashboardVersion invalida em O(1) os snapshots cacheados da Dashboard da conta
// (e da visão global 0 do owner) ao incrementar o contador de versão no Redis.
func BumpDashboardVersion(ctx context.Context, rdb *redis.Client, accountID int64) {
	if rdb == nil {
		return
	}
	pipe := rdb.Pipeline()
	k := dashVersionKey(accountID)
	pipe.Incr(ctx, k)
	pipe.Expire(ctx, k, 24*time.Hour)
	if accountID > 0 {
		k0 := dashVersionKey(0)
		pipe.Incr(ctx, k0)
		pipe.Expire(ctx, k0, 24*time.Hour)
	}
	_, _ = pipe.Exec(ctx)
}

// DashboardVersion retorna a versão atual de dados da Dashboard da conta.
func (qc *QueryCache) DashboardVersion(ctx context.Context, accountID int64) string {
	if qc == nil || qc.rdb == nil {
		return "0"
	}
	val, err := qc.rdb.Get(ctx, dashVersionKey(accountID)).Result()
	if err != nil || val == "" {
		return "0"
	}
	return val
}

// GetOrLoadWithNegCache adiciona negative-cache curto para falhas de APIs externas (Meta/Google Ads),
// evitando travar múltiplas requisições simultâneas quando a API externa oscila.
func (qc *QueryCache) GetOrLoadWithNegCache(
	ctx context.Context,
	key string,
	ttl, negTTL time.Duration,
	fallbackJSON []byte,
	loader func(context.Context) ([]byte, error),
) ([]byte, error) {
	negKey := key + ":neg"
	if qc != nil && qc.rdb != nil && key != "" {
		if cached, err := qc.rdb.Get(ctx, key).Bytes(); err == nil && len(cached) > 0 {
			return cached, nil
		}
		if negTTL > 0 && len(fallbackJSON) > 0 {
			if exists, err := qc.rdb.Exists(ctx, negKey).Result(); err == nil && exists > 0 {
				return fallbackJSON, nil
			}
		}
	}
	raw, err := qc.GetOrLoadBytes(ctx, key, ttl, loader)
	if err != nil && qc != nil && qc.rdb != nil && negTTL > 0 && len(fallbackJSON) > 0 {
		_ = qc.rdb.Set(ctx, negKey, "1", negTTL).Err()
	}
	return raw, err
}

