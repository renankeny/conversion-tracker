package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

var ctwaConvertibleEvents = map[string]bool{
	domain.EventPurchase:         true,
	domain.EventLead:             true,
	domain.EventLeadSubmitted:    true,
	domain.EventInitiateCheckout: true,
	domain.EventAddToCart:        true,
	domain.EventViewContent:      true,
	domain.EventContact:          true,
	"OrderCreated":               true,
	"OrderShipped":               true,
	"OrderDelivered":             true,
	"OrderCanceled":              true,
	"OrderReturned":              true,
	"CartAbandoned":              true,
	"QualifiedLead":              true,
}

func (h *AdminWhatsAppChats) Convert(w http.ResponseWriter, r *http.Request) {
	chatID, _ := strconv.ParseInt(chi.URLParam(r, "chatID"), 10, 64)
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)

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

	cctx, err := h.repo.GetChatConversionContext(r.Context(), accID, chatID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cctx == nil {
		writeError(w, http.StatusNotFound, "chat not found")
		return
	}

	actionSource := "website"
	msgChannel := ""
	eventName := body.EventName
	if cctx.CtwaClid != "" {
		actionSource = "business_messaging"
		msgChannel = "whatsapp"
		if eventName == domain.EventLead {
			eventName = domain.EventLeadSubmitted
		}
	} else if eventName == domain.EventLeadSubmitted {
		eventName = domain.EventLead
	}

	in := domain.SendEventInput{
		PixelName:        cctx.PixelName,
		EventName:        eventName,
		EventID:          util.GenerateID10(),
		EventTime:        time.Now().Unix(),
		Gateway:          "whatsapp",
		ActionSource:     actionSource,
		MessagingChannel: msgChannel,
		CtwaClid:         cctx.CtwaClid,
		PageID:           cctx.PageID,
		ExternalID:       cctx.ExternalID,
		Phone:            cctx.Phone,
		FirstName:        cctx.FirstName,
		Value:            body.Value,
		Currency:         util.FirstNonEmpty(body.Currency, "BRL"),
	}
	if err := h.enqueuer.EnqueueSendEvent(r.Context(), in); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"queued": true, "event_id": in.EventID})
}

