package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/campaigns"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminCampaigns struct {
	svc    *campaigns.Service
	pixels *repo.PixelRepo
}

func NewAdminCampaigns(svc *campaigns.Service, pixels *repo.PixelRepo) *AdminCampaigns {
	return &AdminCampaigns{svc: svc, pixels: pixels}
}

func parseCampaignPeriod(r *http.Request) (since, until string, from, to time.Time) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil || loc == nil {
		loc = time.FixedZone("BRT", -3*3600)
	}
	now := time.Now().In(loc)
	rawSince := strings.TrimSpace(r.URL.Query().Get("since"))
	rawUntil := strings.TrimSpace(r.URL.Query().Get("until"))

	tFrom, err1 := time.ParseInLocation("2006-01-02", rawSince, loc)
	tTo, err2 := time.ParseInLocation("2006-01-02", rawUntil, loc)
	if err1 != nil || err2 != nil {
		tTo = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		tFrom = tTo.AddDate(0, 0, -6)
	}
	if tTo.Before(tFrom) {
		tTo = tFrom
	}
	return tFrom.Format("2006-01-02"), tTo.Format("2006-01-02"), tFrom, tTo.AddDate(0, 0, 1)
}

func (h *AdminCampaigns) Overview(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	since, until, from, to := parseCampaignPeriod(r)

	platform := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("platform")))
	if platform != "meta" && platform != "google" {
		platform = "all"
	}
	attrMode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("attribution")))
	if attrMode != "first_click" {
		attrMode = "last_click"
	}
	metaAccID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("meta_account_id")), 10, 64)
	gadsAccID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("google_ads_account_id")), 10, 64)

	params := campaigns.OverviewParams{
		AccountID:          scope.AccountID,
		PixelName:          strings.TrimSpace(r.URL.Query().Get("pixel")),
		Platform:           platform,
		AttributionMode:    attrMode,
		Since:              since,
		Until:              until,
		From:               from,
		To:                 to,
		MetaAccountID:      metaAccID,
		MetaAdAccountID:    strings.TrimSpace(r.URL.Query().Get("meta_ad_account_id")),
		GoogleAdsAccountID: gadsAccID,
		GoogleCustomerID:   strings.TrimSpace(r.URL.Query().Get("google_customer_id")),
		ForceRefresh:       r.URL.Query().Get("refresh") == "1" || r.URL.Query().Get("refresh") == "true",
	}

	resp, err := h.svc.Overview(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AdminCampaigns) MutateStatus(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para alterar status de campanhas")
		return
	}
	var in domain.CampaignStatusMutateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if strings.TrimSpace(in.ObjectID) == "" {
		writeError(w, http.StatusBadRequest, "object_id obrigatório")
		return
	}
	scope := TenantFromContext(r.Context())
	if err := h.svc.MutateStatus(r.Context(), scope.AccountID, in); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	nextStatus := "PAUSED"
	if strings.EqualFold(in.Status, "ACTIVE") || strings.EqualFold(in.Status, "ENABLED") {
		nextStatus = "ACTIVE"
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": nextStatus})
}

func (h *AdminCampaigns) MutateBudget(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para alterar orçamento de campanhas")
		return
	}
	var in domain.CampaignBudgetMutateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if in.DailyBudget <= 0 {
		writeError(w, http.StatusBadRequest, "informe um orçamento diário maior que zero")
		return
	}
	scope := TenantFromContext(r.Context())
	if err := h.svc.MutateBudget(r.Context(), scope.AccountID, in); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "daily_budget": in.DailyBudget})
}

func (h *AdminCampaigns) UpdatePixelFinancials(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para alterar taxas do pixel")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID de pixel inválido")
		return
	}
	var cfg domain.PixelFinancialConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	scope := TenantFromContext(r.Context())
	if err := h.pixels.UpdateFinancialConfig(r.Context(), id, scope.AccountID, cfg); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pixel não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao salvar taxas do pixel")
		return
	}
	h.svc.InvalidateAccountCache(r.Context(), scope.AccountID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pixel_financials": cfg})
}
