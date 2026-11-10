package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminAccounts gerencia o CRUD de empresas (multi-tenant).
type AdminAccounts struct {
	repo *repo.AccountRepo
}

func NewAdminAccounts(r *repo.AccountRepo) *AdminAccounts {
	return &AdminAccounts{repo: r}
}

type accountInput struct {
	Name                       string `json:"name"`
	Slug                       string `json:"slug"`
	Enabled                    *bool  `json:"enabled"`
	SentEventsRetentionDays    *int   `json:"sent_events_retention_days"`
	EventCountersRetentionDays *int   `json:"event_counters_retention_days"`
	UpdateRetention            bool   `json:"update_retention"`
}

func (h *AdminAccounts) List(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	list, err := h.repo.ListByHierarchy(r.Context(), c)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (h *AdminAccounts) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims, _ := auth.ClaimsFrom(r.Context())
	if !h.repo.CanAccess(r.Context(), claims, id) {
		writeError(w, http.StatusForbidden, "sem acesso a esta empresa")
		return
	}

	a, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "empresa não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *AdminAccounts) Create(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if c.Role == "employee" {
		writeError(w, http.StatusForbidden, "funcionários não podem criar empresas")
		return
	}

	var in accountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "o nome da empresa é obrigatório")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		slug = slugify(in.Name)
	}
	if !validSlug(slug) {
		writeError(w, http.StatusBadRequest, "identificador (slug) inválido — use apenas letras minúsculas, números e hífen (mín. 2 caracteres)")
		return
	}

	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	a := &domain.Account{
		Name:                       in.Name,
		Slug:                       slug,
		Enabled:                    enabled,
		OwnerID:                    &c.UserID,
		SentEventsRetentionDays:    in.SentEventsRetentionDays,
		EventCountersRetentionDays: in.EventCountersRetentionDays,
	}
	if err := h.repo.Create(r.Context(), a); err != nil {
		if isDuplicateKey(err) {
			writeError(w, http.StatusConflict, "este identificador (slug) já está em uso por outra empresa")
			return
		}
		writeError(w, http.StatusInternalServerError, "create error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *AdminAccounts) Update(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role == "employee" {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem editar empresas")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if !h.repo.CanAccess(r.Context(), c, id) {
		writeError(w, http.StatusForbidden, "sem permissão para editar esta empresa")
		return
	}

	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "empresa não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}

	var in accountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		existing.Name = name
	}
	if in.Slug != "" {
		s := strings.ToLower(strings.TrimSpace(in.Slug))
		if !validSlug(s) {
			writeError(w, http.StatusBadRequest, "identificador (slug) inválido — use apenas letras minúsculas, números e hífen")
			return
		}
		existing.Slug = s
	}
	if in.Enabled != nil {
		existing.Enabled = *in.Enabled
	}
	if in.UpdateRetention {
		existing.SentEventsRetentionDays = in.SentEventsRetentionDays
		existing.EventCountersRetentionDays = in.EventCountersRetentionDays
	} else {
		if in.SentEventsRetentionDays != nil {
			existing.SentEventsRetentionDays = in.SentEventsRetentionDays
		}
		if in.EventCountersRetentionDays != nil {
			existing.EventCountersRetentionDays = in.EventCountersRetentionDays
		}
	}
	if err := h.repo.Update(r.Context(), existing); err != nil {
		if isDuplicateKey(err) {
			writeError(w, http.StatusConflict, "este identificador (slug) já está em uso por outra empresa")
			return
		}
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *AdminAccounts) Delete(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role == "employee" {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem remover empresas")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if id == 1 {
		writeError(w, http.StatusBadRequest, "a empresa principal (ID #1) não pode ser removida")
		return
	}
	if !h.repo.CanAccess(r.Context(), c, id) {
		writeError(w, http.StatusForbidden, "sem permissão para remover esta empresa")
		return
	}

	var body struct {
		Confirm bool `json:"confirm"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if !body.Confirm {
		writeError(w, http.StatusConflict, "operação destrutiva: envie {\"confirm\": true}")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, "erro ao remover empresa: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
