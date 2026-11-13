package queue

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

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
	if oauthAcc != nil && in.PageID == "" && oauthAcc.DefaultPageID != "" && (in.ActionSource == "business_messaging" || in.CtwaClid != "") {
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
	isInbound bool,
) error {
	pixel, err := deps.Pixels.FindByID(ctx, inst.PixelID)
	if err != nil || pixel == nil || !pixel.Enabled {
		return nil
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

	if !isInbound || deps.Enqueuer == nil || (ctwaClid == "" && sourceID == "") {
		return nil
	}

	actionSource := "business_messaging"
	messagingChannel := "whatsapp"
	eventID := "ctwa:" + ctwaClid
	if ctwaClid == "" {
		actionSource = "system_generated"
		messagingChannel = ""
		eventID = fmt.Sprintf("ctwa_ad:%s:%d", sourceID, chatID)
	} else if metaInfo.PageID == "" {
		actionSource = "system_generated"
		messagingChannel = ""
		ctwaClid = ""
	}

	if exists, _ := deps.SentEvents.ExistsSent(ctx, pixel.AccountID, eventID, pixel.Name, ""); exists {
		return nil
	}

	var linked domain.LinkedTrackingFields
	if deps.Users != nil {
		linked, _ = deps.Users.LinkedFieldsByAccount(ctx, inst.AccountID, u.ExternalID)
	}

	if err := deps.Enqueuer.EnqueueSendEvent(ctx, domain.SendEventInput{
		PixelName:        pixel.Name,
		EventName:        domain.EventLeadSubmitted,
		EventID:          eventID,
		EventTime:        time.Now().Unix(),
		Gateway:          "whatsapp",
		ActionSource:     actionSource,
		MessagingChannel: messagingChannel,
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
		FBP:              util.FirstNonEmpty(u.FBP, linked.FBP),
		FBC:              util.FirstNonEmpty(u.FBC, linked.FBC),
		GCLID:            util.FirstNonEmpty(u.GCLID, linked.GCLID),
		WBraid:           util.FirstNonEmpty(u.WBraid, linked.WBraid),
		GBraid:           util.FirstNonEmpty(u.GBraid, linked.GBraid),
		TTClid:           util.FirstNonEmpty(u.TTClid, linked.TTClid),
		TTP:              util.FirstNonEmpty(u.TTP, linked.TTP),
		GA4ClientID:      util.FirstNonEmpty(u.GA4ClientID, linked.GA4ClientID),
		GA4SessionID:     util.FirstNonEmpty(u.GA4SessionID, linked.GA4SessionID),
		Currency:         domain.NormalizeWhatsAppCurrency(inst.Currency),
	}); err != nil {
		slog.Error("enqueue ctwa lead event failed", "pixel_name", pixel.Name, "event_id", eventID, "error", err)
		return fmt.Errorf("enqueue ctwa lead event: %w", err)
	}
	return nil
}
