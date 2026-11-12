package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func extractActions(raw any) (leads int64, purchases int64) {
	arr, ok := raw.([]interface{})
	if !ok {
		return 0, 0
	}
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
		case "lead", "onsite_conversion.lead_grouped", "offsite_conversion.fb_pixel_lead", "onsite_conversion.messaging_conversation_started_7d":
			if v > leads {
				leads = v
			}
		case "purchase", "omni_purchase", "onsite_conversion.purchase", "offsite_conversion.fb_pixel_purchase":
			if v > purchases {
				purchases = v
			}
		}
	}
	return leads, purchases
}

func FetchCreativeRanking(ctx context.Context, adAccountID, token, timeRange string) ([]CreativeInsight, error) {
	adAccountID = strings.TrimSpace(adAccountID)
	if adAccountID != "" && !strings.HasPrefix(adAccountID, "act_") {
		adAccountID = "act_" + adAccountID
	}

	reqURL := fmt.Sprintf("%s/%s/insights?level=ad&fields=ad_name,ad_id,spend,clicks,actions&time_range=%s&sort=spend_descending&limit=3&access_token=%s",
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
		leads, purchases := extractActions(item["actions"])

		topAds = append(topAds, CreativeInsight{
			AdID:      adID,
			AdName:    adName,
			Spend:     spend,
			Clicks:    clicks,
			Leads:     leads,
			Purchases: purchases,
		})

		batchRequests = append(batchRequests, map[string]string{
			"method":       "GET",
			"relative_url": fmt.Sprintf("%s?fields=creative.thumbnail_width(480).thumbnail_height(480){image_url,thumbnail_url}", adID),
		})
	}

	if len(batchRequests) == 0 {
		return topAds, nil
	}

	batchJSON, _ := json.Marshal(batchRequests)
	batchURL := fmt.Sprintf("%s?access_token=%s&batch=%s", GraphAPIBaseURL, url.QueryEscape(token), url.QueryEscape(string(batchJSON)))

	breq, err := http.NewRequestWithContext(ctx, http.MethodPost, batchURL, nil)
	if err != nil {
		return topAds, nil
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
					} `json:"creative"`
				}
				if err := json.Unmarshal([]byte(br.Body), &bodyData); err == nil {
					topAds[i].ImageURL = bodyData.Creative.ImageURL
					topAds[i].ThumbnailURL = bodyData.Creative.ThumbnailURL
				}
			}
		}
	}
	return topAds, nil
}

