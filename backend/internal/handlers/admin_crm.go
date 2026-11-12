package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

type AdminCRM struct {
	crmRepo  *repo.CRMRepo
	users    *repo.UserRepo
	queue    *queue.Enqueuer
	pixels   *repo.PixelRepo
	gads     *repo.GoogleAdsAccountRepo
	whatsapp *repo.WhatsAppRepo
	metaAcc  *repo.MetaAccountRepo
	rdb      *redis.Client
}

func NewAdminCRM(crmRepo *repo.CRMRepo, users *repo.UserRepo, q *queue.Enqueuer) *AdminCRM {
	return &AdminCRM{crmRepo: crmRepo, users: users, queue: q}
}

func (h *AdminCRM) SetSecurityDeps(pixels *repo.PixelRepo, gads *repo.GoogleAdsAccountRepo) {
	h.pixels = pixels
	h.gads = gads
}

func (h *AdminCRM) SetWhatsAppRepo(wa *repo.WhatsAppRepo) {
	h.whatsapp = wa
}

func (h *AdminCRM) SetMetaAccountRepo(m *repo.MetaAccountRepo) {
	h.metaAcc = m
}

func (h *AdminCRM) SetRedis(rdb *redis.Client) {
	h.rdb = rdb
}

func (h *AdminCRM) notifyCRMUpdated(ctx context.Context, accountID int64, action string) {
	if h.rdb == nil || accountID <= 0 {
		return
	}
	wshub.PublishCustom(ctx, h.rdb, accountID, "crm_updated", map[string]any{"action": action})
}

func isEmployeeRole(ctx context.Context) bool {
	c, _ := auth.ClaimsFrom(ctx)
	return c != nil && c.Role == "employee"
}

// GetBoard retorna as colunas e os leads.
func (h *AdminCRM) GetBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := TenantFromContext(ctx)
	accountID := scope.EffectiveAccountID(0)
	pixel := r.URL.Query().Get("pixel")

	_ = h.crmRepo.EnsureDefaultStages(ctx, accountID)

	stages, err := h.crmRepo.GetStages(ctx, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get stages")
		return
	}

	leads, err := h.crmRepo.GetLeads(ctx, accountID, pixel)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get leads")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"stages": stages,
		"leads":  leads,
	})
}

type StageReq struct {
	Name               string `json:"name"`
	Color              string `json:"color"`
	TriggerEvent       string `json:"trigger_event"`
	TriggerEventCTWA   string `json:"trigger_event_ctwa"`
	GoogleAdsAccountID *int64 `json:"google_ads_account_id"`
	GoogleAdsActionID  string `json:"google_ads_action_id"`
}

func (h *AdminCRM) CreateStage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if isEmployeeRole(ctx) {
		writeError(w, http.StatusForbidden, "funcionários não podem criar etapas do funil")
		return
	}
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)

	var req StageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "stage name is required")
		return
	}
	if req.GoogleAdsAccountID != nil && *req.GoogleAdsAccountID <= 0 {
		req.GoogleAdsAccountID = nil
	}
	if req.GoogleAdsAccountID == nil {
		req.GoogleAdsActionID = ""
	}
	if err := RequireAccountGoogleAds(ctx, h.gads, accountID, req.GoogleAdsAccountID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	if err := h.crmRepo.CreateStage(ctx, accountID, req.Name, req.Color, req.TriggerEvent, req.TriggerEventCTWA, req.GoogleAdsAccountID, req.GoogleAdsActionID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create stage")
		return
	}
	h.notifyCRMUpdated(ctx, accountID, "stage_created")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminCRM) UpdateStage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if isEmployeeRole(ctx) {
		writeError(w, http.StatusForbidden, "funcionários não podem configurar etapas do funil")
		return
	}
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req StageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "stage name is required")
		return
	}
	if req.GoogleAdsAccountID != nil && *req.GoogleAdsAccountID <= 0 {
		req.GoogleAdsAccountID = nil
	}
	if req.GoogleAdsAccountID == nil {
		req.GoogleAdsActionID = ""
	}
	if err := RequireAccountGoogleAds(ctx, h.gads, accountID, req.GoogleAdsAccountID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	if err := h.crmRepo.UpdateStage(ctx, stageID, accountID, req.Name, req.Color, req.TriggerEvent, req.TriggerEventCTWA, req.GoogleAdsAccountID, req.GoogleAdsActionID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update stage")
		return
	}
	h.notifyCRMUpdated(ctx, accountID, "stage_updated")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminCRM) DeleteStage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if isEmployeeRole(ctx) {
		writeError(w, http.StatusForbidden, "funcionários não podem excluir etapas do funil")
		return
	}
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.crmRepo.DeleteStage(ctx, stageID, accountID); err != nil {
		if errors.Is(err, repo.ErrCannotDeleteLastStage) {
			writeError(w, http.StatusBadRequest, "Não é possível excluir a última etapa do funil.")
			return
		}
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "stage not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete stage")
		return
	}
	h.notifyCRMUpdated(ctx, accountID, "stage_deleted")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type ReorderReq struct {
	Stages []struct {
		ID         int64 `json:"id"`
		OrderIndex int   `json:"order_index"`
	} `json:"stages"`
}

func (h *AdminCRM) ReorderStages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if isEmployeeRole(ctx) {
		writeError(w, http.StatusForbidden, "funcionários não podem reordenar o funil")
		return
	}
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)

	var req ReorderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	orders := make(map[int64]int)
	for _, s := range req.Stages {
		orders[s.ID] = s.OrderIndex
	}

	if err := h.crmRepo.UpdateStageOrder(ctx, accountID, orders); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update orders")
		return
	}
	h.notifyCRMUpdated(ctx, accountID, "stages_reordered")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
