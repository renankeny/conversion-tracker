package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func (h *Webhooks) resolveWebhookAccount(ctx context.Context, in *purchaseInput) (*domain.Pixel, int, string) {
	var resolved *domain.Pixel
	if in.PixelQuery != "" && h.pixels != nil {
		if p, err := h.pixels.FindByNameOrToken(ctx, in.PixelQuery); err == nil && p != nil && p.Enabled {
			resolved = p
		}
	}

	if resolved != nil {
		if in.AccountID > 0 && in.AccountID != resolved.AccountID {
			slog.Warn("webhook pixel/account mismatch",
				"gateway", in.GatewayName, "account_id", in.AccountID, "pixel_account_id", resolved.AccountID)
			return nil, http.StatusForbidden, "pixel does not belong to account"
		}
		in.AccountID = resolved.AccountID
	}

	if in.AccountID <= 0 {
		if sole, ok := h.users.SoleAccountID(ctx); ok {
			in.AccountID = sole
		} else {
			slog.Warn("webhook without account context", "gateway", in.GatewayName)
			return nil, http.StatusBadRequest, "missing account context: add ?acc= or ?pixel= to the webhook url"
		}
	}
	return resolved, 0, ""
}

func (h *Webhooks) selectFallbackPixel(ctx context.Context, resolvedQueryPixel *domain.Pixel, accountID int64) *domain.Pixel {
	if resolvedQueryPixel != nil {
		return resolvedQueryPixel
	}
	if accountID <= 0 || h.pixels == nil {
		return nil
	}
	list, err := h.pixels.ListByAccount(ctx, accountID)
	if err != nil {
		return nil
	}
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Enabled {
			return &list[i]
		}
	}
	return nil
}

func webhookEventID(orderID, itemType, productID string) string {
	if orderID == "" {
		return util.GenerateID10()
	}
	if itemType == "" || itemType == "main" {
		return orderID
	}
	return orderID + ":" + itemType + ":" + productID
}

func webhookEventIDForEvent(eventName, orderID, itemType, productID string) string {
	base := webhookEventID(orderID, itemType, productID)
	ev := strings.TrimSpace(eventName)
	if ev == "" || strings.EqualFold(ev, domain.EventPurchase) {
		return base
	}
	return strings.ToLower(ev) + ":" + base
}

func applyWebhookTrackingEnrichment(user *domain.TrackedUser, in purchaseInput) {
	if user == nil {
		return
	}
	user.ClientIP = util.FirstNonEmpty(user.ClientIP, strings.TrimSpace(in.ClientIP))
	user.ClientUserAgent = util.FirstNonEmpty(user.ClientUserAgent, strings.TrimSpace(in.ClientUserAgent))
	user.Country = util.FirstNonEmpty(user.Country, strings.TrimSpace(in.Country))
	user.State = util.FirstNonEmpty(user.State, strings.TrimSpace(in.State))
	user.City = util.FirstNonEmpty(user.City, strings.TrimSpace(in.City))
	user.PostalCode = util.FirstNonEmpty(user.PostalCode, strings.TrimSpace(in.PostalCode))
	user.GCLID = util.FirstNonEmpty(user.GCLID, strings.TrimSpace(in.GCLID))
	user.TTClid = util.FirstNonEmpty(user.TTClid, strings.TrimSpace(in.TTClid))
	if srcURL := strings.TrimSpace(in.SourceURL); srcURL != "" {
		user.URL = util.FirstNonEmpty(user.URL, srcURL)
		user.FirstURL = util.FirstNonEmpty(user.FirstURL, srcURL)
	}
	cleanSrc := util.StripUserIDFromUTM(in.UTMSource)
	cleanMed := util.StripUserIDFromUTM(in.UTMMedium)
	cleanCmp := util.StripUserIDFromUTM(in.UTMCampaign)
	cleanCnt := util.StripUserIDFromUTM(in.UTMContent)
	cleanTrm := util.StripUserIDFromUTM(in.UTMTerm)
	user.FirstUTMSource = util.FirstNonEmpty(user.FirstUTMSource, cleanSrc)
	user.FirstUTMMedium = util.FirstNonEmpty(user.FirstUTMMedium, cleanMed)
	user.FirstUTMCampaign = util.FirstNonEmpty(user.FirstUTMCampaign, cleanCmp)
	user.FirstUTMContent = util.FirstNonEmpty(user.FirstUTMContent, cleanCnt)
	user.FirstUTMTerm = util.FirstNonEmpty(user.FirstUTMTerm, cleanTrm)
	user.LastUTMSource = util.FirstNonEmpty(user.LastUTMSource, cleanSrc)
	user.LastUTMMedium = util.FirstNonEmpty(user.LastUTMMedium, cleanMed)
	user.LastUTMCampaign = util.FirstNonEmpty(user.LastUTMCampaign, cleanCmp)
	user.LastUTMContent = util.FirstNonEmpty(user.LastUTMContent, cleanCnt)
	user.LastUTMTerm = util.FirstNonEmpty(user.LastUTMTerm, cleanTrm)
}

// ctwaAttribution decide se a venda vai como business_messaging (CTWA), website ou system_generated.
// Preserva atribuição CTWA mesmo quando o cliente abriu o link de pagamento (sourceURL != ""),
// desde que não haja fbc de campanha web sobreposto e exista page_id resolvido.
func (h *Webhooks) ctwaAttribution(ctx context.Context, accountID int64, pixel *domain.Pixel, user *domain.TrackedUser, fbc, sourceURL string) (actionSource, channel, ctwaClid, pageID string) {
	if user != nil && user.CtwaClid != "" && fbc == "" {
		var pid string
		if h.whatsapp != nil {
			pid = h.whatsapp.FindPageIDByCtwaClid(ctx, accountID, user.CtwaClid)
			if pid == "" && user.ID > 0 {
				pid = h.whatsapp.FindPageIDByTrackedUserID(ctx, accountID, user.ID)
			}
		}
		if pid == "" && h.metaAcc != nil && pixel != nil && pixel.MetaAccountID != nil && *pixel.MetaAccountID > 0 {
			if mAcc, merr := h.metaAcc.FindByID(ctx, *pixel.MetaAccountID); merr == nil && mAcc != nil && mAcc.AccountID == accountID && mAcc.Enabled && mAcc.IsConnected() {
				pid = mAcc.DefaultPageID
			}
		}
		if pid != "" {
			return "business_messaging", "whatsapp", user.CtwaClid, pid
		}
	}
	if sourceURL != "" {
		return "website", "", "", ""
	}
	return "system_generated", "", "", ""
}
