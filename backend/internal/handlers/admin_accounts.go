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

// AdminAccounts gerencia o CRUD de contas (multi-tenant). Apenas owners
// podem criar/editar/remover. Members só conseguem ver a sua própria.
type AdminAccounts struct {
	repo *repo.AccountRepo
}

func NewAdminAccounts(r *repo.AccountRepo) *AdminAccounts {
	return &AdminAccounts{repo: r}
}

type accountInput struct {
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Enabled *bool  `json:"enabled"`
}

func (h *AdminAccounts) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())

	if scope.IsOwner {
		list, err := h.repo.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": list})
		return
	}

	// Member só vê a própria conta.
	a, err := h.repo.FindByID(r.Context(), scope.AccountID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{"data": []domain.Account{}})
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": []domain.Account{*a}})
}

func (h *AdminAccounts) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims, _ := auth.ClaimsFrom(r.Context())
	if !claims.HasAccountAccess(id) {
		writeError(w, http.StatusForbidden, "sem acesso a esta conta")
		return
	}

	a, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *AdminAccounts) Create(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner {
		writeError(w, http.StatusForbidden, "apenas owners podem criar contas")
		return
	}

	var in accountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, http.StatusBadRequest, "name obrigatório")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		slug = slugify(in.Name)
	}
	if !validSlug(slug) {
		writeError(w, http.StatusBadRequest, "slug inválido (use a-z, 0-9, -)")
		return
	}

	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	a := &domain.Account{Name: in.Name, Slug: slug, Enabled: enabled}
	if err := h.repo.Create(r.Context(), a); err != nil {
		if isDuplicateKey(err) {
			writeError(w, http.StatusConflict, "slug já existe")
			return
		}
		writeError(w, http.StatusInternalServerError, "create error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *AdminAccounts) Update(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner {
		writeError(w, http.StatusForbidden, "apenas owners podem editar contas")
		return
	}

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
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}

	var in accountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if in.Name != "" {
		existing.Name = in.Name
	}
	if in.Slug != "" {
		s := strings.ToLower(strings.TrimSpace(in.Slug))
		if !validSlug(s) {
			writeError(w, http.StatusBadRequest, "slug inválido")
			return
		}
		existing.Slug = s
	}
	if in.Enabled != nil {
		existing.Enabled = *in.Enabled
	}
	if err := h.repo.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *AdminAccounts) Delete(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner {
		writeError(w, http.StatusForbidden, "apenas owners podem remover contas")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
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
		writeError(w, http.StatusBadRequest, "delete error: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validSlug(s string) bool {
	if len(s) < 2 {
		return false
	}
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}
	return true
}

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	out := make([]rune, 0, len(s))
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else if c == ' ' || c == '_' {
			out = append(out, '-')
		}
	}
	result := string(out)
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}
