package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
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

func resolvePixelMetaOAuth(ctx context.Context, deps Deps, pixel *domain.Pixel) *domain.MetaAccount {
	if deps.MetaAcc == nil || deps.MetaOAuth == nil || pixel == nil {
		return nil
	}
	if pixel.MetaAccountID != nil && *pixel.MetaAccountID > 0 {
		if acc, err := deps.MetaAcc.FindByID(ctx, *pixel.MetaAccountID); err == nil && acc != nil && acc.Enabled && acc.IsConnected() {
			return acc
		}
	}
	if acc, err := deps.MetaAcc.FindFirstConnectedByAccount(ctx, pixel.AccountID); err == nil && acc != nil && acc.Enabled && acc.IsConnected() {
		return acc
	}
	return nil
}

func dispatchMetaWithOAuthFallback(ctx context.Context, deps Deps, pixel *domain.Pixel, in domain.SendEventInput, skip func(string) bool) []meta.SendResult {
	oauthAcc := resolvePixelMetaOAuth(ctx, deps, pixel)
	if oauthAcc != nil && in.PageID == "" && oauthAcc.DefaultPageID != "" && in.ActionSource == "business_messaging" {
		in.PageID = oauthAcc.DefaultPageID
	}
	results := make([]meta.SendResult, 0, len(pixel.PixelIDs))
	for i, pid := range pixel.PixelIDs {
		if skip != nil && skip(pid) {
			continue
		}
		tok := ""
		if i < len(pixel.AccessTokens) {
			tok = strings.TrimSpace(pixel.AccessTokens[i])
		}
		testCode := ""
		if i < len(pixel.TestCodes) {
			testCode = pixel.TestCodes[i]
		}
		if tok != "" || oauthAcc == nil {
			sub := *pixel
			sub.PixelIDs = []string{pid}
			sub.AccessTokens = []string{tok}
			sub.TestCodes = []string{testCode}
			results = append(results, deps.Meta.Send(ctx, &sub, in)...)
			continue
		}
		rawBody, err := meta.BuildCAPIPayload(in, testCode, deps.Meta.PartnerAgent())
		if err != nil {
			results = append(results, meta.SendResult{PixelID: pid, Err: err})
			continue
		}
		code, respBody, perr := deps.MetaOAuth.SendCAPIEvent(ctx, oauthAcc, pid, rawBody)
		res := meta.SendResult{PixelID: pid, StatusCode: code, Body: respBody, RequestBody: string(rawBody), Err: perr}
		if perr == nil && code >= 300 {
			res.Err = fmt.Errorf("meta oauth capi returned %d: %s", code, respBody)
		}
		results = append(results, res)
	}
	return results
}

func enrichAndDispatchCTWALead(
	ctx context.Context,
	deps Deps,
	inst *domain.WhatsAppInstance,
	u *domain.TrackedUser,
	chatID int64,
	phone, sourceID, ctwaClid string,
) {
	pixel, err := deps.Pixels.FindByID(ctx, inst.PixelID)
	if err != nil || pixel == nil || !pixel.Enabled {
		return
	}

	oauthAcc := resolvePixelMetaOAuth(ctx, deps, pixel)
	var metaInfo cachedAdMetadata
	if sourceID != "" {
		metaInfo = fetchAdMetadataCached(ctx, deps, sourceID, oauthAcc, pixel.MetaPermanentToken)
		if metaInfo.PageID == "" && oauthAcc != nil && oauthAcc.DefaultPageID != "" {
			metaInfo.PageID = oauthAcc.DefaultPageID
		}
		if metaInfo.CampaignName != "" || metaInfo.AdsetName != "" || metaInfo.AdName != "" || metaInfo.PageID != "" {
			_ = deps.WhatsApp.UpdateChatAdEnrichment(
				ctx, chatID,
				metaInfo.CampaignID, metaInfo.CampaignName,
				metaInfo.AdsetID, metaInfo.AdsetName,
				metaInfo.AdName, metaInfo.PageID, metaInfo.ThumbnailURL,
			)
			u.LastUTMCampaign = util.FirstNonEmpty(metaInfo.CampaignName, u.LastUTMCampaign)
			u.LastUTMMedium = util.FirstNonEmpty(metaInfo.AdsetName, u.LastUTMMedium)
			u.LastUTMContent = util.FirstNonEmpty(metaInfo.AdName, u.LastUTMContent)
			if u.FirstUTMCampaign == "" {
				u.FirstUTMCampaign = metaInfo.CampaignName
				u.FirstUTMMedium = metaInfo.AdsetName
				u.FirstUTMContent = metaInfo.AdName
			}
			_ = deps.Users.Upsert(ctx, u)
		}
	} else if oauthAcc != nil && oauthAcc.DefaultPageID != "" {
		metaInfo.PageID = oauthAcc.DefaultPageID
	}

	if ctwaClid == "" || deps.Enqueuer == nil {
		return
	}
	eventID := "ctwa:" + ctwaClid
	if exists, _ := deps.SentEvents.ExistsSent(ctx, eventID, pixel.Name); exists {
		return
	}
	_ = deps.Enqueuer.EnqueueSendEvent(ctx, domain.SendEventInput{
		PixelName:        pixel.Name,
		EventName:        domain.EventLeadSubmitted,
		EventID:          eventID,
		EventTime:        time.Now().Unix(),
		Gateway:          "whatsapp",
		ActionSource:     "business_messaging",
		MessagingChannel: "whatsapp",
		CtwaClid:         ctwaClid,
		PageID:           metaInfo.PageID,
		ExternalID:       u.ExternalID,
		IDWpp:            u.IDWpp,
		ClientIP:         u.ClientIP,
		ClientUserAgent:  u.ClientUserAgent,
		Email:            u.Email,
		Phone:            util.FirstNonEmpty(phone, u.Phone),
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		Country:          u.Country,
		State:            u.State,
		City:             u.City,
		PostalCode:       u.PostalCode,
		FBP:              u.FBP,
		FBC:              u.FBC,
		GCLID:            u.GCLID,
		WBraid:           u.WBraid,
		GBraid:           u.GBraid,
		TTClid:           u.TTClid,
		TTP:              u.TTP,
		GA4ClientID:      u.GA4ClientID,
		GA4SessionID:     u.GA4SessionID,
	})
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
		Name          string                       `json:"name"`
		Campaign      struct{ ID, Name string }    `json:"campaign"`
		Adset         struct{ ID, Name string }    `json:"adset"`
		Creative      struct{ ThumbnailURL, ImageURL string } `json:"creative"`
		TrackingSpecs []struct{ Page []string }    `json:"tracking_specs"`
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
