package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func (h *PublicDashboard) SetPortalDeps(
	crmH *AdminCRM,
	wa *repo.WhatsAppRepo,
	ai *repo.WhatsAppAIRepo,
	jwtSecret, publicAPIURL string,
) {
	h.crmH = crmH
	h.whatsapp = wa
	h.aiRepo = ai
	h.jwtSecret = jwtSecret
	if publicAPIURL != "" {
		h.publicAPIURL = publicAPIURL
	}
}

func (h *PublicDashboard) signPortalMessages(accID int64, msgs []*domain.WhatsAppMessage) {
	for _, m := range msgs {
		queue.SignMessageStreamIfReady(m, h.publicAPIURL, h.jwtSecret, accID)
	}
}

func (h *PublicDashboard) listPortalAIEvents(r *http.Request, chatID int64) []*domain.WhatsAppAIEvent {
	if h.aiRepo == nil || chatID <= 0 {
		return []*domain.WhatsAppAIEvent{}
	}
	evs, err := h.aiRepo.ListAIEventsByChat(r.Context(), chatID)
	if err != nil || evs == nil {
		return []*domain.WhatsAppAIEvent{}
	}
	return evs
}

// Conversations GET /public/dashboard/{token}/conversations
func (h *PublicDashboard) Conversations(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	if !p.DashboardConfig.IsPublicConversationsEnabled() {
		writeError(w, http.StatusForbidden, "aba de conversas desativada para este link")
		return
	}
	if h.whatsapp == nil {
		writeError(w, http.StatusServiceUnavailable, "serviço indisponível")
		return
	}

	filter := parseConversationAuditFilter(r, p.AccountID)
	filter.PixelID = p.ID
	filter.PixelName = p.Name

	res, err := h.whatsapp.ListAuditConversations(r.Context(), filter)
	if err != nil {
		slog.Error("public portal list conversations failed", "pixel", p.Name, "error", err)
		writeError(w, http.StatusInternalServerError, "erro ao carregar conversas")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ConversationMessages GET /public/dashboard/{token}/chats/{chatID}/messages
func (h *PublicDashboard) ConversationMessages(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	if !p.DashboardConfig.IsPublicConversationsEnabled() && !p.DashboardConfig.IsPublicCRMEnabled() {
		writeError(w, http.StatusForbidden, "visualização de conversas desativada para este link")
		return
	}
	if h.whatsapp == nil {
		writeError(w, http.StatusServiceUnavailable, "serviço indisponível")
		return
	}

	chatID, err := strconv.ParseInt(chi.URLParam(r, "chatID"), 10, 64)
	if err != nil || chatID <= 0 {
		writeError(w, http.StatusBadRequest, "identificador de conversa inválido")
		return
	}

	chat, err := h.whatsapp.GetChatByIDForPixel(r.Context(), p.AccountID, p.ID, p.Name, chatID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar conversa")
		return
	}
	if chat == nil {
		writeError(w, http.StatusNotFound, "conversa não encontrada")
		return
	}

	beforeID, limit := parseChatPagination(r)
	msgs, hasMore, nextBeforeID, err := h.whatsapp.ListMessagesPaginated(r.Context(), chat.ID, beforeID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar mensagens")
		return
	}
	h.signPortalMessages(p.AccountID, msgs)
	writeJSON(w, http.StatusOK, map[string]any{
		"chat":           chat,
		"messages":       msgs,
		"ai_events":      h.listPortalAIEvents(r, chat.ID),
		"has_more":       hasMore,
		"next_before_id": nextBeforeID,
	})
}

// CRMLeadChat GET /public/dashboard/{token}/crm/leads/{leadID}/chat
func (h *PublicDashboard) CRMLeadChat(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, false)
	if !ok {
		return
	}
	leadID, err := strconv.ParseInt(chi.URLParam(r, "leadID"), 10, 64)
	if err != nil || leadID <= 0 {
		writeError(w, http.StatusBadRequest, "lead inválido")
		return
	}
	belongs, err := h.crmH.crmRepo.LeadBelongsToPixel(r.Context(), leadID, p.AccountID, p.ID, p.Name)
	if err != nil || !belongs {
		writeError(w, http.StatusNotFound, "lead não encontrado")
		return
	}

	chat, err := h.whatsapp.GetChatByCRMLead(r.Context(), p.AccountID, leadID)
	if err != nil || chat == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"chat": nil, "messages": []any{}, "ai_events": []any{}, "has_more": false, "next_before_id": nil,
		})
		return
	}

	beforeID, limit := parseChatPagination(r)
	msgs, hasMore, nextBeforeID, err := h.whatsapp.ListMessagesPaginated(r.Context(), chat.ID, beforeID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar mensagens")
		return
	}
	h.signPortalMessages(p.AccountID, msgs)
	writeJSON(w, http.StatusOK, map[string]any{
		"chat":           chat,
		"messages":       msgs,
		"ai_events":      h.listPortalAIEvents(r, chat.ID),
		"has_more":       hasMore,
		"next_before_id": nextBeforeID,
	})
}
