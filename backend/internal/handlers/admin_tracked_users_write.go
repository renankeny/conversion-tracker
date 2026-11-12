package handlers

import (
	"encoding/json"
	"errors"
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

	URL *string `json:"url"`

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
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error: "+err.Error())
		return
	}
	if !canEdit(r, existing) {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}

	var in trackedUserUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
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

	applyStr(&existing.IDWpp, in.IDWpp, strings.TrimSpace)
	applyStr(&existing.PixelName, in.PixelName, strings.TrimSpace)
	applyStr(&existing.FBP, in.FBP, strings.TrimSpace)
	applyStr(&existing.FBC, in.FBC, strings.TrimSpace)
	applyStr(&existing.CtwaClid, in.CtwaClid, strings.TrimSpace)
	applyStr(&existing.GCLID, in.GCLID, strings.TrimSpace)
	applyStr(&existing.WBraid, in.WBraid, strings.TrimSpace)
	applyStr(&existing.GBraid, in.GBraid, strings.TrimSpace)
	applyStr(&existing.TTClid, in.TTClid, strings.TrimSpace)
	applyStr(&existing.TTP, in.TTP, strings.TrimSpace)
	applyStr(&existing.GA4ClientID, in.GA4ClientID, strings.TrimSpace)
	applyStr(&existing.GA4SessionID, in.GA4SessionID, strings.TrimSpace)
	applyStr(&existing.ClientIP, in.ClientIP, strings.TrimSpace)
	applyStr(&existing.ClientUserAgent, in.ClientUserAgent, strings.TrimSpace)
	applyStr(&existing.Country, in.Country, normalizeLowerTrim)
	applyStr(&existing.State, in.State, normalizeLowerTrim)
	applyStr(&existing.City, in.City, normalizeLowerTrim)
	applyStr(&existing.PostalCode, in.PostalCode, strings.TrimSpace)
	applyStr(&existing.FirstName, in.FirstName, strings.TrimSpace)
	applyStr(&existing.LastName, in.LastName, strings.TrimSpace)
	applyStr(&existing.Email, in.Email, util.NormalizeEmail)
	applyStr(&existing.Phone, in.Phone, util.NormalizePhone)
	applyStr(&existing.URL, in.URL, strings.TrimSpace)
	applyStr(&existing.FirstUTMSource, in.FirstUTMSource, strings.TrimSpace)
	applyStr(&existing.FirstUTMMedium, in.FirstUTMMedium, strings.TrimSpace)
	applyStr(&existing.FirstUTMCampaign, in.FirstUTMCampaign, strings.TrimSpace)
	applyStr(&existing.FirstUTMContent, in.FirstUTMContent, strings.TrimSpace)
	applyStr(&existing.FirstUTMTerm, in.FirstUTMTerm, strings.TrimSpace)
	applyStr(&existing.LastUTMSource, in.LastUTMSource, strings.TrimSpace)
	applyStr(&existing.LastUTMMedium, in.LastUTMMedium, strings.TrimSpace)
	applyStr(&existing.LastUTMCampaign, in.LastUTMCampaign, strings.TrimSpace)
	applyStr(&existing.LastUTMContent, in.LastUTMContent, strings.TrimSpace)
	applyStr(&existing.LastUTMTerm, in.LastUTMTerm, strings.TrimSpace)

	if err := h.repo.UpdateByID(r.Context(), existing); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}

	fresh, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}
	writeJSON(w, http.StatusOK, fresh)
}

func canEdit(r *http.Request, u *domain.TrackedUser) bool {
	scope := TenantFromContext(r.Context())
	if scope.IsOwner {
		return true
	}
	return u.AccountID == scope.AccountID
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
