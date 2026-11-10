package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// UpdateDashboardConfig PUT /admin/pixels/{id}/dashboard-config
func (h *AdminPixels) UpdateDashboardConfig(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar pixels")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}

	var cfg domain.DashboardConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.repo.UpdateDashboardConfig(r.Context(), id, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, existing.DashboardShareToken)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "dashboard_config": cfg})
}

// GenerateDashboardShareLink POST /admin/pixels/{id}/dashboard-share-link
func (h *AdminPixels) GenerateDashboardShareLink(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar pixels")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	token, err := h.repo.GenerateDashboardShareToken(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "generate error: "+err.Error())
		return
	}
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, existing.DashboardShareToken, token)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                    id,
		"dashboard_share_token": token,
	})
}

// RevokeDashboardShareLink DELETE /admin/pixels/{id}/dashboard-share-link
func (h *AdminPixels) RevokeDashboardShareLink(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar pixels")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	if err := h.repo.RevokeDashboardShareToken(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "revoke error: "+err.Error())
		return
	}
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, existing.DashboardShareToken)
	w.WriteHeader(http.StatusNoContent)
}

func validatePixelInput(in pixelInput, requireAll bool) error {
	if requireAll {
		if strings.TrimSpace(in.Name) == "" {
			return errors.New("nome do pixel é obrigatório")
		}
		hasMeta := len(in.PixelIDs) > 0
		hasMetaOAuth := in.MetaAccountID != nil && *in.MetaAccountID > 0
		hasTikTok := len(in.TikTokPixelIDs) > 0
		hasGoogleAds := in.GoogleAdsAccountID != nil && *in.GoogleAdsAccountID > 0
		hasGA4 := in.GA4MeasurementID != nil && strings.TrimSpace(*in.GA4MeasurementID) != ""
		if !hasMeta && !hasMetaOAuth && !hasTikTok && !hasGoogleAds && !hasGA4 {
			return errors.New("configure pelo menos um canal (Meta, TikTok, Google Ads ou GA4)")
		}
		if hasMeta && !hasMetaOAuth && len(in.AccessTokens) != len(in.PixelIDs) {
			return errors.New("access_tokens length must match pixel_ids")
		}
		if hasTikTok && len(in.TikTokAccessTokens) != len(in.TikTokPixelIDs) {
			return errors.New("tiktok_access_tokens length must match tiktok_pixel_ids")
		}
	}
	if in.PixelIDs != nil && in.AccessTokens != nil &&
		len(in.AccessTokens) != len(in.PixelIDs) {
		return errors.New("access_tokens length must match pixel_ids")
	}
	// TikTok: se algum dos dois for informado, ambos devem ter o mesmo tamanho.
	if in.TikTokPixelIDs != nil && in.TikTokAccessTokens != nil &&
		len(in.TikTokAccessTokens) != len(in.TikTokPixelIDs) {
		return errors.New("tiktok_access_tokens length must match tiktok_pixel_ids")
	}
	return nil
}

func padOrTrim(s []string, n int) []string {
	out := make([]string, n)
	for i := 0; i < n && i < len(s); i++ {
		out[i] = s[i]
	}
	return out
}

func isMaskedOrEmpty(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || s == "****" || strings.Contains(s, "...")
}

func mergeTokens(pixelIDs, incomingTokens []string, oldByPixelID map[string]string, oldSlice []string) []string {
	out := make([]string, len(pixelIDs))
	for i, pid := range pixelIDs {
		tok := ""
		if i < len(incomingTokens) {
			tok = strings.TrimSpace(incomingTokens[i])
		}
		if !isMaskedOrEmpty(tok) {
			out[i] = tok
			continue
		}
		if prev, ok := oldByPixelID[pid]; ok && prev != "" {
			out[i] = prev
		} else if i < len(oldSlice) {
			out[i] = oldSlice[i]
		}
	}
	return out
}

// nilIfZeroInt64 retorna nil se v == 0, senao um *int64 apontando pra v.
func nilIfZeroInt64(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

// cleanDomains normaliza domínios (lowercase, trim, sem duplicatas e sem strings vazias).
func cleanDomains(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, d := range in {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

// normalizeHost limpa um host: remove scheme, path, espaço e força lowercase.
func normalizeHost(in string) string {
	h := strings.TrimSpace(in)
	if h == "" {
		return ""
	}
	h = strings.ToLower(h)
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	if i := strings.IndexAny(h, "/?#"); i != -1 {
		h = h[:i]
	}
	return strings.TrimSuffix(h, ".")
}
