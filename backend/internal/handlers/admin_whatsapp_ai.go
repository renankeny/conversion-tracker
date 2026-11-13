package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/whatsappai"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

type AdminWhatsAppAI struct {
	ai       *repo.WhatsAppAIRepo
	users    *repo.AdminUserRepo
	analyzer *whatsappai.Analyzer
	rdb      *redis.Client
}

func NewAdminWhatsAppAI(
	ai *repo.WhatsAppAIRepo,
	users *repo.AdminUserRepo,
	analyzer *whatsappai.Analyzer,
	rdb *redis.Client,
) *AdminWhatsAppAI {
	return &AdminWhatsAppAI{ai: ai, users: users, analyzer: analyzer, rdb: rdb}
}

func (h *AdminWhatsAppAI) GetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accID := TenantFromContext(ctx).EffectiveAccountID(0)
	var userID int64
	if claims, ok := auth.ClaimsFrom(ctx); ok {
		userID = claims.UserID
	}

	s, err := h.ai.GetSettings(ctx, accID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.HasOpenAIKey = h.ai.ResolveOpenAIKey(ctx, accID, userID) != ""
	s.DefaultPrompt = whatsappai.DefaultBasePrompt
	if strings.TrimSpace(s.BasePrompt) == "" {
		s.BasePrompt = whatsappai.DefaultBasePrompt
	}
	writeJSON(w, http.StatusOK, s)
}

type updateAISettingsInput struct {
	Enabled          bool                   `json:"enabled"`
	Model            string                 `json:"model"`
	QualifiedStageID *int64                 `json:"qualified_stage_id"`
	SaleStageID      *int64                 `json:"sale_stage_id"`
	BasePrompt       string                 `json:"base_prompt"`
	LearnedRules     []domain.AILearnedRule `json:"learned_rules"`
}

func (h *AdminWhatsAppAI) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accID := TenantFromContext(ctx).EffectiveAccountID(0)
	var userID int64
	if claims, ok := auth.ClaimsFrom(ctx); ok {
		if claims.Role == "employee" {
			writeError(w, http.StatusForbidden, "apenas administradores podem alterar configurações da IA")
			return
		}
		userID = claims.UserID
	}

	var in updateAISettingsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}

	existing, err := h.ai.GetSettings(ctx, accID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rules := existing.LearnedRules
	if in.LearnedRules != nil {
		rules = in.LearnedRules
	}

	var qualID, saleID *int64
	if in.QualifiedStageID != nil && *in.QualifiedStageID > 0 {
		qualID = in.QualifiedStageID
	}
	if in.SaleStageID != nil && *in.SaleStageID > 0 {
		saleID = in.SaleStageID
	}

	promptVal := strings.TrimSpace(in.BasePrompt)
	if promptVal == strings.TrimSpace(whatsappai.DefaultBasePrompt) {
		promptVal = ""
	}

	next := &domain.AccountAISettings{
		AccountID:        accID,
		Enabled:          in.Enabled,
		Model:            in.Model,
		QualifiedStageID: qualID,
		SaleStageID:      saleID,
		BasePrompt:       promptVal,
		LearnedRules:     rules,
	}
	if err := h.ai.UpsertSettings(ctx, next); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar configurações da IA: "+err.Error())
		return
	}

	saved, _ := h.ai.GetSettings(ctx, accID)
	saved.HasOpenAIKey = h.ai.ResolveOpenAIKey(ctx, accID, userID) != ""
	saved.DefaultPrompt = whatsappai.DefaultBasePrompt
	if strings.TrimSpace(saved.BasePrompt) == "" {
		saved.BasePrompt = whatsappai.DefaultBasePrompt
	}
	writeJSON(w, http.StatusOK, saved)
}

func (h *AdminWhatsAppAI) RunNow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accID := TenantFromContext(ctx).EffectiveAccountID(0)
	var userID int64
	if claims, ok := auth.ClaimsFrom(ctx); ok {
		if claims.Role == "employee" {
			writeError(w, http.StatusForbidden, "permissão insuficiente")
			return
		}
		userID = claims.UserID
	}
	if h.analyzer == nil {
		writeError(w, http.StatusServiceUnavailable, "analisador indisponível")
		return
	}
	sum, err := h.analyzer.AnalyzeAccount(ctx, accID, userID, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

type contestAIEventInput struct {
	Reason      string `json:"reason"`
	AddToPrompt *bool  `json:"add_to_prompt"`
}

func (h *AdminWhatsAppAI) ContestEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accID := TenantFromContext(ctx).EffectiveAccountID(0)
	eventID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || eventID <= 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var userID int64
	authorEmail := ""
	if claims, ok := auth.ClaimsFrom(ctx); ok {
		userID = claims.UserID
		if u, uerr := h.users.FindByID(ctx, userID); uerr == nil && u != nil {
			authorEmail = u.Email
		}
	}

	var in contestAIEventInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		writeError(w, http.StatusBadRequest, "informe o motivo da contestação")
		return
	}
	addRule := true
	if in.AddToPrompt != nil {
		addRule = *in.AddToPrompt
	}

	ev, err := h.ai.ContestAIEvent(ctx, accID, eventID, userID, reason, addRule, authorEmail)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "evento de IA não encontrado ou já contestado")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if ev.DispatchedEventID != "" {
		h.ai.RevokeDispatchedAIEvent(ctx, accID, ev.DispatchedEventID)
	}
	if ev.EventType == domain.AIEventQualified {
		_ = h.ai.SetChatAIQualified(ctx, ev.ChatID, false)
	}
	crmReverted := false
	if ev.CRMLeadID != nil && *ev.CRMLeadID > 0 {
		crmReverted = h.ai.RevertCRMLeadStage(ctx, accID, *ev.CRMLeadID, ev.CRMNewStageID, ev.CRMPrevStageID, ev.CRMPrevValue)
	}

	if h.rdb != nil {
		cache.BumpDashboardVersion(ctx, h.rdb, accID)
		wshub.PublishCustom(ctx, h.rdb, accID, "whatsapp_ai_event", map[string]any{
			"chat_id":         ev.ChatID,
			"crm_lead_id":     ev.CRMLeadID,
			"tracked_user_id": ev.TrackedUserID,
			"ai_event":        ev,
		})
		if crmReverted {
			wshub.PublishCustom(ctx, h.rdb, accID, "crm_updated", map[string]any{
				"action":  "whatsapp_ai_contested",
				"lead_id": ev.CRMLeadID,
			})
		}
	}
	writeJSON(w, http.StatusOK, ev)
}

