package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type AdminWhatsAppChats struct {
	repo         *repo.WhatsAppRepo
	enqueuer     *queue.Enqueuer
	publicAPIURL string
	jwtSecret    string
}

func NewAdminWhatsAppChats(repo *repo.WhatsAppRepo, enqueuer *queue.Enqueuer, publicAPIURL, jwtSecret string) *AdminWhatsAppChats {
	return &AdminWhatsAppChats{
		repo:         repo,
		enqueuer:     enqueuer,
		publicAPIURL: publicAPIURL,
		jwtSecret:    jwtSecret,
	}
}

func (h *AdminWhatsAppChats) signMessages(accID int64, msgs []*domain.WhatsAppMessage) {
	for _, m := range msgs {
		queue.SignMessageStreamIfReady(m, h.publicAPIURL, h.jwtSecret, accID)
	}
}

func (h *AdminWhatsAppChats) ListChats(w http.ResponseWriter, r *http.Request) {
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	chats, err := h.repo.ListChats(r.Context(), accID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": chats})
}

func (h *AdminWhatsAppChats) GetMessages(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseInt(chi.URLParam(r, "chatID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid chatID")
		return
	}
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)

	chat, err := h.repo.GetChatByID(r.Context(), accID, chatID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chat == nil {
		writeError(w, http.StatusNotFound, "chat not found")
		return
	}
	msgs, err := h.repo.ListMessages(r.Context(), chat.ID, 300)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.signMessages(accID, msgs)
	writeJSON(w, http.StatusOK, map[string]any{"chat": chat, "messages": msgs})
}

func (h *AdminWhatsAppChats) GetByTrackedUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid userID")
		return
	}
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)

	chat, err := h.repo.GetChatByTrackedUser(r.Context(), accID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chat == nil {
		writeJSON(w, http.StatusOK, map[string]any{"chat": nil, "messages": []any{}})
		return
	}
	msgs, err := h.repo.ListMessages(r.Context(), chat.ID, 300)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.signMessages(accID, msgs)
	writeJSON(w, http.StatusOK, map[string]any{"chat": chat, "messages": msgs})
}

func (h *AdminWhatsAppChats) GetByCRMLead(w http.ResponseWriter, r *http.Request) {
	leadID, err := strconv.ParseInt(chi.URLParam(r, "leadID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid leadID")
		return
	}
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)

	chat, err := h.repo.GetChatByCRMLead(r.Context(), accID, leadID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chat == nil {
		writeJSON(w, http.StatusOK, map[string]any{"chat": nil, "messages": []any{}})
		return
	}
	msgs, err := h.repo.ListMessages(r.Context(), chat.ID, 300)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.signMessages(accID, msgs)
	writeJSON(w, http.StatusOK, map[string]any{"chat": chat, "messages": msgs})
}

var ctwaConvertibleEvents = map[string]bool{
	domain.EventPurchase:         true,
	domain.EventLeadSubmitted:    true,
	domain.EventInitiateCheckout: true,
	domain.EventAddToCart:        true,
	domain.EventViewContent:      true,
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
		writeError(w, http.StatusBadRequest, "evento não suportado em business_messaging")
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
	if cctx.CtwaClid == "" {
		writeError(w, http.StatusBadRequest, "este lead não possui ctwa_clid de anúncio Click-to-WhatsApp")
		return
	}

	in := domain.SendEventInput{
		PixelName:        cctx.PixelName,
		EventName:        body.EventName,
		EventID:          util.GenerateID10(),
		EventTime:        time.Now().Unix(),
		ActionSource:     "business_messaging",
		MessagingChannel: "whatsapp",
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
