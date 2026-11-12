package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type cachedAdMetadata struct {
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	AdsetID      string `json:"adset_id"`
	AdsetName    string `json:"adset_name"`
	AdName       string `json:"ad_name"`
	PageID       string `json:"page_id"`
	ThumbnailURL string `json:"thumbnail_url"`
}

func fetchAdMetadataCached(ctx context.Context, deps Deps, sourceID string, oauthAcc *domain.MetaAccount, legacyToken string) cachedAdMetadata {
	cacheKey := "tf:meta:ad:" + sourceID
	if deps.Redis != nil {
		if raw, err := deps.Redis.Get(ctx, cacheKey).Bytes(); err == nil && len(raw) > 0 {
			var cached cachedAdMetadata
			if json.Unmarshal(raw, &cached) == nil {
				return cached
			}
		}
	}

	var out cachedAdMetadata
	if oauthAcc != nil && deps.MetaOAuth != nil {
		if res, err := deps.MetaOAuth.GetAdMetadata(ctx, oauthAcc, sourceID); err == nil {
			out = cachedAdMetadata{
				CampaignID:   res.CampaignID,
				CampaignName: res.CampaignName,
				AdsetID:      res.AdsetID,
				AdsetName:    res.AdsetName,
				AdName:       res.AdName,
				PageID:       res.PageID,
				ThumbnailURL: res.ThumbnailURL,
			}
		}
	}

	if out.AdName == "" && out.CampaignName == "" && legacyToken != "" {
		out = fetchAdMetadataDirect(ctx, sourceID, legacyToken)
	}

	if (out.AdName != "" || out.CampaignName != "" || out.PageID != "") && deps.Redis != nil {
		if b, err := json.Marshal(out); err == nil {
			_ = deps.Redis.Set(ctx, cacheKey, b, 24*time.Hour).Err()
		}
	}
	return out
}

func fetchAdMetadataDirect(ctx context.Context, sourceID, token string) cachedAdMetadata {
	adURL := fmt.Sprintf("https://graph.facebook.com/v26.0/%s?fields=name,campaign{id,name},adset{id,name},creative{thumbnail_url,image_url},tracking_specs&access_token=%s",
		url.PathEscape(sourceID), url.QueryEscape(token))
	cli := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adURL, nil)
	if err != nil {
		return cachedAdMetadata{}
	}
	resp, err := cli.Do(req)
	if err != nil {
		return cachedAdMetadata{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return cachedAdMetadata{}
	}
	var adData struct {
		Name          string                                  `json:"name"`
		Campaign      struct{ ID, Name string }               `json:"campaign"`
		Adset         struct{ ID, Name string }               `json:"adset"`
		Creative      struct{ ThumbnailURL, ImageURL string } `json:"creative"`
		TrackingSpecs []struct{ Page []string }               `json:"tracking_specs"`
	}
	if json.NewDecoder(resp.Body).Decode(&adData) != nil {
		return cachedAdMetadata{}
	}
	out := cachedAdMetadata{
		CampaignID: adData.Campaign.ID, CampaignName: adData.Campaign.Name,
		AdsetID: adData.Adset.ID, AdsetName: adData.Adset.Name, AdName: adData.Name,
		ThumbnailURL: util.FirstNonEmpty(adData.Creative.ThumbnailURL, adData.Creative.ImageURL),
	}
	for _, sp := range adData.TrackingSpecs {
		if len(sp.Page) > 0 && sp.Page[0] != "" {
			out.PageID = sp.Page[0]
			break
		}
	}
	return out
}

