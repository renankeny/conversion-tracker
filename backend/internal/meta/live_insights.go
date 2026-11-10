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
	ImageURL     string `json:"image_url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

type MetaResponse struct {
	Data []map[string]interface{} `json:"data"`
}

// FetchLiveSpend fetches the aggregated spend across campaigns for a given time_range.
// timeRange should be in JSON format: {"since": "YYYY-MM-DD", "until": "YYYY-MM-DD"}
func FetchLiveSpend(ctx context.Context, adAccountID, token, timeRange string) (float64, error) {
	adAccountID = strings.TrimSpace(adAccountID)
	if adAccountID != "" && !strings.HasPrefix(adAccountID, "act_") {
		adAccountID = "act_" + adAccountID
	}

	reqURL := fmt.Sprintf("%s/%s/insights?level=account&fields=spend&time_range=%s&access_token=%s",
		GraphAPIBaseURL,
		url.PathEscape(adAccountID),
		url.QueryEscape(timeRange),
		url.QueryEscape(token),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("new request: %w", err)
	}

	resp, err := insightsHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("meta api error: status %d", resp.StatusCode)
	}

	var res MetaResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, fmt.Errorf("decode json: %w", err)
	}

	var totalSpend float64
	for _, item := range res.Data {
		if spendStr, ok := item["spend"].(string); ok {
			var spend float64
			fmt.Sscanf(spendStr, "%f", &spend)
			totalSpend += spend
		}
	}

	return totalSpend, nil
}

// FetchCreativeRanking fetches top creatives (ads) by spend/results using insights and batch api.
func FetchCreativeRanking(ctx context.Context, adAccountID, token, timeRange string) ([]CreativeInsight, error) {
	adAccountID = strings.TrimSpace(adAccountID)
	if adAccountID != "" && !strings.HasPrefix(adAccountID, "act_") {
		adAccountID = "act_" + adAccountID
	}

	// 1. Fetch Insights level=ad
	reqURL := fmt.Sprintf("%s/%s/insights?level=ad&fields=ad_name,ad_id,spend,clicks&time_range=%s&sort=spend_descending&limit=10&access_token=%s",
		GraphAPIBaseURL,
		url.PathEscape(adAccountID),
		url.QueryEscape(timeRange),
		url.QueryEscape(token),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	resp, err := insightsHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("meta api error: status %d", resp.StatusCode)
	}

	var res MetaResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	var topAds []CreativeInsight
	var batchRequests []map[string]string

	for _, item := range res.Data {
		adID, _ := item["ad_id"].(string)
		adName, _ := item["ad_name"].(string)
		spend, _ := item["spend"].(string)
		clicks, _ := item["clicks"].(string)

		topAds = append(topAds, CreativeInsight{
			AdID:   adID,
			AdName: adName,
			Spend:  spend,
			Clicks: clicks,
		})

		batchRequests = append(batchRequests, map[string]string{
			"method":       "GET",
			"relative_url": fmt.Sprintf("%s?fields=creative{image_url,thumbnail_url}", adID),
		})
	}

	if len(batchRequests) == 0 {
		return topAds, nil
	}

	// 2. Batch Request for creative URLs
	batchJSON, _ := json.Marshal(batchRequests)
	batchURL := fmt.Sprintf("%s?access_token=%s&batch=%s", GraphAPIBaseURL, url.QueryEscape(token), url.QueryEscape(string(batchJSON)))

	breq, err := http.NewRequestWithContext(ctx, http.MethodPost, batchURL, nil)
	if err != nil {
		return topAds, nil // return what we have
	}

	bresp, err := insightsHTTPClient.Do(breq)
	if err != nil || bresp.StatusCode != http.StatusOK {
		if bresp != nil {
			bresp.Body.Close()
		}
		return topAds, nil
	}
	defer bresp.Body.Close()

	var batchRes []struct {
		Code int    `json:"code"`
		Body string `json:"body"`
	}
	if err := json.NewDecoder(bresp.Body).Decode(&batchRes); err == nil {
		for i, br := range batchRes {
			if br.Code == 200 && i < len(topAds) {
				var bodyData struct {
					Creative struct {
						ImageURL     string `json:"image_url"`
						ThumbnailURL string `json:"thumbnail_url"`
						Data         []struct {
							ImageURL     string `json:"image_url"`
							ThumbnailURL string `json:"thumbnail_url"`
						} `json:"data"`
					} `json:"creative"`
				}
				if err := json.Unmarshal([]byte(br.Body), &bodyData); err == nil {
					if bodyData.Creative.ImageURL != "" || bodyData.Creative.ThumbnailURL != "" {
						topAds[i].ImageURL = bodyData.Creative.ImageURL
						topAds[i].ThumbnailURL = bodyData.Creative.ThumbnailURL
					} else if len(bodyData.Creative.Data) > 0 {
						topAds[i].ImageURL = bodyData.Creative.Data[0].ImageURL
						topAds[i].ThumbnailURL = bodyData.Creative.Data[0].ThumbnailURL
					}
				}
			}
		}
	}

	return topAds, nil
}
