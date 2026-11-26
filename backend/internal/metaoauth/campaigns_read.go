package metaoauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

const metaCampaignsGraphVersion = "v26.0"

// MetaHierarchyBundle contém campanhas, conjuntos e anúncios carregados da Graph API v26.0.
type MetaHierarchyBundle struct {
	Currency  string
	Campaigns []domain.CampaignRow
	AdSets    []domain.CampaignRow
	Ads       []domain.CampaignRow
}

// FetchHierarchyInsights busca em paralelo estrutura + insights (campanhas, conjuntos e anúncios)
// na Meta Graph API v26.0 usando o lease OAuth direto (ou token permanente quando fornecido).
func (c *Client) FetchHierarchyInsights(
	ctx context.Context,
	acc *domain.MetaAccount,
	adAccountID, permanentToken, since, until string,
) (MetaHierarchyBundle, error) {
	act := normalizeMetaActID(adAccountID)
	if act == "" && acc != nil {
		act = normalizeMetaActID(acc.AdAccountID)
	}
	if act == "" || act == "act_" {
		return MetaHierarchyBundle{}, errors.New("ad_account_id Meta ausente")
	}
	timeRange := fmt.Sprintf(`{"since":"%s","until":"%s"}`, since, until)

	var accRefID int64
	if acc != nil {
		accRefID = acc.ID
	}

	if acc != nil && strings.TrimSpace(acc.VaultTokenEncrypted) != "" {
		return runWithMetaLease(ctx, c, acc, func(lease metaLeaseEntry) (MetaHierarchyBundle, int, error) {
			lease.graphVersion = metaCampaignsGraphVersion
			return directFetchHierarchyV26(ctx, lease, accRefID, act, timeRange)
		}, nil)
	}
	if tok := strings.TrimSpace(permanentToken); tok != "" {
		lease := metaLeaseEntry{accessToken: tok, graphVersion: metaCampaignsGraphVersion}
		bundle, _, err := directFetchHierarchyV26(ctx, lease, accRefID, act, timeRange)
		return bundle, err
	}
	return MetaHierarchyBundle{}, errors.New("conta Meta Ads sem conexão OAuth ou token permanente")
}

func directGraphGETPaged(ctx context.Context, lease metaLeaseEntry, relPath string, params url.Values, maxPages int) ([]map[string]any, int, error) {
	if maxPages <= 0 {
		maxPages = 4
	}
	raw, code, err := directGraphGETOptionalProof(ctx, lease, relPath, params)
	if err != nil {
		return nil, code, err
	}
	var all []map[string]any
	for page := 0; page < maxPages; page++ {
		var resp struct {
			Data   []map[string]any `json:"data"`
			Paging struct {
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return all, http.StatusOK, nil
		}
		all = append(all, resp.Data...)
		if resp.Paging.Next == "" || page+1 >= maxPages {
			break
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, resp.Paging.Next, nil)
		if err != nil {
			break
		}
		httpResp, err := newDirectMetaHTTPClient(15 * time.Second).Do(req)
		if err != nil {
			break
		}
		raw, _ = io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if httpResp.StatusCode >= 300 {
			break
		}
	}
	return all, http.StatusOK, nil
}

func directFetchHierarchyV26(
	ctx context.Context,
	lease metaLeaseEntry,
	accRefID int64,
	act, timeRange string,
) (MetaHierarchyBundle, int, error) {
	var (
		currency                            = "BRL"
		campRaw, adsetRaw, adRaw            []map[string]any
		campInsRaw, adsetInsRaw, adInsRaw   []map[string]any
		firstErr                            error
		firstCode                           int
		errMu                               sync.Mutex
		wg                                  sync.WaitGroup
	)
	recordErr := func(code int, err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr, firstCode = err, code
		}
		errMu.Unlock()
	}

	wg.Add(7)
	go func() {
		defer wg.Done()
		if raw, _, err := directGraphGETOptionalProof(ctx, lease, act, url.Values{"fields": {"currency"}}); err == nil {
			var info struct {
				Currency string `json:"currency"`
			}
			if json.Unmarshal(raw, &info) == nil && info.Currency != "" {
				currency = strings.ToUpper(strings.TrimSpace(info.Currency))
			}
		}
	}()
	go func() {
		defer wg.Done()
		rows, code, err := directGraphGETPaged(ctx, lease, act+"/campaigns", url.Values{
			"fields": {"id,name,status,effective_status,objective,buying_type,daily_budget,lifetime_budget"},
			"limit":  {"250"},
		}, 4)
		recordErr(code, err)
		campRaw = rows
	}()
	go func() {
		defer wg.Done()
		rows, code, err := directGraphGETPaged(ctx, lease, act+"/adsets", url.Values{
			"fields": {"id,name,campaign_id,status,effective_status,optimization_goal,daily_budget,lifetime_budget"},
			"limit":  {"500"},
		}, 4)
		if code == http.StatusUnauthorized {
			recordErr(code, err)
		}
		adsetRaw = rows
	}()
	go func() {
		defer wg.Done()
		rows, code, err := directGraphGETPaged(ctx, lease, act+"/ads", url.Values{
			"fields": {"id,name,campaign_id,adset_id,status,effective_status,creative.thumbnail_width(240).thumbnail_height(240){id,thumbnail_url,image_url}"},
			"limit":  {"500"},
		}, 4)
		if code == http.StatusUnauthorized {
			recordErr(code, err)
		}
		adRaw = rows
	}()
	go func() {
		defer wg.Done()
		rows, code, err := directGraphGETPaged(ctx, lease, act+"/insights", url.Values{
			"level":      {"campaign"},
			"time_range": {timeRange},
			"fields":     {"campaign_id,campaign_name,spend,impressions,clicks,cpc,cpm,ctr"},
			"limit":      {"500"},
		}, 4)
		recordErr(code, err)
		campInsRaw = rows
	}()
	go func() {
		defer wg.Done()
		rows, _, _ := directGraphGETPaged(ctx, lease, act+"/insights", url.Values{
			"level":      {"adset"},
			"time_range": {timeRange},
			"fields":     {"campaign_id,campaign_name,adset_id,adset_name,spend,impressions,clicks,cpc,cpm,ctr"},
			"limit":      {"500"},
		}, 4)
		adsetInsRaw = rows
	}()
	go func() {
		defer wg.Done()
		rows, _, _ := directGraphGETPaged(ctx, lease, act+"/insights", url.Values{
			"level":      {"ad"},
			"time_range": {timeRange},
			"fields":     {"campaign_id,campaign_name,adset_id,adset_name,ad_id,ad_name,spend,impressions,clicks,cpc,cpm,ctr"},
			"limit":      {"500"},
		}, 4)
		adInsRaw = rows
	}()
	wg.Wait()

	if firstErr != nil && len(campRaw) == 0 && len(campInsRaw) == 0 {
		return MetaHierarchyBundle{}, firstCode, firstErr
	}

	return buildMetaHierarchyBundle(accRefID, act, currency, campRaw, adsetRaw, adRaw, campInsRaw, adsetInsRaw, adInsRaw), http.StatusOK, nil
}
