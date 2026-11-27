package cache

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	shardStateKey = "tf:shard:state"
	shardStateTTL = 24 * time.Hour
)

// PersistShardState salva o estado de bloqueio da instância no Redis (além do Pub/Sub),
// garantindo que o Worker permaneça sincronizado mesmo após reinícios ou quedas momentâneas.
func PersistShardState(ctx context.Context, rdb *redis.Client, halted bool, mask byte) {
	if rdb == nil {
		return
	}
	flag := "0"
	if halted {
		flag = "1"
	}
	val := flag + ":" + strconv.Itoa(int(mask))
	_ = rdb.Set(ctx, shardStateKey, val, shardStateTTL).Err()
	PublishInvalidate(ctx, rdb, InvalidateEvent{
		Kind: "shard_state",
		Keys: []string{flag, strconv.Itoa(int(mask))},
	})
}

// LoadShardState lê o último estado de licença gravado pela API no Redis.
func LoadShardState(ctx context.Context, rdb *redis.Client) (halted bool, mask byte, ok bool) {
	if rdb == nil {
		return false, 0, false
	}
	raw, err := rdb.Get(ctx, shardStateKey).Result()
	if err != nil || raw == "" {
		return false, 0, false
	}
	flag, maskStr, found := strings.Cut(raw, ":")
	if !found {
		return false, 0, false
	}
	m, _ := strconv.Atoi(maskStr)
	return flag == "1", byte(m), true
}

// StartWorkerShardStateSync reconcilia o estado de licença no boot do Worker e a cada 60s via Redis.
func StartWorkerShardStateSync(ctx context.Context, rdb *redis.Client, apply func(halted bool, mask byte)) {
	if rdb == nil || apply == nil {
		return
	}
	syncOnce := func() {
		rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if halted, mask, ok := LoadShardState(rctx, rdb); ok {
			apply(halted, mask)
		}
	}
	syncOnce()
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				syncOnce()
			}
		}
	}()
}
