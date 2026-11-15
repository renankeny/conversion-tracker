package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// UpdateDashboardConfig PUT /admin/pixels/{id}/dashboard-config
func (h *AdminPixels) UpdateDashboardConfig(w http.ResponseWriter, r *http.Request) {
	existing, ok := h.loadEditablePixel(w, r)
	if !ok {
		return
	}
	var cfg domain.DashboardConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if err := h.repo.UpdateDashboardConfig(r.Context(), existing.ID, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar dashboard: "+err.Error())
		return
	}
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, existing.DashboardShareToken)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "dashboard_config": cfg})
}

// GenerateDashboardShareLink POST /admin/pixels/{id}/dashboard-share-link
func (h *AdminPixels) GenerateDashboardShareLink(w http.ResponseWriter, r *http.Request) {
	existing, ok := h.loadEditablePixel(w, r)
	if !ok {
		return
	}
	token, err := h.repo.GenerateDashboardShareToken(r.Context(), existing.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar link: "+err.Error())
		return
	}
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, existing.DashboardShareToken, token)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                    existing.ID,
		"dashboard_share_token": token,
	})
}

// RevokeDashboardShareLink DELETE /admin/pixels/{id}/dashboard-share-link
func (h *AdminPixels) RevokeDashboardShareLink(w http.ResponseWriter, r *http.Request) {
	existing, ok := h.loadEditablePixel(w, r)
	if !ok {
		return
	}
	if err := h.repo.RevokeDashboardShareToken(r.Context(), existing.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao revogar link: "+err.Error())
		return
	}
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, existing.DashboardShareToken)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminPixels) loadEditablePixel(w http.ResponseWriter, r *http.Request) (*domain.Pixel, bool) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar pixels")
		return nil, false
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return nil, false
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pixel não encontrado")
			return nil, false
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar pixel")
		return nil, false
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return nil, false
	}
	return existing, true
}
