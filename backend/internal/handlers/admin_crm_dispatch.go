package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// dispatchCRMStageTrigger enfileira o evento de conversão da etapa do CRM respeitando:
// - a regra de 1ª ou 3ª mensagem CTWA na Etapa 0 (order_index == 0);
// - o valor sentinela "none" (desativado em CTWA);
// - deduplicação cruzada de Lead/LeadSubmitted e Purchase contra webhooks, IA e gatilhos;
// - atribuição correta entre CTWA (business_messaging), LP->WhatsApp (website com URL) e manual (system_generated).
func (h *AdminCRM) dispatchCRMStageTrigger(
	ctx context.Context,
	accountID int64,
	leadID int64,
	stage *domain.CRMStage,
	tu *domain.TrackedUser,
	eventVal float64,
) {
	if h.queue == nil || stage == nil || tu == nil {
		return
	}
	if h.users != nil {
		h.users.EnrichGroupFields(ctx, tu)
	}

	isWppOrCTWA := domain.IsWhatsAppOrCTWAUser(tu)
	targetEvent := domain.EffectiveStageTrigger(stage, isWppOrCTWA)
	if targetEvent == "" {
		return
	}
	if targetEvent == domain.EventPurchase && eventVal <= 0 {
		return
	}

	// Na Etapa 0 ("Novo Lead"), nunca fura a regra da 1ª ou 3ª mensagem da instância WhatsApp.
	if stage.OrderIndex == 0 && isWppOrCTWA && h.whatsapp != nil && tu.ID > 0 {
		if hasChat, reached := h.whatsapp.CheckCtwaInboundThreshold(ctx, accountID, tu.ID); hasChat && !reached {
			return
		}
	}

	resolvedPixel, pixelName := h.resolveCRMDispatchPixel(ctx, accountID, tu)
	if pixelName == "" || pixelName == "api_import" {
		return
	}

	isLeadEv := targetEvent == domain.EventLead || targetEvent == domain.EventLeadSubmitted || targetEvent == domain.EventQualifiedLead
	if targetEvent == domain.EventPurchase {
		if cache.HasInFlightMatchingPurchase(ctx, h.rdb, accountID, pixelName, tu.GroupID, tu.ID, tu.ExternalID, tu.Phone, tu.Email, "", true, eventVal, time.Now()) {
			slog.Info("skipping duplicate crm Purchase already in-flight", "pixel", pixelName, "lead_id", leadID)
			return
		}
	}
	if h.sentEvents != nil {
		if isLeadEv {
			if already, _ := h.sentEvents.HasLeadConversionSent(ctx, accountID, tu.ID, tu.ExternalID, pixelName); already {
				return
			}
		} else if targetEvent == domain.EventPurchase {
			if dup, _ := h.sentEvents.HasMatchingPurchaseSent(ctx, accountID, tu.ID, tu.ExternalID, pixelName, eventVal, time.Now()); dup {
				slog.Info("skipping duplicate crm Purchase already sent by webhook/ai/trigger", "pixel", pixelName, "lead_id", leadID)
				return
			}
		}
	}

	sourceURL := util.FirstNonEmpty(tu.URL, tu.FirstURL)
	actionSource := "website"
	msgChannel := ""
	sendClid := ""
	pageID := ""
	if tu.CtwaClid != "" {
		pageID = h.resolveCRMPageID(ctx, accountID, tu, resolvedPixel)
		if pageID != "" {
			actionSource = "business_messaging"
			msgChannel = "whatsapp"
			sendClid = tu.CtwaClid
			sourceURL = ""
		} else if sourceURL == "" {
			actionSource = "system_generated"
		}
	} else if sourceURL == "" {
		actionSource = "system_generated"
	}

	if actionSource != "business_messaging" && targetEvent == domain.EventLeadSubmitted {
		targetEvent = domain.EventLead
	}

	var valuePtr *float64
	if eventVal > 0 {
		v := eventVal
		valuePtr = &v
	}

	currency := "BRL"
	gateway := "crm"
	if h.whatsapp != nil {
		var pxID int64
		if resolvedPixel != nil {
			pxID = resolvedPixel.ID
		}
		currency = h.whatsapp.ResolveCurrencyForLead(ctx, accountID, tu.ID, pxID)
		if isWppOrCTWA {
			gateway = "whatsapp_crm"
		} else if chat, _ := h.whatsapp.GetChatByTrackedUser(ctx, accountID, tu.ID); chat != nil {
			gateway = "whatsapp_crm"
		}
	}

	eventID := "crm_pur:" + util.GenerateID10()
	if leadID > 0 && targetEvent != domain.EventPurchase {
		eventID = fmt.Sprintf("crm:%d:%d:%s", leadID, stage.ID, targetEvent)
		if h.sentEvents != nil {
			if exists, _ := h.sentEvents.ExistsSent(ctx, accountID, eventID, pixelName, ""); exists {
				return
			}
		}
	}

	in := domain.SendEventInput{
		PixelName:             pixelName,
		EventName:             targetEvent,
		EventID:               eventID,
		EventTime:             time.Now().Unix(),
		SourceURL:             sourceURL,
		Gateway:               gateway,
		ActionSource:          actionSource,
		MessagingChannel:      msgChannel,
		CtwaClid:              sendClid,
		PageID:                pageID,
		ExternalID:            tu.ExternalID,
		IDWpp:                 tu.IDWpp,
		ClientIP:              tu.ClientIP,
		ClientUserAgent:       tu.ClientUserAgent,
		FBP:                   tu.FBP,
		FBC:                   tu.FBC,
		GCLID:                 tu.GCLID,
		WBraid:                tu.WBraid,
		GBraid:                tu.GBraid,
		TTClid:                tu.TTClid,
		TTP:                   tu.TTP,
		GA4ClientID:           tu.GA4ClientID,
		GA4SessionID:          tu.GA4SessionID,
		Email:                 tu.Email,
		Phone:                 tu.Phone,
		FirstName:             tu.FirstName,
		LastName:              tu.LastName,
		Country:               tu.Country,
		State:                 tu.State,
		City:                  tu.City,
		PostalCode:            tu.PostalCode,
		Value:                 valuePtr,
		Currency:              currency,
		OverrideGadsAccountID: stage.GoogleAdsAccountID,
		OverrideGadsActionID:  stage.GoogleAdsActionID,
		CRMSynced:             true,
	}

	if err := h.queue.EnqueueSendEvent(ctx, in); err != nil {
		slog.Error("dispatch crm stage trigger enqueue failed", "err", err, "pixel", pixelName, "event", targetEvent)
	} else {
		if targetEvent == domain.EventPurchase {
			cache.RecordInFlightPurchase(ctx, h.rdb, accountID, pixelName, tu.GroupID, tu.ID, tu.ExternalID, tu.Phone, tu.Email, cache.InFlightPurchase{
				EventID: in.EventID, IsNonOrder: true, Value: eventVal,
			})
		}
		slog.Info("dispatch crm stage trigger enqueued ok", "pixel", pixelName, "event", targetEvent, "event_id", in.EventID)
	}
}
