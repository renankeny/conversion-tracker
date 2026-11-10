package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	maxSessionRevocationTTL = 31 * 24 * time.Hour
	tenantAccessCacheTTL    = 5 * time.Minute
)

// SessionStore gerencia denylist de JTI (logout explícito), corte temporal por usuário
// (troca de senha/role/exclusão) e cache curto de acesso multi-tenant (X-Account-ID).
type SessionStore struct {
	rdb          *redis.Client
	mu           sync.RWMutex
	deniedJTI    map[string]time.Time
	revokedAfter map[int64]int64
}

func NewSessionStore(rdb *redis.Client) *SessionStore {
	return &SessionStore{
		rdb:          rdb,
		deniedJTI:    make(map[string]time.Time),
		revokedAfter: make(map[int64]int64),
	}
}

func authDenyKey(jti string) string {
	return "auth:deny:" + strings.TrimSpace(jti)
}

func authRevokedBeforeKey(userID int64) string {
	return fmt.Sprintf("auth:revoked_before:%d", userID)
}

func tenantAccessKey(userID, accountID int64) string {
	return fmt.Sprintf("tenant:access:%d:%d", userID, accountID)
}

func (s *SessionStore) RevokeToken(ctx context.Context, jti string, ttl time.Duration) {
	if s == nil || strings.TrimSpace(jti) == "" {
		return
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if ttl > maxSessionRevocationTTL {
		ttl = maxSessionRevocationTTL
	}
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, authDenyKey(jti), "1", ttl).Err(); err == nil {
			return
		}
	}
	s.mu.Lock()
	s.deniedJTI[jti] = time.Now().Add(ttl)
	s.mu.Unlock()
}

func (s *SessionStore) RevokeUserSessions(ctx context.Context, userID int64) {
	if s == nil || userID <= 0 {
		return
	}
	cutoff := time.Now().Unix()
	if s.rdb != nil {
		_ = s.rdb.Set(ctx, authRevokedBeforeKey(userID), strconv.FormatInt(cutoff, 10), maxSessionRevocationTTL).Err()
	}
	s.mu.Lock()
	s.revokedAfter[userID] = cutoff
	s.mu.Unlock()
}

func (s *SessionStore) IsRevoked(ctx context.Context, jti string, userID int64, issuedAtUnix int64) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	if exp, ok := s.deniedJTI[jti]; ok && time.Now().Before(exp) {
		s.mu.RUnlock()
		return true
	}
	if cutoff, ok := s.revokedAfter[userID]; ok && issuedAtUnix > 0 && issuedAtUnix <= cutoff {
		s.mu.RUnlock()
		return true
	}
	s.mu.RUnlock()

	if s.rdb == nil {
		return false
	}
	pipe := s.rdb.Pipeline()
	var denyCmd *redis.IntCmd
	if jti != "" {
		denyCmd = pipe.Exists(ctx, authDenyKey(jti))
	}
	var revCmd *redis.StringCmd
	if userID > 0 {
		revCmd = pipe.Get(ctx, authRevokedBeforeKey(userID))
	}
	_, _ = pipe.Exec(ctx)

	if denyCmd != nil && denyCmd.Val() > 0 {
		return true
	}
	if revCmd != nil {
		if raw, err := revCmd.Result(); err == nil && raw != "" {
			if cutoff, perr := strconv.ParseInt(raw, 10, 64); perr == nil && issuedAtUnix > 0 && issuedAtUnix <= cutoff {
				return true
			}
		}
	}
	return false
}

func (s *SessionStore) GetTenantAccess(ctx context.Context, userID, accountID int64) (bool, bool) {
	if s == nil || s.rdb == nil || userID <= 0 || accountID <= 0 {
		return false, false
	}
	val, err := s.rdb.Get(ctx, tenantAccessKey(userID, accountID)).Result()
	if err != nil {
		return false, false
	}
	return val == "1", true
}

func (s *SessionStore) SetTenantAccess(ctx context.Context, userID, accountID int64, allowed bool) {
	if s == nil || s.rdb == nil || userID <= 0 || accountID <= 0 {
		return
	}
	v := "0"
	if allowed {
		v = "1"
	}
	_ = s.rdb.Set(ctx, tenantAccessKey(userID, accountID), v, tenantAccessCacheTTL).Err()
}

