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
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
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

func (c *Client) callProxy(ctx context.Context, action, vaultToken, adAccountID, targetID, timeRange string, payload any, out any) error {
	if vaultToken == "" {
		return errors.New("conta Meta Ads sem conexão OAuth ativa")
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
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var errEnv struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errEnv)
		if errEnv.Error != "" {
			return errors.New(errEnv.Error)
		}
		return fmt.Errorf("gateway meta ads status %d", resp.StatusCode)
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
	err := c.callProxy(ctx, "list_ad_accounts", acc.VaultTokenEncrypted, "", "", "", nil, &out)
	return out, err
}

func (c *Client) ListPages(ctx context.Context, acc *domain.MetaAccount) (map[string]any, error) {
	var out map[string]any
	err := c.callProxy(ctx, "list_pages", acc.VaultTokenEncrypted, "", "", "", nil, &out)
	return out, err
}

func (c *Client) ListPixels(ctx context.Context, acc *domain.MetaAccount, adAccountID string) (map[string]any, error) {
	if adAccountID == "" {
		adAccountID = acc.AdAccountID
	}
	var out map[string]any
	err := c.callProxy(ctx, "list_pixels", acc.VaultTokenEncrypted, adAccountID, "", "", nil, &out)
	return out, err
}

func (c *Client) GetAdMetadata(ctx context.Context, acc *domain.MetaAccount, adID string) (AdMetadata, error) {
	var out AdMetadata
	err := c.callProxy(ctx, "get_ad_metadata", acc.VaultTokenEncrypted, acc.AdAccountID, adID, "", nil, &out)
	return out, err
}

func (c *Client) FetchLiveSpend(ctx context.Context, acc *domain.MetaAccount, adAccountID, timeRange string) (float64, error) {
	res, err := c.FetchLiveInsights(ctx, acc, adAccountID, timeRange)
	return res.Spend, err
}

func (c *Client) FetchLiveInsights(ctx context.Context, acc *domain.MetaAccount, adAccountID, timeRange string) (meta.LiveInsightsSummary, error) {
	if adAccountID == "" {
		adAccountID = acc.AdAccountID
	}
	var out meta.LiveInsightsSummary
	err := c.callProxy(ctx, "live_spend", acc.VaultTokenEncrypted, adAccountID, "", timeRange, nil, &out)
	out.Connected = err == nil
	return out, err
}

func (c *Client) FetchCreativeRanking(ctx context.Context, acc *domain.MetaAccount, adAccountID, timeRange string) ([]meta.CreativeInsight, error) {
	if adAccountID == "" {
		adAccountID = acc.AdAccountID
	}
	var out struct {
		Creatives []meta.CreativeInsight `json:"creatives"`
	}
	err := c.callProxy(ctx, "creative_ranking", acc.VaultTokenEncrypted, adAccountID, "", timeRange, nil, &out)
	return out.Creatives, err
}

func (c *Client) SendCAPIEvent(ctx context.Context, acc *domain.MetaAccount, pixelID string, rawPayload []byte) (int, string, error) {
	var out struct {
		StatusCode int    `json:"status_code"`
		Body       string `json:"body"`
	}
	err := c.callProxy(ctx, "send_capi_event", acc.VaultTokenEncrypted, acc.AdAccountID, pixelID, "", rawPayload, &out)
	return out.StatusCode, out.Body, err
}
