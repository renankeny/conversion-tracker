package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type PlatformTokens struct {
	tokens *repo.PlatformTokenRepo
}

func NewPlatformTokens(tokens *repo.PlatformTokenRepo) *PlatformTokens {
	return &PlatformTokens{tokens: tokens}
}

// GetToken recupera o token de uma plataforma
func (h *PlatformTokens) GetToken(w http.ResponseWriter, r *http.Request) {
	accID := TenantFromContext(r.Context()).AccountID
	platformName := chi.URLParam(r, "platform")

	token, err := h.tokens.FindByPlatform(r.Context(), accID, platformName)
	if err != nil {
		if err == repo.ErrNotFound {
			writeJSON(w, http.StatusOK, map[string]string{"token_secret": ""})
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token_secret": token.TokenSecret})
}

// SetToken atualiza ou insere o token de uma plataforma
func (h *PlatformTokens) SetToken(w http.ResponseWriter, r *http.Request) {
	accID := TenantFromContext(r.Context()).AccountID
	platformName := chi.URLParam(r, "platform")

	var payload struct {
		TokenSecret string `json:"token_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	t := &domain.PlatformToken{
		AccountID:    accID,
		PlatformName: platformName,
		TokenSecret:  payload.TokenSecret,
	}

	if err := h.tokens.UpsertToken(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
