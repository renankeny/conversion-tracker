package metaoauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type Client struct {
	cfg Config
}

func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg}
}

type AdMetadata struct {
	AdID         string `json:"ad_id"`
	AdName       string `json:"ad_name"`
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	AdsetID      string `json:"adset_id"`
	AdsetName    string `json:"adset_name"`
	PageID       string `json:"page_id"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type ProxyError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *ProxyError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("gateway meta ads status %d", e.StatusCode)
}

func (c *Client) syncTokenExpiry(ctx context.Context, acc *domain.MetaAccount, newExp time.Time, force bool) {
	if acc == nil || acc.ID <= 0 || c.cfg.Pool == nil {
		return
	}
	if !force && acc.TokenExpiresAt != nil && !acc.TokenExpiresAt.IsZero() {
		diff := newExp.Sub(*acc.TokenExpiresAt)
		if diff > -time.Hour && diff < time.Hour {
			return
		}
	}
	acc.TokenExpiresAt = &newExp
	_, _ = c.cfg.Pool.Exec(ctx, `
		UPDATE meta_accounts SET token_expires_at = $2, updated_at = NOW() WHERE id = $1
	`, acc.ID, newExp)
}

func (c *Client) callProxy(ctx context.Context, acc *domain.MetaAccount, action, adAccountID, targetID, timeRange string, payload any, out any) error {
	vaultToken := ""
	if acc != nil {
		vaultToken = strings.TrimSpace(acc.VaultTokenEncrypted)
	}
	if vaultToken == "" {
		return &ProxyError{StatusCode: http.StatusUnauthorized, Message: "conta Meta Ads sem conexão OAuth ativa"}
	}
	hostname, _ := os.Hostname()
	var rawPld json.RawMessage
	if payload != nil {
		if b, ok := payload.([]byte); ok {
			rawPld = b
		} else {
			rawPld, _ = json.Marshal(payload)
		}
	}
	reqMap := map[string]any{
		"idn": c.cfg.primaryIdentity(ctx),
		"dom": c.cfg.cleanDomain(),
		"hst": hostname,
		"uid": util.ActiveInstanceUID(),
		"act": action,
		"vtk": vaultToken,
		"aid": adAccountID,
		"tid": targetID,
		"trg": timeRange,
		"pld": rawPld,
	}
	sealed, err := sealGatewayJSON(reqMap)
	if err != nil {
		return err
	}
	bodyBytes, _ := json.Marshal(map[string]string{"v": sealed})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayBaseURL()+"/api/v1/meta-ads/proxy", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.cfg.httpClient(20 * time.Second).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if expHdr := strings.TrimSpace(resp.Header.Get("X-Meta-Token-Expires-At")); expHdr != "" {
		if expUnix, pErr := strconv.ParseInt(expHdr, 10, 64); pErr == nil && expUnix > 0 {
			c.syncTokenExpiry(ctx, acc, time.Unix(expUnix, 0), false)
		}
	}

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			c.syncTokenExpiry(ctx, acc, time.Now().Add(-time.Second), true)
		}
		var errEnv struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errEnv)
		return &ProxyError{StatusCode: resp.StatusCode, Body: string(raw), Message: errEnv.Error}
	}
	var env struct {
		Vector string `json:"v"`
	}
	if err := json.Unmarshal(raw, &env); err != nil || env.Vector == "" {
		return errors.New("pacote de resposta invalido")
	}
	return openGatewayJSON(env.Vector, out)
}

func (c *Client) ListAdAccounts(ctx context.Context, acc *domain.MetaAccount) (map[string]any, error) {
	var out map[string]any
	err := c.callProxy(ctx, acc, "list_ad_accounts", "", "", "", nil, &out)
	return out, err
}

func (c *Client) ListPages(ctx context.Context, acc *domain.MetaAccount) (map[string]any, error) {
	var out map[string]any
	err := c.callProxy(ctx, acc, "list_pages", "", "", "", nil, &out)
	return out, err
}

func (c *Client) ListPixels(ctx context.Context, acc *domain.MetaAccount, adAccountID string) (map[string]any, error) {
	if adAccountID == "" && acc != nil {
		adAccountID = acc.AdAccountID
	}
	var out map[string]any
	err := c.callProxy(ctx, acc, "list_pixels", adAccountID, "", "", nil, &out)
	return out, err
}

func (c *Client) GetAdMetadata(ctx context.Context, acc *domain.MetaAccount, adID string) (AdMetadata, error) {
	adAccID := ""
	if acc != nil {
		adAccID = acc.AdAccountID
	}
	return runWithMetaLease(ctx, c, acc, func(lease metaLeaseEntry) (AdMetadata, int, error) {
		return directGetAdMetadata(ctx, lease, adID)
	}, func() (AdMetadata, error) {
		var out AdMetadata
		err := c.callProxy(ctx, acc, "get_ad_metadata", adAccID, adID, "", nil, &out)
		return out, err
	})
}

func (c *Client) FetchLiveSpend(ctx context.Context, acc *domain.MetaAccount, adAccountID, timeRange string) (float64, error) {
	res, err := c.FetchLiveInsights(ctx, acc, adAccountID, timeRange)
	return res.Spend, err
}

func (c *Client) FetchLiveInsights(ctx context.Context, acc *domain.MetaAccount, adAccountID, timeRange string) (meta.LiveInsightsSummary, error) {
	if adAccountID == "" && acc != nil {
		adAccountID = acc.AdAccountID
	}
	out, err := runWithMetaLease(ctx, c, acc, func(lease metaLeaseEntry) (meta.LiveInsightsSummary, int, error) {
		return directFetchLiveInsights(ctx, lease, adAccountID, timeRange)
	}, func() (meta.LiveInsightsSummary, error) {
		var fb meta.LiveInsightsSummary
		err := c.callProxy(ctx, acc, "live_spend", adAccountID, "", timeRange, nil, &fb)
		return fb, err
	})
	out.Connected = err == nil
	return out, err
}

func (c *Client) FetchCreativeRanking(ctx context.Context, acc *domain.MetaAccount, adAccountID, timeRange string) ([]meta.CreativeInsight, error) {
	if adAccountID == "" && acc != nil {
		adAccountID = acc.AdAccountID
	}
	return runWithMetaLease(ctx, c, acc, func(lease metaLeaseEntry) ([]meta.CreativeInsight, int, error) {
		return directFetchCreativeRanking(ctx, lease, adAccountID, timeRange)
	}, func() ([]meta.CreativeInsight, error) {
		var fb struct {
			Creatives []meta.CreativeInsight `json:"creatives"`
		}
		err := c.callProxy(ctx, acc, "creative_ranking", adAccountID, "", timeRange, nil, &fb)
		return fb.Creatives, err
	})
}

func (c *Client) SendCAPIEvent(ctx context.Context, acc *domain.MetaAccount, pixelID string, rawPayload []byte) (int, string, error) {
	adAccID := ""
	if acc != nil {
		adAccID = acc.AdAccountID
	}
	out, err := runWithMetaLease(ctx, c, acc, func(lease metaLeaseEntry) (capiOutcome, int, error) {
		return directSendCAPIEvent(ctx, lease, pixelID, rawPayload)
	}, func() (capiOutcome, error) {
		var fb capiOutcome
		err := c.callProxy(ctx, acc, "send_capi_event", adAccID, pixelID, "", rawPayload, &fb)
		return fb, err
	})
	if err != nil {
		var pErr *ProxyError
		if errors.As(err, &pErr) && out.StatusCode == 0 {
			return pErr.StatusCode, pErr.Body, err
		}
	}
	return out.StatusCode, out.Body, err
}
