package googleads

import (
	"sync"
	"time"
)

type StateEntry struct {
	GoogleAdsAccountID int64
	AccountID          int64
	UserID             int64
	ExpiresAt          time.Time
}

type StateStore struct {
	mu      sync.Mutex
	entries map[string]StateEntry
}

func NewStateStore() *StateStore {
	s := &StateStore{entries: map[string]StateEntry{}}
	go func() {
		t := time.NewTicker(2 * time.Minute)
		defer t.Stop()
		for range t.C {
			s.gc()
		}
	}()
	return s
}

func (s *StateStore) Put(state string, e StateEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ExpiresAt.IsZero() {
		e.ExpiresAt = time.Now().Add(5 * time.Minute)
	}
	s.entries[state] = e
}

func (s *StateStore) Take(state string) (StateEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[state]
	if !ok {
		return StateEntry{}, false
	}
	delete(s.entries, state)
	if time.Now().After(e.ExpiresAt) {
		return StateEntry{}, false
	}
	return e, true
}

func (s *StateStore) gc() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, e := range s.entries {
		if now.After(e.ExpiresAt) {
			delete(s.entries, k)
		}
	}
}

