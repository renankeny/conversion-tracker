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

func (h *AdminCRM) ListTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)

	tags, err := h.crmRepo.ListTags(ctx, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tags")
		return
	}
	if tags == nil {
		tags = []domain.CRMTag{}
	}
	writeJSON(w, http.StatusOK, tags)
}

func (h *AdminCRM) CreateTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)

	var tag domain.CRMTag
	if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	tag.Name = strings.TrimSpace(tag.Name)
	if tag.Name == "" {
		writeError(w, http.StatusBadRequest, "tag name is required")
		return
	}
	if strings.TrimSpace(tag.Color) == "" {
		tag.Color = "#3B82F6"
	}
	tag.AccountID = accountID

	if err := h.crmRepo.CreateTag(ctx, &tag); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create tag")
		return
	}
	h.notifyCRMUpdated(ctx, accountID, "tag_created")
	writeJSON(w, http.StatusOK, tag)
}

func (h *AdminCRM) DeleteTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if isEmployeeRole(ctx) {
		writeError(w, http.StatusForbidden, "funcionários não podem excluir tags da empresa")
		return
	}
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	tagID, err := strconv.ParseInt(chi.URLParam(r, "tag_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tag id")
		return
	}
	if err := h.crmRepo.DeleteTag(ctx, tagID, accountID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tag not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete tag")
		return
	}
	h.notifyCRMUpdated(ctx, accountID, "tag_deleted")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminCRM) AssignTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	leadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	tagID, err2 := strconv.ParseInt(chi.URLParam(r, "tag_id"), 10, 64)
	if err != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "invalid ids")
		return
	}

	if _, err := h.crmRepo.GetLeadByID(ctx, leadID, accountID); err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}

	if err := h.crmRepo.AssignTag(ctx, accountID, leadID, tagID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusForbidden, "tag não pertence a esta empresa")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to assign tag")
		return
	}
	h.notifyCRMUpdated(ctx, accountID, "tag_assigned")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminCRM) RemoveTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	leadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	tagID, err2 := strconv.ParseInt(chi.URLParam(r, "tag_id"), 10, 64)
	if err != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "invalid ids")
		return
	}

	if _, err := h.crmRepo.GetLeadByID(ctx, leadID, accountID); err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}

	if err := h.crmRepo.RemoveTag(ctx, accountID, leadID, tagID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove tag")
		return
	}
	h.notifyCRMUpdated(ctx, accountID, "tag_removed")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminCRM) APIListStages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	_ = h.crmRepo.EnsureDefaultStages(ctx, accountID)

	stages, err := h.crmRepo.GetStages(ctx, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get stages")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stages": stages})
}

func (h *AdminCRM) APICreateLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)

	var req struct {
		FirstName  string  `json:"first_name"`
		LastName   string  `json:"last_name"`
		Email      string  `json:"email"`
		Phone      string  `json:"phone"`
		StageID    int64   `json:"stage_id"`
		Value      float64 `json:"value"`
		ExternalID string  `json:"external_id"`
		PixelName  string  `json:"pixel_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.StageID == 0 {
		writeError(w, http.StatusBadRequest, "stage_id is required")
		return
	}
	stage, err := h.crmRepo.GetStageByID(ctx, req.StageID, accountID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage_id for this account")
		return
	}
	if stage.TriggerEvent == domain.EventPurchase && req.Value <= 0 {
		writeError(w, http.StatusBadRequest, "value required for purchase stage")
		return
	}
	if req.PixelName != "" && req.PixelName != "api_import" && h.pixels != nil {
		if _, err := RequireAccountPixel(ctx, h.pixels, accountID, req.PixelName); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	email := util.NormalizeEmail(req.Email)
	phone := util.NormalizePhone(req.Phone)
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)

	if email == "" && phone == "" && strings.TrimSpace(req.ExternalID) == "" {
		writeError(w, http.StatusBadRequest, "must provide email, phone, or external_id")
		return
	}

	trackedUser := h.resolveOrCreateCRMTrackedUser(ctx, accountID, firstName, lastName, email, phone, req.ExternalID, req.PixelName)
	if trackedUser == nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	if email != "" || phone != "" {
		if _, lerr := h.users.ResolveAndLink(ctx, accountID, trackedUser.ExternalID, phone, email); lerr != nil {
			slog.Warn("resolve identity link on crm create failed", "err", lerr, "external_id", trackedUser.ExternalID)
		}
	}

	leadID, prevStageID, err := h.crmRepo.CreateOrUpdateLeadAPI(ctx, accountID, trackedUser.ID, req.StageID, req.Value)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create or update lead")
		return
	}

	if stage.TriggerEvent != "" && (prevStageID == nil || *prevStageID != req.StageID) {
		if fullLead, gerr := h.crmRepo.GetLeadByID(ctx, leadID, accountID); gerr == nil && fullLead.TrackedUser != nil {
			h.dispatchCRMStageTrigger(ctx, accountID, stage, fullLead.TrackedUser, req.Value)
		} else {
			h.dispatchCRMStageTrigger(ctx, accountID, stage, trackedUser, req.Value)
		}
	}

	h.notifyCRMUpdated(ctx, accountID, "lead_created")
	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"lead_id":  leadID,
		"stage_id": req.StageID,
		"user_id":  trackedUser.ID,
	})
}
