package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// trackedUserUpdateInput é o payload editável. Todos os campos são opcionais.
type trackedUserUpdateInput struct {
	IDWpp           *string `json:"id_wpp"`
	PixelName       *string `json:"pixel_name"`
	FBP             *string `json:"fbp"`
	FBC             *string `json:"fbc"`
	CtwaClid        *string `json:"ctwa_clid"`
	GCLID           *string `json:"gclid"`
	WBraid          *string `json:"wbraid"`
	GBraid          *string `json:"gbraid"`
	TTClid          *string `json:"ttclid"`
	TTP             *string `json:"ttp"`
	GA4ClientID     *string `json:"ga4_client_id"`
	GA4SessionID    *string `json:"ga4_session_id"`
	ClientIP        *string `json:"client_ip"`
	ClientUserAgent *string `json:"client_user_agent"`

	Country    *string `json:"country"`
	State      *string `json:"state"`
	City       *string `json:"city"`
	PostalCode *string `json:"postal_code"`

	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`

	URL      *string `json:"url"`
	FirstURL *string `json:"first_url"`

	FirstUTMSource   *string `json:"first_utm_source"`
	FirstUTMMedium   *string `json:"first_utm_medium"`
	FirstUTMCampaign *string `json:"first_utm_campaign"`
	FirstUTMContent  *string `json:"first_utm_content"`
	FirstUTMTerm     *string `json:"first_utm_term"`

	LastUTMSource   *string `json:"last_utm_source"`
	LastUTMMedium   *string `json:"last_utm_medium"`
	LastUTMCampaign *string `json:"last_utm_campaign"`
	LastUTMContent  *string `json:"last_utm_content"`
	LastUTMTerm     *string `json:"last_utm_term"`
}

// Update PUT /admin/tracked-users/{id}
func (h *AdminTrackedUsers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	existing, err := h.repo.FindRawByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		slog.Error("admin tracked_users load before update failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "erro ao carregar usuário")
		return
	}
	if !canEdit(r, existing) {
		writeError(w, http.StatusForbidden, "sem acesso a este usuário")
		return
	}

	var in trackedUserUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}

	if in.PixelName != nil {
		nextPixel := strings.TrimSpace(*in.PixelName)
		if nextPixel != "" && nextPixel != existing.PixelName && nextPixel != "api_import" && h.pixels != nil {
			if _, perr := RequireAccountPixel(r.Context(), h.pixels, existing.AccountID, nextPixel); perr != nil {
				writeError(w, http.StatusForbidden, perr.Error())
				return
			}
		}
	}

	origPhone := existing.Phone
	origEmail := existing.Email
	clearedCols := make([]string, 0, 4)

	applyTrackedField(&clearedCols, "id_wpp", &existing.IDWpp, in.IDWpp, strings.TrimSpace)
	applyStr(&existing.PixelName, in.PixelName, strings.TrimSpace)
	applyTrackedField(&clearedCols, "fbp", &existing.FBP, in.FBP, strings.TrimSpace)
	applyTrackedField(&clearedCols, "fbc", &existing.FBC, in.FBC, strings.TrimSpace)
	applyTrackedField(&clearedCols, "ctwa_clid", &existing.CtwaClid, in.CtwaClid, strings.TrimSpace)
	applyTrackedField(&clearedCols, "gclid", &existing.GCLID, in.GCLID, strings.TrimSpace)
	applyTrackedField(&clearedCols, "wbraid", &existing.WBraid, in.WBraid, strings.TrimSpace)
	applyTrackedField(&clearedCols, "gbraid", &existing.GBraid, in.GBraid, strings.TrimSpace)
	applyTrackedField(&clearedCols, "ttclid", &existing.TTClid, in.TTClid, strings.TrimSpace)
	applyTrackedField(&clearedCols, "ttp", &existing.TTP, in.TTP, strings.TrimSpace)
	applyTrackedField(&clearedCols, "ga4_client_id", &existing.GA4ClientID, in.GA4ClientID, strings.TrimSpace)
	applyTrackedField(&clearedCols, "ga4_session_id", &existing.GA4SessionID, in.GA4SessionID, strings.TrimSpace)
	applyTrackedField(&clearedCols, "client_ip", &existing.ClientIP, in.ClientIP, strings.TrimSpace)
	applyTrackedField(&clearedCols, "client_user_agent", &existing.ClientUserAgent, in.ClientUserAgent, strings.TrimSpace)
	applyTrackedField(&clearedCols, "country", &existing.Country, in.Country, util.NormalizeCountry)
	applyTrackedField(&clearedCols, "state", &existing.State, in.State, util.NormalizeStateISO)
	applyTrackedField(&clearedCols, "city", &existing.City, in.City, normalizeLowerTrim)
	applyTrackedField(&clearedCols, "postal_code", &existing.PostalCode, in.PostalCode, strings.TrimSpace)
	applyTrackedField(&clearedCols, "first_name", &existing.FirstName, in.FirstName, strings.TrimSpace)
	applyTrackedField(&clearedCols, "last_name", &existing.LastName, in.LastName, strings.TrimSpace)
	applyTrackedField(&clearedCols, "email", &existing.Email, in.Email, util.NormalizeEmail)
	applyTrackedField(&clearedCols, "phone", &existing.Phone, in.Phone, util.NormalizePhone)
	applyTrackedField(&clearedCols, "url", &existing.URL, in.URL, strings.TrimSpace)
	applyTrackedField(&clearedCols, "first_url", &existing.FirstURL, in.FirstURL, strings.TrimSpace)
	applyTrackedField(&clearedCols, "first_utm_source", &existing.FirstUTMSource, in.FirstUTMSource, strings.TrimSpace)
	applyTrackedField(&clearedCols, "first_utm_medium", &existing.FirstUTMMedium, in.FirstUTMMedium, strings.TrimSpace)
	applyTrackedField(&clearedCols, "first_utm_campaign", &existing.FirstUTMCampaign, in.FirstUTMCampaign, strings.TrimSpace)
	applyTrackedField(&clearedCols, "first_utm_content", &existing.FirstUTMContent, in.FirstUTMContent, strings.TrimSpace)
	applyTrackedField(&clearedCols, "first_utm_term", &existing.FirstUTMTerm, in.FirstUTMTerm, strings.TrimSpace)
	applyTrackedField(&clearedCols, "last_utm_source", &existing.LastUTMSource, in.LastUTMSource, strings.TrimSpace)
	applyTrackedField(&clearedCols, "last_utm_medium", &existing.LastUTMMedium, in.LastUTMMedium, strings.TrimSpace)
	applyTrackedField(&clearedCols, "last_utm_campaign", &existing.LastUTMCampaign, in.LastUTMCampaign, strings.TrimSpace)
	applyTrackedField(&clearedCols, "last_utm_content", &existing.LastUTMContent, in.LastUTMContent, strings.TrimSpace)
	applyTrackedField(&clearedCols, "last_utm_term", &existing.LastUTMTerm, in.LastUTMTerm, strings.TrimSpace)

	if err := h.repo.UpdateByID(r.Context(), existing); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		slog.Error("admin tracked_users update failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "erro ao atualizar usuário")
		return
	}

	if len(clearedCols) > 0 {
		if err := h.repo.ClearGroupColumns(r.Context(), existing.AccountID, existing.GroupID, origPhone, origEmail, clearedCols); err != nil {
			slog.Warn("admin tracked_users clear group columns failed", "id", id, "error", err)
		}
	}

	fresh, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}
	writeJSON(w, http.StatusOK, fresh)
}

// Delete DELETE /admin/tracked-users/{id}
func (h *AdminTrackedUsers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	existing, err := h.repo.FindRawByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		slog.Error("admin tracked_users load before delete failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "erro ao carregar usuário")
		return
	}
	if !canEdit(r, existing) {
		writeError(w, http.StatusForbidden, "sem acesso a este usuário")
		return
	}

	deleted, err := h.repo.DeleteByID(r.Context(), existing.AccountID, id, existing.GroupID)
	if err != nil {
		slog.Error("admin tracked_users delete failed", "id", id, "account_id", existing.AccountID, "error", err)
		writeError(w, http.StatusInternalServerError, "erro ao excluir usuário")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

func canEdit(r *http.Request, u *domain.TrackedUser) bool {
	scope := TenantFromContext(r.Context())
	if scope.IsOwner {
		return scope.AccountID == 0 || u.AccountID == scope.AccountID
	}
	return scope.AccountID > 0 && u.AccountID == scope.AccountID
}

func applyTrackedField(cleared *[]string, col string, dst *string, src *string, norm func(string) string) {
	if src == nil {
		return
	}
	if strings.TrimSpace(*src) == "" {
		*dst = ""
		*cleared = append(*cleared, col)
		return
	}
	v := *src
	if norm != nil {
		v = norm(v)
	}
	*dst = v
}

func applyStr(dst *string, src *string, norm func(string) string) {
	if src == nil {
		return
	}
	v := *src
	if norm != nil {
		v = norm(v)
	}
	*dst = v
}

func normalizeLowerTrim(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
