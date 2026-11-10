package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func (h *AdminPixels) Update(w http.ResponseWriter, r *http.Request) {
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
	var in pixelInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validatePixelInput(in, false); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	oldName := existing.Name
	if in.Name != "" {
		existing.Name = strings.TrimSpace(in.Name)
	}
	if in.Label != "" {
		existing.Label = in.Label
	}
	oldMetaTokens := make(map[string]string, len(existing.PixelIDs))
	for i, pid := range existing.PixelIDs {
		if i < len(existing.AccessTokens) {
			oldMetaTokens[pid] = existing.AccessTokens[i]
		}
	}
	oldMetaSlice := append([]string(nil), existing.AccessTokens...)

	oldTikTokTokens := make(map[string]string, len(existing.TikTokPixelIDs))
	for i, pid := range existing.TikTokPixelIDs {
		if i < len(existing.TikTokAccessTokens) {
			oldTikTokTokens[pid] = existing.TikTokAccessTokens[i]
		}
	}
	oldTikTokSlice := append([]string(nil), existing.TikTokAccessTokens...)

	if in.PixelIDs != nil {
		existing.PixelIDs = in.PixelIDs
	}
	if in.AccessTokens != nil {
		existing.AccessTokens = mergeTokens(existing.PixelIDs, in.AccessTokens, oldMetaTokens, oldMetaSlice)
	} else if in.PixelIDs != nil {
		existing.AccessTokens = mergeTokens(existing.PixelIDs, make([]string, len(existing.PixelIDs)), oldMetaTokens, oldMetaSlice)
	}
	if in.TestCodes != nil {
		existing.TestCodes = padOrTrim(in.TestCodes, len(existing.PixelIDs))
	}
	if in.Domains != nil {
		existing.Domains = cleanDomains(in.Domains)
	}
	if in.TikTokPixelIDs != nil {
		existing.TikTokPixelIDs = in.TikTokPixelIDs
	}
	if in.TikTokAccessTokens != nil {
		existing.TikTokAccessTokens = mergeTokens(existing.TikTokPixelIDs, in.TikTokAccessTokens, oldTikTokTokens, oldTikTokSlice)
	} else if in.TikTokPixelIDs != nil {
		existing.TikTokAccessTokens = mergeTokens(existing.TikTokPixelIDs, make([]string, len(existing.TikTokPixelIDs)), oldTikTokTokens, oldTikTokSlice)
	}
	if in.TikTokTestCodes != nil {
		existing.TikTokTestCodes = padOrTrim(in.TikTokTestCodes, len(existing.TikTokPixelIDs))
	}
	if in.GoogleAdsAccountID != nil {
		existing.GoogleAdsAccountID = nilIfZeroInt64(*in.GoogleAdsAccountID)
	}
	if in.MetaAccountID != nil {
		existing.MetaAccountID = nilIfZeroInt64(*in.MetaAccountID)
	}
	if in.TrackerConfig != nil {
		existing.TrackerConfig = *in.TrackerConfig
	}
	if in.Enabled != nil {
		existing.Enabled = *in.Enabled
	}
	if in.TrackingHost != nil {
		existing.TrackingHost = normalizeHost(*in.TrackingHost)
	}
	if in.GA4MeasurementID != nil {
		existing.GA4MeasurementID = strings.TrimSpace(*in.GA4MeasurementID)
	}
	if in.GA4APISecret != nil {
		val := strings.TrimSpace(*in.GA4APISecret)
		if !isMaskedOrEmpty(val) {
			existing.GA4APISecret = val
		} else if val == "" && existing.GA4MeasurementID == "" {
			existing.GA4APISecret = ""
		}
	}
	if in.MetaPermanentToken != nil {
		val := strings.TrimSpace(*in.MetaPermanentToken)
		if !isMaskedOrEmpty(val) {
			existing.MetaPermanentToken = val
		}
	}
	if in.MetaAdAccountID != nil {
		existing.MetaAdAccountID = strings.TrimSpace(*in.MetaAdAccountID)
	}
	if in.AccountID != nil && *in.AccountID > 0 && *in.AccountID != existing.AccountID {
		targetAccID, accErr := ResolveTargetAccountID(r.Context(), h.pool, in.AccountID)
		if accErr != nil {
			writeError(w, http.StatusForbidden, accErr.Error())
			return
		}
		existing.AccountID = targetAccID
		if in.GoogleAdsAccountID == nil && existing.GoogleAdsAccountID != nil {
			if RequireAccountGoogleAds(r.Context(), h.gads, existing.AccountID, existing.GoogleAdsAccountID) != nil {
				existing.GoogleAdsAccountID = nil
			}
		}
		if in.MetaAccountID == nil && existing.MetaAccountID != nil {
			if RequireAccountMetaAds(r.Context(), h.metaAccs, existing.AccountID, existing.MetaAccountID) != nil {
				existing.MetaAccountID = nil
			}
		}
	}
	if err := RequireAccountGoogleAds(r.Context(), h.gads, existing.AccountID, existing.GoogleAdsAccountID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err := RequireAccountMetaAds(r.Context(), h.metaAccs, existing.AccountID, existing.MetaAccountID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if in.AggregationEnabled != nil {
		existing.AggregationEnabled = *in.AggregationEnabled
	}
	if in.AggregationWindowMinutes != nil {
		existing.AggregationWindowMinutes = *in.AggregationWindowMinutes
	}
	if len(existing.PixelIDs) == 0 && existing.MetaAccountID == nil && len(existing.TikTokPixelIDs) == 0 && existing.GoogleAdsAccountID == nil && existing.GA4MeasurementID == "" {
		writeError(w, http.StatusBadRequest, "configure pelo menos um canal (Meta, TikTok, Google Ads ou GA4)")
		return
	}
	if err := h.repo.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	h.refreshDomains(r.Context())
	h.repo.InvalidateCache(r.Context(), oldName, existing.Name, existing.PublicToken, existing.DashboardShareToken)
	writeJSON(w, http.StatusOK, existing.PublicView())
}

// Delete remove o pixel. Exige `confirm: true` no body.
func (h *AdminPixels) Delete(w http.ResponseWriter, r *http.Request) {
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
	var body struct {
		Confirm bool `json:"confirm"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if !body.Confirm {
		writeError(w, http.StatusConflict, "destructive operation requires {\"confirm\": true} in body")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "delete error")
		return
	}
	h.refreshDomains(r.Context())
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, existing.DashboardShareToken)
	w.WriteHeader(http.StatusNoContent)
}
