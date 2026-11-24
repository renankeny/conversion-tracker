package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

var ctwaConvertibleEvents = map[string]bool{
	domain.EventPurchase:         true,
	domain.EventLead:             true,
	domain.EventLeadSubmitted:    true,
	domain.EventInitiateCheckout: true,
	domain.EventAddToCart:        true,
	domain.EventViewContent:      true,
	domain.EventContact:          true,
	domain.EventSchedule:         true,
	domain.EventOrderCreated:     true,
	domain.EventOrderShipped:     true,
	domain.EventOrderDelivered:   true,
	domain.EventOrderCanceled:    true,
	domain.EventOrderReturned:    true,
	domain.EventCartAbandoned:    true,
	domain.EventQualifiedLead:    true,
}

func (h *AdminWhatsAppChats) Convert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatID, _ := strconv.ParseInt(chi.URLParam(r, "chatID"), 10, 64)
	accID := TenantFromContext(ctx).EffectiveAccountID(0)

	var body struct {
		EventName string   `json:"event_name"`
		Value     *float64 `json:"value,omitempty"`
		Currency  string   `json:"currency,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !ctwaConvertibleEvents[body.EventName] {
		writeError(w, http.StatusBadRequest, "evento não suportado para conversão WhatsApp")
		return
	}
	if body.EventName == domain.EventPurchase && (body.Value == nil || *body.Value <= 0) {
		writeError(w, http.StatusBadRequest, "valor obrigatório (> 0) para Purchase")
		return
	}

	cctx, err := h.repo.GetChatConversionContext(ctx, accID, chatID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cctx == nil {
		writeError(w, http.StatusNotFound, "chat not found")
		return
	}

	var u *domain.TrackedUser
	if h.usersRepo != nil && cctx.TrackedUserID > 0 {
		if found, uerr := h.usersRepo.FindByID(ctx, cctx.TrackedUserID); uerr == nil {
			u = found
		}
	}

	ctwaClid := cctx.CtwaClid
	pageID := cctx.PageID
	if ctwaClid != "" && pageID == "" {
		pageID = h.repo.FindPageIDByCtwaClid(ctx, accID, ctwaClid)
		if pageID == "" && cctx.TrackedUserID > 0 {
			pageID = h.repo.FindPageIDByTrackedUserID(ctx, accID, cctx.TrackedUserID)
		}
		if pageID == "" && h.pixelsRepo != nil && h.metaAccRepo != nil {
			if px, perr := h.pixelsRepo.FindByName(ctx, cctx.PixelName); perr == nil && px != nil && px.MetaAccountID != nil && *px.MetaAccountID > 0 {
				if mAcc, merr := h.metaAccRepo.FindByID(ctx, *px.MetaAccountID); merr == nil && mAcc != nil && mAcc.AccountID == accID && mAcc.Enabled && mAcc.IsConnected() {
					pageID = mAcc.DefaultPageID
				}
			}
		}
	}

	sourceURL := ""
	if u != nil {
		sourceURL = util.FirstNonEmpty(u.URL, u.FirstURL)
	}
	actionSource := "website"
	msgChannel := ""
	eventName := body.EventName
	if ctwaClid != "" && pageID != "" {
		actionSource = "business_messaging"
		msgChannel = "whatsapp"
		sourceURL = ""
		if eventName == domain.EventLead {
			eventName = domain.EventLeadSubmitted
		}
	} else {
		ctwaClid = ""
		pageID = ""
		if sourceURL == "" {
			actionSource = "system_generated"
		}
		if eventName == domain.EventLeadSubmitted || eventName == domain.EventQualifiedLead {
			eventName = domain.EventLead
		}
	}

	var uGroupID *int64
	var uEmail string
	if u != nil {
		uGroupID = u.GroupID
		uEmail = u.Email
	}
	if cctx.TrackedUserID > 0 {
		if eventName == domain.EventPurchase && body.Value != nil {
			now := time.Now()
			dupInFlight := cache.HasInFlightMatchingPurchase(ctx, h.rdb, accID, cctx.PixelName, uGroupID, cctx.TrackedUserID, cctx.ExternalID, cctx.Phone, uEmail, "", true, *body.Value, now)
			dupDB := false
			if !dupInFlight && h.aiRepo != nil {
				dupDB, _ = h.aiRepo.HasMatchingPurchaseSent(ctx, accID, cctx.TrackedUserID, cctx.PixelName, *body.Value, now)
			}
			if dupInFlight || dupDB {
				if h.crmRepo != nil {
					if leadID, synced := h.crmRepo.SyncOrCreateLeadOnPurchase(ctx, accID, cctx.TrackedUserID, *body.Value, false); synced && h.rdb != nil {
						wshub.PublishCustom(ctx, h.rdb, accID, "crm_updated", map[string]any{"action": "whatsapp_chat_converted", "lead_id": leadID, "event": eventName})
					}
				}
				writeJSON(w, http.StatusOK, map[string]any{"queued": false, "deduplicated": true})
				return
			}
		} else if h.aiRepo != nil && (eventName == domain.EventLead || eventName == domain.EventLeadSubmitted || eventName == domain.EventQualifiedLead) {
			if dup, _ := h.aiRepo.HasLeadConversionSent(ctx, accID, cctx.TrackedUserID, cctx.PixelName); dup {
				writeJSON(w, http.StatusOK, map[string]any{"queued": false, "deduplicated": true})
				return
			}
		}
	}

	if h.crmRepo != nil && cctx.TrackedUserID > 0 {
		var leadID int64
		var synced bool
		if eventName == domain.EventPurchase && body.Value != nil {
			leadID, synced = h.crmRepo.SyncOrCreateLeadOnPurchase(ctx, accID, cctx.TrackedUserID, *body.Value, true)
		} else {
			leadID, synced = h.crmRepo.SyncLeadFromEventTrigger(ctx, accID, cctx.TrackedUserID, eventName, body.Value)
		}
		if synced && h.rdb != nil {
			wshub.PublishCustom(ctx, h.rdb, accID, "crm_updated", map[string]any{"action": "whatsapp_chat_converted", "lead_id": leadID, "event": eventName})
		}
	}

	in := domain.SendEventInput{
		PixelName:        cctx.PixelName,
		EventName:        eventName,
		EventID:          "wpp_manual_" + util.GenerateID10(),
		EventTime:        time.Now().Unix(),
		SourceURL:        sourceURL,
		Gateway:          "whatsapp_manual",
		ActionSource:     actionSource,
		MessagingChannel: msgChannel,
		CtwaClid:         ctwaClid,
		PageID:           pageID,
		ExternalID:       cctx.ExternalID,
		Phone:            cctx.Phone,
		FirstName:        cctx.FirstName,
		Value:            body.Value,
		Currency:         domain.NormalizeWhatsAppCurrency(util.FirstNonEmpty(body.Currency, cctx.Currency)),
		CRMSynced:        true,
	}
	if err := h.enqueuer.EnqueueSendEvent(ctx, in); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if eventName == domain.EventPurchase && body.Value != nil {
		cache.RecordInFlightPurchase(ctx, h.rdb, accID, cctx.PixelName, uGroupID, cctx.TrackedUserID, cctx.ExternalID, cctx.Phone, uEmail, cache.InFlightPurchase{
			EventID: in.EventID, IsNonOrder: true, Value: *body.Value,
		})
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"queued": true, "event_id": in.EventID})
}
