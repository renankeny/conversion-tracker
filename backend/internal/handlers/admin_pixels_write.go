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
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func (h *AdminPixels) Update(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar pixels")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pixel não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar pixel")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	var in pixelInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if err := validatePixelInput(in, false); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if trimmedName := strings.TrimSpace(in.Name); trimmedName != "" && trimmedName != existing.Name {
		writeError(w, http.StatusBadRequest, "o identificador (slug) do pixel não pode ser alterado após a criação; altere o campo Rótulo")
		return
	}
	if in.Label != nil {
		existing.Label = strings.TrimSpace(*in.Label)
	}

	oldMetaTokens := indexTokensByPixelID(existing.PixelIDs, existing.AccessTokens)
	oldTikTokTokens := indexTokensByPixelID(existing.TikTokPixelIDs, existing.TikTokAccessTokens)

	if in.PixelIDs != nil {
		existing.PixelIDs = cleanStringSlice(in.PixelIDs)
	}
	if in.AccessTokens != nil {
		existing.AccessTokens = mergeTokens(existing.PixelIDs, in.AccessTokens, oldMetaTokens)
	} else if in.PixelIDs != nil {
		existing.AccessTokens = mergeTokens(existing.PixelIDs, nil, oldMetaTokens)
	}
	existing.TestCodes = []string{}
	if in.Domains != nil {
		existing.Domains = cleanDomains(in.Domains)
	}
	if in.TikTokPixelIDs != nil {
		existing.TikTokPixelIDs = cleanStringSlice(in.TikTokPixelIDs)
	}
	if in.TikTokAccessTokens != nil {
		existing.TikTokAccessTokens = mergeTokens(existing.TikTokPixelIDs, in.TikTokAccessTokens, oldTikTokTokens)
	} else if in.TikTokPixelIDs != nil {
		existing.TikTokAccessTokens = mergeTokens(existing.TikTokPixelIDs, nil, oldTikTokTokens)
	}
	existing.TikTokTestCodes = []string{}
	if in.GoogleAdsAccountID != nil {
		existing.GoogleAdsAccountID = nilIfZeroInt64(*in.GoogleAdsAccountID)
		if existing.GoogleAdsAccountID == nil {
			existing.GoogleAdsCustomerID = ""
			existing.GoogleAdsCustomerName = ""
			existing.GoogleAdsLoginCustomerID = ""
		}
	}
	if in.MetaAccountID != nil {
		existing.MetaAccountID = nilIfZeroInt64(*in.MetaAccountID)
		if existing.MetaAccountID == nil && in.MetaAdAccountID == nil {
			existing.MetaAdAccountID = ""
		}
	}
	if in.TrackerConfig != nil {
		existing.TrackerConfig = *in.TrackerConfig
	}
	applyOptionalPixelFields(existing, in)
	if existing.GA4MeasurementID == "" {
		existing.GA4APISecret = ""
	}

	prevAccountID := existing.AccountID
	isAccountTransfer := false
	if in.AccountID != nil && *in.AccountID > 0 && *in.AccountID != existing.AccountID {
		targetAccID, accErr := ResolveTargetAccountID(r.Context(), h.pool, in.AccountID)
		if accErr != nil {
			writeError(w, http.StatusForbidden, accErr.Error())
			return
		}
		isAccountTransfer = true
		existing.AccountID = targetAccID
		if in.GoogleAdsAccountID == nil && existing.GoogleAdsAccountID != nil {
			if RequireAccountGoogleAds(r.Context(), h.gads, existing.AccountID, existing.GoogleAdsAccountID) != nil {
				existing.GoogleAdsAccountID = nil
				existing.GoogleAdsCustomerID = ""
				existing.GoogleAdsCustomerName = ""
				existing.GoogleAdsLoginCustomerID = ""
			}
		}
		if in.MetaAccountID == nil && existing.MetaAccountID != nil {
			if RequireAccountMetaAds(r.Context(), h.metaAccs, existing.AccountID, existing.MetaAccountID) != nil {
				existing.MetaAccountID = nil
				existing.MetaAdAccountID = ""
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
	if !isAccountTransfer {
		hasOtherChannels := len(existing.TikTokPixelIDs) > 0 || existing.GoogleAdsAccountID != nil || existing.GA4MeasurementID != ""
		if len(existing.PixelIDs) == 0 && existing.MetaAccountID != nil && !hasOtherChannels {
			writeError(w, http.StatusBadRequest, "selecione pelo menos um Pixel ID da Meta para disparar eventos via OAuth")
			return
		}
		if len(existing.PixelIDs) == 0 && existing.MetaAccountID == nil && !hasOtherChannels {
			writeError(w, http.StatusBadRequest, "configure pelo menos um canal (Meta, TikTok, Google Ads ou GA4)")
			return
		}
	}
	syncPixelGoogleAdsDefaults(r.Context(), h.gads, existing)
	syncPixelMetaAdsDefaults(r.Context(), h.metaAccs, existing)
	if err := h.repo.Update(r.Context(), existing); err != nil {
		if isDuplicateKey(err) {
			writeError(w, http.StatusConflict, "já existe um pixel com este nome (slug)")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao atualizar pixel: "+err.Error())
		return
	}
	if isAccountTransfer {
		if migErr := h.repo.MigratePixelAccountData(r.Context(), existing.Name, prevAccountID, existing.AccountID); migErr != nil {
			slog.Warn("migrate pixel account data", "pixel_name", existing.Name, "error", migErr)
		}
	}
	h.refreshDomains(r.Context())
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, existing.DashboardShareToken)
	h.repo.BumpDashboardCache(r.Context(), existing.AccountID)
	writeJSON(w, http.StatusOK, existing.PublicView())
}

func (h *AdminPixels) Delete(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar pixels")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pixel não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar pixel")
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
		writeError(w, http.StatusConflict, "operação destrutiva exige {\"confirm\": true}")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir pixel")
		return
	}
	h.refreshDomains(r.Context())
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, existing.DashboardShareToken)
	h.repo.BumpDashboardCache(r.Context(), existing.AccountID)
	w.WriteHeader(http.StatusNoContent)
}
