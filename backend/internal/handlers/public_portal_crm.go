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

func (h *PublicDashboard) resolvePortalCRM(w http.ResponseWriter, r *http.Request, requireEdit bool) (*domain.Pixel, bool) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return nil, false
	}
	if !p.DashboardConfig.IsPublicCRMEnabled() {
		writeError(w, http.StatusForbidden, "aba de CRM desativada para este link")
		return nil, false
	}
	if requireEdit && !p.DashboardConfig.IsPublicCRMEditable() {
		writeError(w, http.StatusForbidden, "este link está configurado como somente visualização")
		return nil, false
	}
	if h.crmH == nil || h.crmH.crmRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "serviço de CRM indisponível")
		return nil, false
	}
	return p, true
}

func (h *PublicDashboard) loadPortalLead(w http.ResponseWriter, r *http.Request, p *domain.Pixel) (int64, *domain.CRMLead, bool) {
	leadID, err := strconv.ParseInt(chi.URLParam(r, "leadID"), 10, 64)
	if err != nil || leadID <= 0 {
		writeError(w, http.StatusBadRequest, "identificador inválido")
		return 0, nil, false
	}
	belongs, err := h.crmH.crmRepo.LeadBelongsToPixel(r.Context(), leadID, p.AccountID, p.ID, p.Name)
	if err != nil || !belongs {
		writeError(w, http.StatusNotFound, "lead não encontrado neste projeto")
		return 0, nil, false
	}
	lead, err := h.crmH.crmRepo.GetLeadByID(r.Context(), leadID, p.AccountID)
	if err != nil || lead == nil {
		writeError(w, http.StatusNotFound, "lead não encontrado")
		return 0, nil, false
	}
	return leadID, lead, true
}

// CRMBoard GET /public/dashboard/{token}/crm/board
func (h *PublicDashboard) CRMBoard(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, false)
	if !ok {
		return
	}
	ctx := r.Context()
	_ = h.crmH.crmRepo.EnsureDefaultStages(ctx, p.AccountID)

	stages, err := h.crmH.crmRepo.GetStages(ctx, p.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar etapas")
		return
	}
	leads, err := h.crmH.crmRepo.GetLeads(ctx, p.AccountID, p.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar leads")
		return
	}
	if leads == nil {
		leads = []domain.CRMLead{}
	}
	tags, _ := h.crmH.crmRepo.ListTags(ctx, p.AccountID)
	if tags == nil {
		tags = []domain.CRMTag{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"stages":     stages,
		"leads":      leads,
		"tags":       tags,
		"allow_edit": p.DashboardConfig.IsPublicCRMEditable(),
	})
}

// CRMLeadDetails GET /public/dashboard/{token}/crm/leads/{leadID}
func (h *PublicDashboard) CRMLeadDetails(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, false)
	if !ok {
		return
	}
	_, lead, ok := h.loadPortalLead(w, r, p)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, lead)
}

// CRMMoveLead PUT /public/dashboard/{token}/crm/leads/{leadID}/move
func (h *PublicDashboard) CRMMoveLead(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, true)
	if !ok {
		return
	}
	leadID, lead, ok := h.loadPortalLead(w, r, p)
	if !ok {
		return
	}
	var req MoveLeadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "dados inválidos")
		return
	}
	ctx := r.Context()
	stage, err := h.crmH.crmRepo.GetStageByID(ctx, req.StageID, p.AccountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "etapa não encontrada")
		return
	}

	isCTWALead := domain.IsWhatsAppOrCTWAUser(lead.TrackedUser)
	targetEvent := domain.EffectiveStageTrigger(stage, isCTWALead)
	if targetEvent == domain.EventPurchase && req.Value <= 0 {
		writeError(w, http.StatusBadRequest, "informe o valor da venda")
		return
	}
	if err := h.crmH.crmRepo.UpdateLeadStageAndValue(ctx, leadID, p.AccountID, req.StageID, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao mover lead")
		return
	}
	if targetEvent != "" && (lead.StageID == nil || *lead.StageID != req.StageID) {
		h.crmH.dispatchCRMStageTrigger(ctx, p.AccountID, leadID, stage, lead.TrackedUser, req.Value)
	}
	h.crmH.notifyCRMUpdated(ctx, p.AccountID, "lead_moved")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CRMUpdateLead PUT /public/dashboard/{token}/crm/leads/{leadID}
func (h *PublicDashboard) CRMUpdateLead(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, true)
	if !ok {
		return
	}
	leadID, _, ok := h.loadPortalLead(w, r, p)
	if !ok {
		return
	}
	var req UpdateLeadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "dados inválidos")
		return
	}
	ctx := r.Context()
	extID, normPhone, normEmail, err := h.crmH.crmRepo.UpdateLeadPIIAndValue(
		ctx, leadID, p.AccountID, req.FirstName, req.LastName, req.Email, req.Phone, req.Value,
	)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lead não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao atualizar lead")
		return
	}
	if h.crmH.users != nil && extID != "" && (normPhone != "" || normEmail != "") {
		if _, lerr := h.crmH.users.ResolveAndLink(ctx, p.AccountID, extID, normPhone, normEmail); lerr != nil {
			slog.Warn("resolve identity link on public crm update failed", "err", lerr, "external_id", extID)
		}
	}
	h.crmH.notifyCRMUpdated(ctx, p.AccountID, "lead_updated")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CRMCreateLead POST /public/dashboard/{token}/crm/leads
func (h *PublicDashboard) CRMCreateLead(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, true)
	if !ok {
		return
	}
	var req struct {
		FirstName string  `json:"first_name"`
		LastName  string  `json:"last_name"`
		Email     string  `json:"email"`
		Phone     string  `json:"phone"`
		StageID   int64   `json:"stage_id"`
		Value     float64 `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.StageID <= 0 {
		writeError(w, http.StatusBadRequest, "dados inválidos")
		return
	}
	ctx := r.Context()
	stage, err := h.crmH.crmRepo.GetStageByID(ctx, req.StageID, p.AccountID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "etapa inválida")
		return
	}
	email := util.NormalizeEmail(req.Email)
	phone := util.NormalizePhone(req.Phone)
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	if email == "" && phone == "" {
		writeError(w, http.StatusBadRequest, "informe telefone ou e-mail")
		return
	}
	tu := h.crmH.resolveOrCreateCRMTrackedUser(ctx, p.AccountID, firstName, lastName, email, phone, "", p.Name)
	if tu == nil {
		writeError(w, http.StatusInternalServerError, "erro ao criar contato")
		return
	}
	if tu.PixelName == "" {
		tu.PixelName = p.Name
		_ = h.crmH.users.Upsert(ctx, tu)
	}
	effEvent := domain.EffectiveStageTrigger(stage, domain.IsWhatsAppOrCTWAUser(tu))
	if effEvent == domain.EventPurchase && req.Value <= 0 {
		writeError(w, http.StatusBadRequest, "informe o valor da venda")
		return
	}
	leadID, prevStageID, err := h.crmH.crmRepo.CreateOrUpdateLeadAPI(ctx, p.AccountID, tu.ID, req.StageID, req.Value)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar lead")
		return
	}
	if effEvent != "" && (prevStageID == nil || *prevStageID != req.StageID) {
		h.crmH.dispatchCRMStageTrigger(ctx, p.AccountID, leadID, stage, tu, req.Value)
	}
	h.crmH.notifyCRMUpdated(ctx, p.AccountID, "lead_created")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "lead_id": leadID, "stage_id": req.StageID})
}
