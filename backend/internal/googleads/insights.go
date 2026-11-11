package googleads

import (
	"context"
	"errors"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type LiveInsightsSummary struct {
	Spend       float64 `json:"spend"`
	Clicks      int64   `json:"clicks"`
	Impressions int64   `json:"impressions"`
	Connected   bool    `json:"connected"`
}

type KeywordInsight struct {
	Keyword      string  `json:"keyword"`
	MatchType    string  `json:"match_type"`
	CampaignName string  `json:"campaign_name"`
	Clicks       int64   `json:"clicks"`
	Impressions  int64   `json:"impressions"`
	Spend        float64 `json:"spend"`
	CPC          float64 `json:"cpc"`
}

func (c *Client) FetchLiveInsights(ctx context.Context, account *domain.GoogleAdsAccount, since, until string) (LiveInsightsSummary, error) {
	if account == nil || account.RefreshToken == "" || account.CustomerID == "" {
		return LiveInsightsSummary{Connected: false}, errors.New("conta Google Ads não conectada")
	}
	payload := map[string]string{"since": since, "until": until}
	var out LiveInsightsSummary
	if err := c.oauth.CallCentralProxy(ctx, "live_insights", account.RefreshToken, account.CustomerID, account.LoginCustomerID, payload, &out); err != nil {
		return LiveInsightsSummary{Connected: false}, err
	}
	out.Connected = true
	return out, nil
}

func (c *Client) FetchKeywordRanking(ctx context.Context, account *domain.GoogleAdsAccount, since, until string) ([]KeywordInsight, error) {
	if account == nil || account.RefreshToken == "" || account.CustomerID == "" {
		return []KeywordInsight{}, errors.New("conta Google Ads não conectada")
	}
	payload := map[string]string{"since": since, "until": until}
	var out struct {
		Keywords []KeywordInsight `json:"keywords"`
	}
	if err := c.oauth.CallCentralProxy(ctx, "keyword_ranking", account.RefreshToken, account.CustomerID, account.LoginCustomerID, payload, &out); err != nil {
		return []KeywordInsight{}, err
	}
	if out.Keywords == nil {
		return []KeywordInsight{}, nil
	}
	return out.Keywords, nil
}

