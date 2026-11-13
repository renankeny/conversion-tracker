package googleads

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
)

type StateEntry struct {
	GoogleAdsAccountID int64
	AccountID          int64
	UserID             int64
	ExpiresAt          time.Time
}

// StateStore delega o armazenamento de state OAuth para cache.OAuthStateStore (Redis GETDEL + fallback local),
// sem goroutine em background vazando.
type StateStore struct {
	inner *cache.OAuthStateStore
}

func NewStateStore(rdb ...*redis.Client) *StateStore {
	var client *redis.Client
	if len(rdb) > 0 {
		client = rdb[0]
	}
	return &StateStore{inner: cache.NewOAuthStateStore(client)}
}

func (s *StateStore) Put(ctx context.Context, state string, e StateEntry) {
	s.inner.Put(ctx, state, cache.OAuthStateEntry{
		GoogleAdsAccountID: e.GoogleAdsAccountID,
		AccountID:          e.AccountID,
		UserID:             e.UserID,
		ExpiresAt:          e.ExpiresAt,
	})
}

func (s *StateStore) Take(ctx context.Context, state string) (StateEntry, bool) {
	entry, ok := s.inner.Take(ctx, state)
	if !ok {
		return StateEntry{}, false
	}
	return StateEntry{
		GoogleAdsAccountID: entry.GoogleAdsAccountID,
		AccountID:          entry.AccountID,
		UserID:             entry.UserID,
		ExpiresAt:          entry.ExpiresAt,
	}, true
}
