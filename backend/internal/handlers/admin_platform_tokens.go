package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type PlatformTokens struct {
	tokens      *repo.PlatformTokenRepo
	webhookLogs *repo.WebhookLogRepo
}

func NewPlatformTokens(tokens *repo.PlatformTokenRepo) *PlatformTokens {
	return &PlatformTokens{tokens: tokens}
}

func (h *PlatformTokens) SetWebhookLogs(wl *repo.WebhookLogRepo) {
	h.webhookLogs = wl
}

func (h *PlatformTokens) WithWebhookLogs(wl *repo.WebhookLogRepo) *PlatformTokens {
	h.webhookLogs = wl
	return h
}

func resolvePlatformScopeAccountID(r *http.Request, scope TenantScope, bodyAccountID int64) int64 {
	if scope.AccountID > 0 {
		return scope.AccountID
	}
	if !scope.IsOwner {
		return scope.AccountID
	}
	if bodyAccountID > 0 {
		return bodyAccountID
	}
	if qAcc := strings.TrimSpace(r.URL.Query().Get("account_id")); qAcc != "" {
		if id, err := strconv.ParseInt(qAcc, 10, 64); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

// Summary consolida status de tokens configurados e estatísticas de webhooks por plataforma.
func (h *PlatformTokens) Summary(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	accID := resolvePlatformScopeAccountID(r, scope, 0)
	if accID < 0 {
		writeError(w, http.StatusForbidden, "empresa inválida")
		return
	}

	items := make(map[string]domain.PlatformSummaryItem)

	if h.tokens != nil && accID > 0 {
		tokMap, err := h.tokens.ListByAccount(r.Context(), accID)
		if err != nil {
			slog.Error("list platform tokens failed", "account_id", accID, "error", err)
			writeError(w, http.StatusInternalServerError, "falha ao listar tokens das plataformas")
			return
		}
		for plat, tok := range tokMap {
			updatedAt := tok.UpdatedAt
			items[plat] = domain.PlatformSummaryItem{
				HasToken:       true,
				MaskedToken:    domain.MaskPlatformSecret(tok.TokenSecret),
				TokenUpdatedAt: &updatedAt,
			}
		}
	}

	if h.webhookLogs != nil {
		statsMap, err := h.webhookLogs.SummaryBySource(r.Context(), accID)
		if err != nil {
			slog.Warn("summary webhook_logs failed", "account_id", accID, "error", err)
		} else {
			for src, st := range statsMap {
				cur := items[src]
				cur.TotalAccepted = st.TotalAccepted
				cur.TotalFailed = st.TotalFailed
				cur.LastReceivedAt = st.LastReceivedAt
				items[src] = cur
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"account_id": accID,
		"items":      items,
	})
}

// GetToken retorna metadados mascarados do token de uma plataforma (nunca expõe o segredo em texto aberto).
func (h *PlatformTokens) GetToken(w http.ResponseWriter, r *http.Request) {
	platformName := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "platform")))
	if !domain.IsSupportedCheckoutPlatform(platformName) {
		writeError(w, http.StatusBadRequest, "plataforma não homologada")
		return
	}

	scope := TenantFromContext(r.Context())
	accID := resolvePlatformScopeAccountID(r, scope, 0)
	if accID <= 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"has_token":    false,
			"masked_token": "",
		})
		return
	}

	token, err := h.tokens.FindByPlatform(r.Context(), accID, platformName)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{
				"has_token":    false,
				"masked_token": "",
			})
			return
		}
		slog.Error("get platform token failed", "platform", platformName, "account_id", accID, "error", err)
		writeError(w, http.StatusInternalServerError, "falha ao consultar token da plataforma")
		return
	}

	hasToken := strings.TrimSpace(token.TokenSecret) != ""
	writeJSON(w, http.StatusOK, map[string]any{
		"has_token":    hasToken,
		"masked_token": domain.MaskPlatformSecret(token.TokenSecret),
		"updated_at":   token.UpdatedAt,
	})
}

// SetToken atualiza, criptografa ou remove o token de segurança de uma plataforma.
func (h *PlatformTokens) SetToken(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "sem permissão para alterar tokens de plataformas")
		return
	}

	platformName := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "platform")))
	if !domain.IsSupportedCheckoutPlatform(platformName) {
		writeError(w, http.StatusBadRequest, "plataforma não homologada")
		return
	}

	var payload struct {
		TokenSecret string `json:"token_secret"`
		AccountID   int64  `json:"account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}

	scope := TenantFromContext(r.Context())
	accID := resolvePlatformScopeAccountID(r, scope, payload.AccountID)
	if accID <= 0 {
		writeError(w, http.StatusBadRequest, "selecione uma empresa antes de salvar o token")
		return
	}

	secret := strings.TrimSpace(payload.TokenSecret)
	t := &domain.PlatformToken{
		AccountID:    accID,
		PlatformName: platformName,
		TokenSecret:  secret,
	}

	if err := h.tokens.UpsertToken(r.Context(), t); err != nil {
		slog.Error("save platform token failed", "platform", platformName, "account_id", accID, "error", err)
		writeError(w, http.StatusInternalServerError, "falha ao salvar token da plataforma")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "success",
		"has_token":    secret != "",
		"masked_token": domain.MaskPlatformSecret(secret),
	})
}
