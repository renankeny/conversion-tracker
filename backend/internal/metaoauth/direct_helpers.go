package metaoauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/meta"
)

type capiOutcome struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}

func digitsOnlyStr(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeMetaActID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "act_") {
		return "act_" + digitsOnlyStr(s)
	}
	return s
}

func parseDirectLiveInsights(raw []byte) meta.LiveInsightsSummary {
	var res struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(raw, &res)
	var out meta.LiveInsightsSummary
	out.Connected = true
	for _, row := range res.Data {
		var sp float64
		var cl, imp int64
		if s, ok := row["spend"].(string); ok {
			_, _ = fmt.Sscanf(s, "%f", &sp)
		}
		if s, ok := row["clicks"].(string); ok {
			_, _ = fmt.Sscanf(s, "%d", &cl)
		}
		if s, ok := row["impressions"].(string); ok {
			_, _ = fmt.Sscanf(s, "%d", &imp)
		}
		msg, ld, pur, rev := extractDirectAccountConversions(row["actions"], row["action_values"])
		out.Spend += sp
		out.Clicks += cl
		out.Impressions += imp
		out.Messages += msg
		out.Leads += ld
		out.Purchases += pur
		out.Revenue += rev
	}
	return out
}

func extractDirectAccountConversions(rawActions, rawValues any) (messages, leads, purchases int64, revenue float64) {
	if arr, ok := rawActions.([]any); ok {
		for _, el := range arr {
			m, ok := el.(map[string]any)
			if !ok {
				continue
			}
			aType, _ := m["action_type"].(string)
			valStr, _ := m["value"].(string)
			var v int64
			_, _ = fmt.Sscanf(valStr, "%d", &v)
			switch aType {
			case "onsite_conversion.messaging_conversation_started_7d", "onsite_conversion.total_messaging_connection":
				messages = max(messages, v)
			case "lead", "onsite_conversion.lead_grouped", "offsite_conversion.fb_pixel_lead":
				leads = max(leads, v)
			case "purchase", "omni_purchase", "onsite_conversion.purchase", "offsite_conversion.fb_pixel_purchase":
				purchases = max(purchases, v)
			}
		}
	}
	if arr, ok := rawValues.([]any); ok {
		for _, el := range arr {
			m, ok := el.(map[string]any)
			if !ok {
				continue
			}
			aType, _ := m["action_type"].(string)
			valStr, _ := m["value"].(string)
			var v float64
			_, _ = fmt.Sscanf(valStr, "%f", &v)
			switch aType {
			case "purchase", "omni_purchase", "onsite_conversion.purchase", "offsite_conversion.fb_pixel_purchase":
				revenue = max(revenue, v)
			}
		}
	}
	return messages, leads, purchases, revenue
}

func formatDirectPlacement(pub, pos string) string {
	p, s := strings.ToLower(strings.TrimSpace(pub)), strings.ToLower(strings.TrimSpace(pos))
	switch p {
	case "instagram":
		switch {
		case strings.Contains(s, "reels"):
			return "Instagram Reels"
		case strings.Contains(s, "story") || strings.Contains(s, "stories"):
			return "Instagram Stories"
		case strings.Contains(s, "explore"):
			return "Instagram Explorar"
		default:
			return "Feed do Instagram"
		}
	case "facebook":
		switch {
		case strings.Contains(s, "reels"):
			return "Facebook Reels"
		case strings.Contains(s, "stories") || strings.Contains(s, "story"):
			return "Facebook Stories"
		case strings.Contains(s, "marketplace"):
			return "Facebook Marketplace"
		case strings.Contains(s, "video"):
			return "Facebook Vídeo"
		default:
			return "Feed do Facebook"
		}
	case "messenger":
		return "Messenger"
	case "audience_network":
		return "Audience Network"
	}
	if pub != "" && pos != "" {
		return pub + " · " + pos
	}
	return pub
}

func directFetchCreativeRanking(ctx context.Context, lease metaLeaseEntry, adAccountID, timeRange string) ([]meta.CreativeInsight, int, error) {
	act := normalizeMetaActID(adAccountID)
	if act == "" || act == "act_" {
		return nil, http.StatusBadRequest, errors.New("ad_account_id ausente")
	}
	q := url.Values{"level": {"ad"}, "fields": {"ad_name,ad_id,spend,clicks,actions"}, "time_range": {timeRange}, "sort": {"spend_descending"}, "limit": {"3"}}
	raw, code, err := directGraphGET(ctx, lease, act+"/insights", q)
	if err != nil {
		return nil, code, err
	}
	var res struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(raw, &res)

	placements := map[string]string{}
	pq := url.Values{"level": {"ad"}, "fields": {"ad_id,impressions,spend"}, "breakdowns": {"publisher_platform,platform_position"}, "time_range": {timeRange}, "sort": {"impressions_descending"}, "limit": {"25"}}
	if pRaw, _, perr := directGraphGET(ctx, lease, act+"/insights", pq); perr == nil {
		var pRes struct {
			Data []map[string]any `json:"data"`
		}
		if json.Unmarshal(pRaw, &pRes) == nil {
			for _, row := range pRes.Data {
				aid, _ := row["ad_id"].(string)
				if aid != "" && placements[aid] == "" {
					pub, _ := row["publisher_platform"].(string)
					pos, _ := row["platform_position"].(string)
					placements[aid] = formatDirectPlacement(pub, pos)
				}
			}
		}
	}

	out := make([]meta.CreativeInsight, 0, len(res.Data))
	for _, item := range res.Data {
		adID, _ := item["ad_id"].(string)
		adName, _ := item["ad_name"].(string)
		spend, _ := item["spend"].(string)
		clicks, _ := item["clicks"].(string)
		_, leads, purchases, _ := extractDirectAccountConversions(item["actions"], nil)
		ci := meta.CreativeInsight{
			AdID: adID, AdName: adName, Spend: spend, Clicks: clicks,
			Leads: leads, Purchases: purchases, Placement: placements[adID],
		}
		if clean := digitsOnlyStr(adID); clean != "" {
			cq := url.Values{"fields": {"creative.thumbnail_width(480).thumbnail_height(480){image_url,thumbnail_url}"}}
			if cRaw, _, cerr := directGraphGET(ctx, lease, clean, cq); cerr == nil {
				var cData struct {
					Creative struct {
						ImageURL     string `json:"image_url"`
						ThumbnailURL string `json:"thumbnail_url"`
					} `json:"creative"`
				}
				if json.Unmarshal(cRaw, &cData) == nil {
					ci.ImageURL = cData.Creative.ImageURL
					ci.ThumbnailURL = cData.Creative.ThumbnailURL
				}
			}
		}
		out = append(out, ci)
	}
	return out, http.StatusOK, nil
}
