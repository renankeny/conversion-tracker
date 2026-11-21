package metaoauth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

var errLeaseUnsupported = errors.New("central lease unsupported")

const maxClientMetaLeaseTTL = 6 * time.Hour

type metaLeaseEntry struct {
	accessToken    string
	appSecretProof string
	graphVersion   string
	expiresAt      time.Time
}

type metaLeasePayload struct {
	AccessToken    string `json:"atok"`
	AppSecretProof string `json:"asp"`
	GraphVersion   string `json:"gver"`
	LeaseTTLSec    int64  `json:"ttl"`
	TokenExpiresAt int64  `json:"texp,omitempty"`
}

var (
	metaLeaseMu    sync.RWMutex
	metaLeaseCache = make(map[string]metaLeaseEntry)
	metaLeaseSF    singleflight.Group
)

// InvalidateAllLeases limpa imediatamente todos os leases OAuth da Meta em memória RAM.
func InvalidateAllLeases() {
	metaLeaseMu.Lock()
	clear(metaLeaseCache)
	metaLeaseMu.Unlock()
}

func invalidateMetaLease(vaultToken string) {
	vt := strings.TrimSpace(vaultToken)
	if vt == "" {
		return
	}
	metaLeaseMu.Lock()
	delete(metaLeaseCache, vt)
	metaLeaseMu.Unlock()
}

func (c *Client) acquireLease(ctx context.Context, acc *domain.MetaAccount) (metaLeaseEntry, error) {
	if util.IsShardHalted() {
		return metaLeaseEntry{}, &ProxyError{StatusCode: http.StatusLocked, Message: "acesso da instância suspenso"}
	}
	vaultToken := ""
	if acc != nil {
		vaultToken = strings.TrimSpace(acc.VaultTokenEncrypted)
	}
	if vaultToken == "" {
		return metaLeaseEntry{}, &ProxyError{StatusCode: http.StatusUnauthorized, Message: "conta Meta Ads sem conexão OAuth ativa"}
	}

	now := time.Now()
	metaLeaseMu.RLock()
	cached, ok := metaLeaseCache[vaultToken]
	metaLeaseMu.RUnlock()
	if ok && now.Before(cached.expiresAt) {
		return cached, nil
	}

	ch := metaLeaseSF.DoChan(vaultToken, func() (any, error) {
		lctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var pld metaLeasePayload
		if err := c.callProxy(lctx, acc, "issue_lease", "", "", "", nil, &pld); err != nil {
			var pErr *ProxyError
			if errors.As(err, &pErr) && pErr.StatusCode == http.StatusBadRequest && strings.Contains(strings.ToLower(pErr.Message), "unknown") {
				return metaLeaseEntry{}, errLeaseUnsupported
			}
			return metaLeaseEntry{}, err
		}
		if pld.AccessToken == "" || pld.AppSecretProof == "" {
			return metaLeaseEntry{}, errLeaseUnsupported
		}
		if pld.TokenExpiresAt > 0 {
			c.syncTokenExpiry(lctx, acc, time.Unix(pld.TokenExpiresAt, 0), false)
		}
		gver := strings.TrimSpace(pld.GraphVersion)
		if gver == "" {
			gver = "v26.0"
		}
		ttl := time.Duration(pld.LeaseTTLSec) * time.Second
		if ttl <= 0 || ttl > maxClientMetaLeaseTTL {
			ttl = maxClientMetaLeaseTTL
		}
		entry := metaLeaseEntry{
			accessToken:    pld.AccessToken,
			appSecretProof: pld.AppSecretProof,
			graphVersion:   gver,
			expiresAt:      time.Now().Add(ttl),
		}
		metaLeaseMu.Lock()
		if len(metaLeaseCache) >= 512 {
			tNow := time.Now()
			for k, v := range metaLeaseCache {
				if tNow.After(v.expiresAt) {
					delete(metaLeaseCache, k)
				}
			}
		}
		metaLeaseCache[vaultToken] = entry
		metaLeaseMu.Unlock()
		return entry, nil
	})
	select {
	case <-ctx.Done():
		return metaLeaseEntry{}, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return metaLeaseEntry{}, res.Err
		}
		return res.Val.(metaLeaseEntry), nil
	}
}

func runWithMetaLease[T any](
	ctx context.Context,
	c *Client,
	acc *domain.MetaAccount,
	fn func(metaLeaseEntry) (T, int, error),
	legacyFallback func() (T, error),
) (T, error) {
	var zero T
	lease, err := c.acquireLease(ctx, acc)
	if err != nil {
		if errors.Is(err, errLeaseUnsupported) && legacyFallback != nil {
			return legacyFallback()
		}
		return zero, err
	}
	res, status, callErr := fn(lease)
	if callErr == nil {
		return res, nil
	}
	if status == http.StatusUnauthorized && acc != nil {
		invalidateMetaLease(acc.VaultTokenEncrypted)
		if fresh, lErr := c.acquireLease(ctx, acc); lErr == nil && fresh.accessToken != lease.accessToken {
			res2, status2, callErr2 := fn(fresh)
			if callErr2 == nil {
				return res2, nil
			}
			status, callErr = status2, callErr2
		}
		if status == http.StatusUnauthorized {
			c.syncTokenExpiry(ctx, acc, time.Now().Add(-time.Second), true)
		}
	}
	return res, callErr
}
