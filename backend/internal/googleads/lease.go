package googleads

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const maxClientGoogleLeaseTTL = 55 * time.Minute

var errGoogleLeaseUnsupported = errors.New("google central lease unsupported")

type googleLeaseEntry struct {
	accessToken string
	expiresAt   time.Time
}

type googleLeasePayload struct {
	AccessToken string `json:"atok"`
	LeaseTTLSec int64  `json:"ttl"`
}

var (
	dataManagerIngestURL = "https://datamanager.googleapis.com/v1/events:ingest"
	gadsDirectTransport  http.RoundTripper
	pooledGadsTransport  = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   8 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        128,
		MaxIdleConnsPerHost: 64,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 8 * time.Second,
	}

	gadsLeaseMu    sync.RWMutex
	gadsLeaseCache = make(map[string]googleLeaseEntry)
	gadsLeaseSF    singleflight.Group
)

// InvalidateAllLeases limpa imediatamente todos os leases OAuth do Google Ads em memória RAM.
func InvalidateAllLeases() {
	gadsLeaseMu.Lock()
	clear(gadsLeaseCache)
	gadsLeaseMu.Unlock()
}

func invalidateGoogleLease(vaultToken string) {
	vt := strings.TrimSpace(vaultToken)
	if vt == "" {
		return
	}
	gadsLeaseMu.Lock()
	delete(gadsLeaseCache, vt)
	gadsLeaseMu.Unlock()
}

func (c *Client) acquireGoogleLease(ctx context.Context, account *domain.GoogleAdsAccount) (string, error) {
	if util.IsShardHalted() {
		return "", errors.New("acesso da instância suspenso")
	}
	vt := strings.TrimSpace(account.RefreshToken)
	if vt == "" {
		return "", errors.New("conta Google Ads sem conexão OAuth ativa")
	}
	now := time.Now()
	gadsLeaseMu.RLock()
	cached, ok := gadsLeaseCache[vt]
	gadsLeaseMu.RUnlock()
	if ok && now.Before(cached.expiresAt) {
		return cached.accessToken, nil
	}

	ch := gadsLeaseSF.DoChan(vt, func() (any, error) {
		lctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var pld googleLeasePayload
		if err := c.oauth.CallCentralProxy(lctx, "issue_lease", vt, account.CustomerID, account.LoginCustomerID, nil, &pld); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unknown action") {
				return "", errGoogleLeaseUnsupported
			}
			return "", err
		}
		if pld.AccessToken == "" {
			return "", errGoogleLeaseUnsupported
		}
		ttl := time.Duration(pld.LeaseTTLSec) * time.Second
		if ttl <= 0 || ttl > maxClientGoogleLeaseTTL {
			ttl = 45 * time.Minute
		}
		gadsLeaseMu.Lock()
		gadsLeaseCache[vt] = googleLeaseEntry{
			accessToken: pld.AccessToken,
			expiresAt:   time.Now().Add(ttl),
		}
		gadsLeaseMu.Unlock()
		return pld.AccessToken, nil
	})
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return "", res.Err
		}
		return res.Val.(string), nil
	}
}

func directPostDataManager(ctx context.Context, accessToken string, rawPayload []byte) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dataManagerIngestURL, bytes.NewReader(rawPayload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	tr := http.RoundTripper(pooledGadsTransport)
	if gadsDirectTransport != nil {
		tr = gadsDirectTransport
	}
	cli := &http.Client{Timeout: 15 * time.Second, Transport: tr}
	resp, err := cli.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

func (c *Client) sendConversionWithLease(ctx context.Context, account *domain.GoogleAdsAccount, rawPayload []byte) (int, string, error) {
	tok, err := c.acquireGoogleLease(ctx, account)
	if err != nil {
		if errors.Is(err, errGoogleLeaseUnsupported) {
			var fb struct {
				StatusCode int    `json:"status_code"`
				Body       string `json:"body"`
			}
			fErr := c.oauth.CallCentralProxy(ctx, "send_conversion", account.RefreshToken, account.CustomerID, account.LoginCustomerID, rawPayload, &fb)
			return fb.StatusCode, fb.Body, fErr
		}
		return 0, "", err
	}
	code, body, err := directPostDataManager(ctx, tok, rawPayload)
	if err != nil {
		return 0, "", err
	}
	if code == http.StatusUnauthorized {
		invalidateGoogleLease(account.RefreshToken)
		if freshTok, lErr := c.acquireGoogleLease(ctx, account); lErr == nil && freshTok != tok {
			return directPostDataManager(ctx, freshTok, rawPayload)
		}
	}
	return code, body, nil
}
