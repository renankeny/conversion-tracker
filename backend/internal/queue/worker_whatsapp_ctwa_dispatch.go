package queue

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func shouldDispatchCTWALead(minInbound, currentInbound int, isInbound bool) bool {
	if !isInbound {
		return false
	}
	req := domain.NormalizeCtwaMinMessages(minInbound)
	return currentInbound >= req
}

// resolveFirstStageCTWAEvent lê o evento configurado na 1ª etapa do funil CRM (order_index = 0).
// Respeita "none" (desativado pelo cliente) e mantém fallback para LeadSubmitted em contas legadas.
func resolveFirstStageCTWAEvent(ctx context.Context, deps Deps, accountID int64) (string, bool) {
	if deps.CRM == nil || accountID <= 0 {
		return domain.EventLeadSubmitted, true
	}
	stage, err := deps.CRM.GetFirstStage(ctx, accountID)
	if err != nil || stage == nil {
		return domain.EventLeadSubmitted, true
	}
	if strings.EqualFold(strings.TrimSpace(stage.TriggerEventCTWA), domain.CRMTriggerNone) {
		return "", false
	}
	if ev := domain.EffectiveStageTrigger(stage, true); ev != "" && ev != domain.EventPurchase {
		return ev, true
	}
	return domain.EventLeadSubmitted, true
}

func maybeDispatchCTWALeadSubmitted(
	ctx context.Context,
	deps Deps,
	inst *domain.WhatsAppInstance,
	pixel *domain.Pixel,
	u *domain.TrackedUser,
	chatID int64,
	phone, sourceID, ctwaClid, pageID string,
	isInbound bool,
) error {
	if pixel == nil || !isInbound || deps.Enqueuer == nil || u == nil || (ctwaClid == "" && sourceID == "") {
		return nil
	}

	minReq := inst.EffectiveCtwaMinMessages()
	if minReq > 1 {
		inboundCount, err := deps.WhatsApp.CountInboundMessages(ctx, chatID)
		if err != nil {
			return fmt.Errorf("count inbound messages for ctwa lead: %w", err)
		}
		if !shouldDispatchCTWALead(minReq, inboundCount, isInbound) {
			return nil
		}
	}

	eventName, enabled := resolveFirstStageCTWAEvent(ctx, deps, inst.AccountID)
	if !enabled {
		return nil
	}

	if deps.SentEvents != nil {
		if already, _ := deps.SentEvents.HasLeadConversionSent(ctx, pixel.AccountID, u.ID, u.ExternalID, pixel.Name); already {
			return nil
		}
	}

	if ctwaClid != "" && pageID == "" && deps.WhatsApp != nil {
		pageID = deps.WhatsApp.FindPageIDByCtwaClid(ctx, inst.AccountID, ctwaClid)
		if pageID == "" && u.ID > 0 {
			pageID = deps.WhatsApp.FindPageIDByTrackedUserID(ctx, inst.AccountID, u.ID)
		}
	}
	if ctwaClid != "" && pageID == "" {
		if oauthAcc, _ := resolvePixelMetaOAuth(ctx, deps, pixel); oauthAcc != nil && oauthAcc.DefaultPageID != "" {
			pageID = oauthAcc.DefaultPageID
		}
	}

	actionSource := "business_messaging"
	messagingChannel := "whatsapp"
	eventID := "ctwa:" + ctwaClid
	sendClid := ctwaClid
	if ctwaClid == "" {
		actionSource = "system_generated"
		messagingChannel = ""
		eventID = fmt.Sprintf("ctwa_ad:%s:%d", sourceID, chatID)
	} else if pageID == "" {
		actionSource = "system_generated"
		messagingChannel = ""
		sendClid = ""
		eventID = fmt.Sprintf("ctwa_nopage:%s", ctwaClid)
	}

	if deps.SentEvents != nil {
		if exists, _ := deps.SentEvents.ExistsSent(ctx, pixel.AccountID, eventID, pixel.Name, ""); exists {
			return nil
		}
	}
	if claimed, redisOK := cache.ClaimEventDedup(ctx, deps.Redis, pixel.AccountID, pixel.Name, eventID, cache.EventDedupTTL); redisOK && !claimed {
		return nil
	}

	if deps.Users != nil {
		deps.Users.EnrichGroupFields(ctx, u)
	}

	if err := deps.Enqueuer.EnqueueSendEvent(ctx, domain.SendEventInput{
		PixelName:        pixel.Name,
		EventName:        eventName,
		EventID:          eventID,
		EventTime:        time.Now().Unix(),
		Gateway:          "whatsapp",
		ActionSource:     actionSource,
		MessagingChannel: messagingChannel,
		CtwaClid:         sendClid,
		PageID:           pageID,
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
		Currency:         domain.NormalizeWhatsAppCurrency(inst.Currency),
	}); err != nil {
		cache.ReleaseEventDedup(ctx, deps.Redis, pixel.AccountID, pixel.Name, eventID)
		slog.Error("enqueue ctwa lead event failed", "pixel_name", pixel.Name, "event_id", eventID, "error", err)
		return fmt.Errorf("enqueue ctwa lead event: %w", err)
	}
	return nil
}
