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
)

type MoveLeadReq struct {
	StageID int64   `json:"stage_id"`
	Value   float64 `json:"value"`
}

// MoveLead movimenta o lead e dispara evento se a coluna tiver trigger_event.
func (h *AdminCRM) MoveLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := TenantFromContext(ctx)
	accountID := scope.EffectiveAccountID(0)
	leadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req MoveLeadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	lead, err := h.crmRepo.GetLeadByID(ctx, leadID, accountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}

	stage, err := h.crmRepo.GetStageByID(ctx, req.StageID, accountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "stage not found")
		return
	}

	isCTWALead := false
	if lead.TrackedUser != nil {
		tu := lead.TrackedUser
		isCTWALead = tu.CtwaClid != "" || tu.WaAdID != "" || (tu.IDWpp != "" && strings.HasPrefix(tu.ExternalID, "wpp:"))
	}
	targetEvent := stage.TriggerEvent
	if isCTWALead && stage.TriggerEventCTWA != "" {
		targetEvent = stage.TriggerEventCTWA
	}

	if targetEvent == domain.EventPurchase && req.Value <= 0 {
		writeError(w, http.StatusBadRequest, "value required for purchase event")
		return
	}

	if err := h.crmRepo.UpdateLeadStageAndValue(ctx, leadID, accountID, req.StageID, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to move lead")
		return
	}

	hasTrigger := stage.TriggerEvent != "" || stage.TriggerEventCTWA != ""
	if hasTrigger && (lead.StageID == nil || *lead.StageID != req.StageID) {
		h.dispatchCRMStageTrigger(ctx, accountID, stage, lead.TrackedUser, req.Value)
	}

	h.notifyCRMUpdated(ctx, accountID, "lead_moved")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type UpdateLeadReq struct {
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	Value     float64 `json:"value"`
}

// UpdateLead atualiza dados PII do lead (TrackedUser), unifica group_id e persiste o Valor (CRMLead).
func (h *AdminCRM) UpdateLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := TenantFromContext(ctx)
	accountID := scope.EffectiveAccountID(0)
	leadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req UpdateLeadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	externalID, normPhone, normEmail, err := h.crmRepo.UpdateLeadPIIAndValue(
		ctx, leadID, accountID, req.FirstName, req.LastName, req.Email, req.Phone, req.Value,
	)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lead not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update lead")
		return
	}

	if h.users != nil && externalID != "" && (normPhone != "" || normEmail != "") {
		if _, lerr := h.users.ResolveAndLink(ctx, accountID, externalID, normPhone, normEmail); lerr != nil {
			slog.Warn("resolve identity link on crm update failed", "err", lerr, "external_id", externalID)
		}
	}

	h.notifyCRMUpdated(ctx, accountID, "lead_updated")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminCRM) GetLeadDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	leadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	lead, err := h.crmRepo.GetLeadByID(ctx, leadID, accountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}

	writeJSON(w, http.StatusOK, lead)
}

func (h *AdminCRM) DeleteLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if isEmployeeRole(ctx) {
		writeError(w, http.StatusForbidden, "funcionários não podem excluir leads")
		return
	}
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	leadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.crmRepo.DeleteLead(ctx, leadID, accountID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lead not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete lead")
		return
	}

	h.notifyCRMUpdated(ctx, accountID, "lead_deleted")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminCRM) AddNote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	leadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var note domain.CRMNote
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	note.Content = strings.TrimSpace(note.Content)
	if note.Content == "" {
		writeError(w, http.StatusBadRequest, "note content is required")
		return
	}

	if _, err := h.crmRepo.GetLeadByID(ctx, leadID, accountID); err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}

	note.CRMLeadID = leadID
	if err := h.crmRepo.AddNote(ctx, &note); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add note")
		return
	}

	h.notifyCRMUpdated(ctx, accountID, "note_added")
	writeJSON(w, http.StatusOK, note)
}

func (h *AdminCRM) DeleteNote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := TenantFromContext(ctx).EffectiveAccountID(0)
	noteID, err := strconv.ParseInt(chi.URLParam(r, "note_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}

	if err := h.crmRepo.DeleteNote(ctx, noteID, accountID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete note")
		return
	}
	h.notifyCRMUpdated(ctx, accountID, "note_deleted")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
