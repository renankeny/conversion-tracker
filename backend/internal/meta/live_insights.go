package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const GraphAPIBaseURL = "https://graph.facebook.com/v26.0"

var insightsHTTPClient = &http.Client{Timeout: 30 * time.Second}

type CampaignInsight struct {
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	Spend        string `json:"spend"`
	Clicks       string `json:"clicks"`
	Impressions  string `json:"impressions"`
}

type CreativeInsight struct {
	AdID         string `json:"ad_id"`
	AdName       string `json:"ad_name"`
	Spend        string `json:"spend"`
	Clicks       string `json:"clicks"`
	Leads        int64  `json:"leads"`
	Purchases    int64  `json:"purchases"`
	Placement    string `json:"placement,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

type MetaResponse struct {
	Data []map[string]interface{} `json:"data"`
}

type LiveInsightsSummary struct {
	Spend       float64 `json:"spend"`
	Clicks      int64   `json:"clicks"`
	Impressions int64   `json:"impressions"`
	Messages    int64   `json:"messages"`
	Leads       int64   `json:"leads"`
	Purchases   int64   `json:"purchases"`
	Revenue     float64 `json:"revenue"`
	PrevSpend   float64 `json:"prev_spend,omitempty"`
	PrevClicks  int64   `json:"prev_clicks,omitempty"`
	Connected   bool    `json:"connected"`
}

func FetchLiveSpend(ctx context.Context, adAccountID, token, timeRange string) (float64, error) {
	res, err := FetchLiveInsights(ctx, adAccountID, token, timeRange)
	return res.Spend, err
}

func extractAccountActions(rawActions, rawValues any) (messages, leads, purchases int64, revenue float64) {
	if arr, ok := rawActions.([]interface{}); ok {
		for _, el := range arr {
			m, ok := el.(map[string]interface{})
			if !ok {
				continue
			}
			aType, _ := m["action_type"].(string)
			valStr, _ := m["value"].(string)
			var v int64
			_, _ = fmt.Sscanf(valStr, "%d", &v)
			switch aType {
			case "onsite_conversion.messaging_conversation_started_7d", "onsite_conversion.total_messaging_connection":
				if v > messages {
					messages = v
				}
			case "lead", "onsite_conversion.lead_grouped", "offsite_conversion.fb_pixel_lead":
				if v > leads {
					leads = v
				}
			case "purchase", "omni_purchase", "onsite_conversion.purchase", "offsite_conversion.fb_pixel_purchase":
				if v > purchases {
					purchases = v
				}
			}
		}
	}
	if arr, ok := rawValues.([]interface{}); ok {
		for _, el := range arr {
			m, ok := el.(map[string]interface{})
			if !ok {
				continue
			}
			aType, _ := m["action_type"].(string)
			valStr, _ := m["value"].(string)
			var v float64
			_, _ = fmt.Sscanf(valStr, "%f", &v)
			switch aType {
			case "purchase", "omni_purchase", "onsite_conversion.purchase", "offsite_conversion.fb_pixel_purchase":
				if v > revenue {
					revenue = v
				}
			}
		}
	}
	return messages, leads, purchases, revenue
}

func FetchLiveInsights(ctx context.Context, adAccountID, token, timeRange string) (LiveInsightsSummary, error) {
	adAccountID = strings.TrimSpace(adAccountID)
	if adAccountID != "" && !strings.HasPrefix(adAccountID, "act_") {
		adAccountID = "act_" + adAccountID
	}

	reqURL := fmt.Sprintf("%s/%s/insights?level=account&fields=spend,clicks,impressions,actions,action_values&time_range=%s&access_token=%s",
		GraphAPIBaseURL,
		url.PathEscape(adAccountID),
		url.QueryEscape(timeRange),
		url.QueryEscape(token),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return LiveInsightsSummary{}, fmt.Errorf("new request: %w", err)
	}

	resp, err := insightsHTTPClient.Do(req)
	if err != nil {
		return LiveInsightsSummary{}, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return LiveInsightsSummary{}, fmt.Errorf("meta api error: status %d", resp.StatusCode)
	}

	var res MetaResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return LiveInsightsSummary{}, fmt.Errorf("decode json: %w", err)
	}

	var out LiveInsightsSummary
	out.Connected = true
	for _, item := range res.Data {
		if spendStr, ok := item["spend"].(string); ok {
			var spend float64
			_, _ = fmt.Sscanf(spendStr, "%f", &spend)
			out.Spend += spend
		}
		if clicksStr, ok := item["clicks"].(string); ok {
			var cl int64
			_, _ = fmt.Sscanf(clicksStr, "%d", &cl)
			out.Clicks += cl
		}
		if imprStr, ok := item["impressions"].(string); ok {
			var imp int64
			_, _ = fmt.Sscanf(imprStr, "%d", &imp)
			out.Impressions += imp
		}
		msg, ld, pur, rev := extractAccountActions(item["actions"], item["action_values"])
		out.Messages += msg
		out.Leads += ld
		out.Purchases += pur
		out.Revenue += rev
	}
	return out, nil
}
