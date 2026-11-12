package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
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

func parseChatPagination(r *http.Request) (int64, int) {
	var beforeID int64
	if s := r.URL.Query().Get("before_id"); s != "" {
		beforeID, _ = strconv.ParseInt(s, 10, 64)
	}
	limit := 20
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	return beforeID, limit
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

	beforeID, limit := parseChatPagination(r)
	msgs, hasMore, nextBeforeID, err := h.repo.ListMessagesPaginated(r.Context(), chat.ID, beforeID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.signMessages(accID, msgs)
	writeJSON(w, http.StatusOK, map[string]any{
		"chat":           chat,
		"messages":       msgs,
		"has_more":       hasMore,
		"next_before_id": nextBeforeID,
	})
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
		writeJSON(w, http.StatusOK, map[string]any{
			"chat":           nil,
			"messages":       []any{},
			"has_more":       false,
			"next_before_id": nil,
		})
		return
	}

	beforeID, limit := parseChatPagination(r)
	msgs, hasMore, nextBeforeID, err := h.repo.ListMessagesPaginated(r.Context(), chat.ID, beforeID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.signMessages(accID, msgs)
	writeJSON(w, http.StatusOK, map[string]any{
		"chat":           chat,
		"messages":       msgs,
		"has_more":       hasMore,
		"next_before_id": nextBeforeID,
	})
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
		writeJSON(w, http.StatusOK, map[string]any{
			"chat":           nil,
			"messages":       []any{},
			"has_more":       false,
			"next_before_id": nil,
		})
		return
	}

	beforeID, limit := parseChatPagination(r)
	msgs, hasMore, nextBeforeID, err := h.repo.ListMessagesPaginated(r.Context(), chat.ID, beforeID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.signMessages(accID, msgs)
	writeJSON(w, http.StatusOK, map[string]any{
		"chat":           chat,
		"messages":       msgs,
		"has_more":       hasMore,
		"next_before_id": nextBeforeID,
	})
}
