package cache

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const oauthStateTTL = 10 * time.Minute

type OAuthStateEntry struct {
	GoogleAdsAccountID int64     `json:"google_ads_account_id"`
	MetaAccountID      int64     `json:"meta_account_id"`
	AccountID          int64     `json:"account_id"`
	UserID             int64     `json:"user_id"`
	ExpiresAt          time.Time `json:"expires_at"`
}

// OAuthStateStore persiste o nonce OAuth do Google Ads no Redis (GETDEL atômico),
// mantendo fallback em memória caso o Redis esteja temporariamente indisponível.
type OAuthStateStore struct {
	rdb     *redis.Client
	mu      sync.Mutex
	local   map[string]OAuthStateEntry
}

func NewOAuthStateStore(rdb *redis.Client) *OAuthStateStore {
	return &OAuthStateStore{
		rdb:   rdb,
		local: make(map[string]OAuthStateEntry),
	}
}

func oauthStateKey(nonce string) string {
	return "gads:oauth:state:" + strings.TrimSpace(nonce)
}

func (s *OAuthStateStore) Put(ctx context.Context, nonce string, entry OAuthStateEntry) {
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return
	}
	if entry.ExpiresAt.IsZero() {
		entry.ExpiresAt = time.Now().Add(oauthStateTTL)
	}
	if s.rdb != nil {
		if raw, err := json.Marshal(entry); err == nil {
			if err := s.rdb.Set(ctx, oauthStateKey(nonce), raw, oauthStateTTL).Err(); err == nil {
				return
			}
		}
	}
	s.mu.Lock()
	s.pruneLocked()
	s.local[nonce] = entry
	s.mu.Unlock()
}

func (s *OAuthStateStore) Take(ctx context.Context, nonce string) (OAuthStateEntry, bool) {
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return OAuthStateEntry{}, false
	}
	if s.rdb != nil {
		raw, err := s.rdb.GetDel(ctx, oauthStateKey(nonce)).Bytes()
		if err == nil && len(raw) > 0 {
			var entry OAuthStateEntry
			if json.Unmarshal(raw, &entry) == nil && time.Now().Before(entry.ExpiresAt) {
				return entry, true
			}
			return OAuthStateEntry{}, false
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.local[nonce]
	if !ok {
		return OAuthStateEntry{}, false
	}
	delete(s.local, nonce)
	if time.Now().After(entry.ExpiresAt) {
		return OAuthStateEntry{}, false
	}
	return entry, true
}

func (s *OAuthStateStore) pruneLocked() {
	now := time.Now()
	for k, v := range s.local {
		if now.After(v.ExpiresAt) {
			delete(s.local, k)
		}
	}
}

