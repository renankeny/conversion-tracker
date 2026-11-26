package googleads

import (
	"context"
	"errors"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// HierarchyBundle contém campanhas, grupos (AdGroup + PMax AssetGroup) e anúncios via Google Ads API v26.
type HierarchyBundle struct {
	Currency  string
	Campaigns []domain.CampaignRow
	AdSets    []domain.CampaignRow
	Ads       []domain.CampaignRow
}

type proxyHierarchyItem struct {
	ID              string  `json:"id"`
	Level           string  `json:"level"`
	EntitySubtype   string  `json:"entity_subtype"`
	CampaignID      string  `json:"campaign_id"`
	CampaignName    string  `json:"campaign_name"`
	AdSetID         string  `json:"adset_id"`
	AdSetName       string  `json:"adset_name"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	EffectiveStatus string  `json:"effective_status"`
	ChannelType     string  `json:"channel_type"`
	BudgetID        string  `json:"budget_id"`
	BudgetShared    bool    `json:"budget_shared"`
	DailyBudget     float64 `json:"daily_budget"`
	Spend           float64 `json:"spend"`
	Impressions     int64   `json:"impressions"`
	Clicks          int64   `json:"clicks"`
}

func (c *Client) FetchHierarchyInsights(
	ctx context.Context,
	account *domain.GoogleAdsAccount,
	customerID, loginCustomerID, since, until string,
) (HierarchyBundle, error) {
	if account == nil || strings.TrimSpace(account.RefreshToken) == "" {
		return HierarchyBundle{}, errors.New("conta Google Ads não conectada")
	}
	cid := strings.TrimSpace(customerID)
	if cid == "" {
		cid = strings.TrimSpace(account.CustomerID)
	}
	if cid == "" {
		return HierarchyBundle{}, errors.New("customer_id Google Ads ausente")
	}
	lcid := strings.TrimSpace(loginCustomerID)
	if lcid == "" {
		lcid = strings.TrimSpace(account.LoginCustomerID)
	}

	var resp struct {
		Currency  string               `json:"currency"`
		Campaigns []proxyHierarchyItem `json:"campaigns"`
		AdSets    []proxyHierarchyItem `json:"adsets"`
		Ads       []proxyHierarchyItem `json:"ads"`
	}
	payload := map[string]string{"since": since, "until": until}
	if err := c.oauth.CallCentralProxy(ctx, "list_hierarchy_insights", account.RefreshToken, cid, lcid, payload, &resp); err != nil {
		return HierarchyBundle{}, err
	}
	curr := strings.ToUpper(strings.TrimSpace(resp.Currency))
	if curr == "" {
		curr = "BRL"
	}
	return HierarchyBundle{
		Currency:  curr,
		Campaigns: mapGoogleProxyItems(resp.Campaigns, account.ID, cid, lcid, curr),
		AdSets:    mapGoogleProxyItems(resp.AdSets, account.ID, cid, lcid, curr),
		Ads:       mapGoogleProxyItems(resp.Ads, account.ID, cid, lcid, curr),
	}, nil
}

func mapGoogleProxyItems(items []proxyHierarchyItem, accID int64, cid, lcid, currency string) []domain.CampaignRow {
	out := make([]domain.CampaignRow, 0, len(items))
	for _, it := range items {
		row := domain.CampaignRow{
			ID:              it.ID,
			Platform:        "google",
			Level:           it.Level,
			EntitySubtype:   it.EntitySubtype,
			AccountRefID:    accID,
			AdAccountID:     cid,
			LoginCustomerID: lcid,
			Currency:        currency,
			CampaignID:      it.CampaignID,
			CampaignName:    it.CampaignName,
			AdSetID:         it.AdSetID,
			AdSetName:       it.AdSetName,
			Name:            it.Name,
			Status:          it.Status,
			EffectiveStatus: it.EffectiveStatus,
			ChannelType:     it.ChannelType,
			BudgetID:        it.BudgetID,
			BudgetShared:    it.BudgetShared,
			DailyBudget:     it.DailyBudget,
			Spend:           it.Spend,
			Impressions:     it.Impressions,
			Clicks:          it.Clicks,
			CanEditStatus:   it.Status != "ARCHIVED",
			CanEditBudget:   it.Level == "campaign" && it.BudgetID != "" && it.Status != "ARCHIVED",
		}
		if row.Level == "campaign" && row.DailyBudget > 0 {
			row.BudgetLevel = "campaign"
		}
		if row.Impressions > 0 {
			row.CTR = (float64(row.Clicks) / float64(row.Impressions)) * 100.0
			row.CPM = (row.Spend / float64(row.Impressions)) * 1000.0
		}
		if row.Clicks > 0 {
			row.CPC = row.Spend / float64(row.Clicks)
		}
		out = append(out, row)
	}
	return out
}
