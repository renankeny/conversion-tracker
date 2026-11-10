package cache

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const wsTicketTTL = 30 * time.Second

type WSTicket struct {
	UserID    int64     `json:"user_id"`
	AccountID int64     `json:"account_id"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

// WSTicketStore emite e consome tickets efêmeros de uso único (30s) para upgrade WebSocket,
// evitando expor o JWT de longa duração em query strings (?token=) e logs de proxy.
type WSTicketStore struct {
	rdb   *redis.Client
	mu    sync.Mutex
	local map[string]WSTicket
}

func NewWSTicketStore(rdb *redis.Client) *WSTicketStore {
	return &WSTicketStore{
		rdb:   rdb,
		local: make(map[string]WSTicket),
	}
}

func wsTicketKey(id string) string {
	return "ws:ticket:" + strings.TrimSpace(id)
}

func (s *WSTicketStore) Issue(ctx context.Context, userID, accountID int64, role string) (string, error) {
	id := uuid.NewString()
	t := WSTicket{
		UserID:    userID,
		AccountID: accountID,
		Role:      role,
		ExpiresAt: time.Now().Add(wsTicketTTL),
	}
	if s.rdb != nil {
		if raw, err := json.Marshal(t); err == nil {
			if err := s.rdb.Set(ctx, wsTicketKey(id), raw, wsTicketTTL).Err(); err == nil {
				return id, nil
			}
		}
	}
	s.mu.Lock()
	now := time.Now()
	for k, v := range s.local {
		if now.After(v.ExpiresAt) {
			delete(s.local, k)
		}
	}
	s.local[id] = t
	s.mu.Unlock()
	return id, nil
}

func (s *WSTicketStore) Consume(ctx context.Context, id string) (WSTicket, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return WSTicket{}, false
	}
	if s.rdb != nil {
		raw, err := s.rdb.GetDel(ctx, wsTicketKey(id)).Bytes()
		if err == nil && len(raw) > 0 {
			var t WSTicket
			if json.Unmarshal(raw, &t) == nil && time.Now().Before(t.ExpiresAt) {
				return t, true
			}
			return WSTicket{}, false
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.local[id]
	if !ok {
		return WSTicket{}, false
	}
	delete(s.local, id)
	if time.Now().After(t.ExpiresAt) {
		return WSTicket{}, false
	}
	return t, true
}

