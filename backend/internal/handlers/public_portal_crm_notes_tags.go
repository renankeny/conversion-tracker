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
)

// CRMAddNote POST /public/dashboard/{token}/crm/leads/{leadID}/notes
func (h *PublicDashboard) CRMAddNote(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, true)
	if !ok {
		return
	}
	leadID, _, ok := h.loadPortalLead(w, r, p)
	if !ok {
		return
	}
	var note domain.CRMNote
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		writeError(w, http.StatusBadRequest, "dados inválidos")
		return
	}
	note.Content = strings.TrimSpace(note.Content)
	if note.Content == "" {
		writeError(w, http.StatusBadRequest, "conteúdo da nota é obrigatório")
		return
	}
	note.CRMLeadID = leadID
	if err := h.crmH.crmRepo.AddNote(r.Context(), &note); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar anotação")
		return
	}
	h.crmH.notifyCRMUpdated(r.Context(), p.AccountID, "note_added")
	writeJSON(w, http.StatusOK, note)
}

// CRMDeleteNote DELETE /public/dashboard/{token}/crm/leads/{leadID}/notes/{noteID}
func (h *PublicDashboard) CRMDeleteNote(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, true)
	if !ok {
		return
	}
	leadID, lead, ok := h.loadPortalLead(w, r, p)
	if !ok {
		return
	}
	noteID, err := strconv.ParseInt(chi.URLParam(r, "noteID"), 10, 64)
	if err != nil || noteID <= 0 {
		writeError(w, http.StatusBadRequest, "identificador de nota inválido")
		return
	}
	noteBelongs := false
	for _, n := range lead.Notes {
		if n.ID == noteID && (n.CRMLeadID == 0 || n.CRMLeadID == leadID) {
			noteBelongs = true
			break
		}
	}
	if !noteBelongs {
		writeError(w, http.StatusNotFound, "anotação não encontrada")
		return
	}
	if err := h.crmH.crmRepo.DeleteNote(r.Context(), noteID, p.AccountID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao remover anotação")
		return
	}
	h.crmH.notifyCRMUpdated(r.Context(), p.AccountID, "note_deleted")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CRMCreateTag POST /public/dashboard/{token}/crm/tags
func (h *PublicDashboard) CRMCreateTag(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, true)
	if !ok {
		return
	}
	var tag domain.CRMTag
	if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
		writeError(w, http.StatusBadRequest, "dados inválidos")
		return
	}
	tag.Name = strings.TrimSpace(tag.Name)
	if tag.Name == "" {
		writeError(w, http.StatusBadRequest, "nome da tag é obrigatório")
		return
	}
	if strings.TrimSpace(tag.Color) == "" {
		tag.Color = "#3B82F6"
	}
	tag.AccountID = p.AccountID
	if err := h.crmH.crmRepo.CreateTag(r.Context(), &tag); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao criar tag")
		return
	}
	h.crmH.notifyCRMUpdated(r.Context(), p.AccountID, "tag_created")
	writeJSON(w, http.StatusOK, tag)
}

// CRMAssignTag POST /public/dashboard/{token}/crm/leads/{leadID}/tags/{tagID}
func (h *PublicDashboard) CRMAssignTag(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, true)
	if !ok {
		return
	}
	leadID, _, ok := h.loadPortalLead(w, r, p)
	if !ok {
		return
	}
	tagID, err := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)
	if err != nil || tagID <= 0 {
		writeError(w, http.StatusBadRequest, "tag inválida")
		return
	}
	if err := h.crmH.crmRepo.AssignTag(r.Context(), p.AccountID, leadID, tagID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tag não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao vincular tag")
		return
	}
	h.crmH.notifyCRMUpdated(r.Context(), p.AccountID, "tag_assigned")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CRMRemoveTag DELETE /public/dashboard/{token}/crm/leads/{leadID}/tags/{tagID}
func (h *PublicDashboard) CRMRemoveTag(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolvePortalCRM(w, r, true)
	if !ok {
		return
	}
	leadID, _, ok := h.loadPortalLead(w, r, p)
	if !ok {
		return
	}
	tagID, err := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)
	if err != nil || tagID <= 0 {
		writeError(w, http.StatusBadRequest, "tag inválida")
		return
	}
	if err := h.crmH.crmRepo.RemoveTag(r.Context(), p.AccountID, leadID, tagID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao remover tag")
		return
	}
	h.crmH.notifyCRMUpdated(r.Context(), p.AccountID, "tag_removed")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
