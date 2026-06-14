package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminAPIKeys gerencia o CRUD de tokens server-to-server.
//
// Owner enxerga todas; member só vê as da própria conta. Owners podem
// criar keys para qualquer conta; members só pra própria.
type AdminAPIKeys struct {
	repo *repo.APIKeyRepo
}

func NewAdminAPIKeys(r *repo.APIKeyRepo) *AdminAPIKeys {
	return &AdminAPIKeys{repo: r}
}

func (h *AdminAPIKeys) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	accountID := int64(0)
	if !scope.IsOwner {
		accountID = scope.AccountID
	}
	list, err := h.repo.ListByAccount(r.Context(), accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

type apiKeyCreateInput struct {
	Name      string `json:"name"`
	AccountID *int64 `json:"account_id"`
}

func (h *AdminAPIKeys) Create(w http.ResponseWriter, r *http.Request) {
	var in apiKeyCreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, http.StatusBadRequest, "name obrigatório")
		return
	}
	scope := TenantFromContext(r.Context())
	override := int64(0)
	if in.AccountID != nil {
		override = *in.AccountID
	}
	accountID := scope.EffectiveAccountID(override)
	// Defesa em profundidade: members não podem criar key para outra conta.
	if !scope.IsOwner && accountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a essa conta")
		return
	}

	key, plaintext, err := h.repo.Create(r.Context(), accountID, strings.TrimSpace(in.Name))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create error: "+err.Error())
		return
	}

	// IMPORTANTE: o plaintext só é exposto aqui, na resposta da criação.
	writeJSON(w, http.StatusCreated, map[string]any{
		"key":       key,
		"plaintext": plaintext, // mostrar 1 vez no painel
	})
}

// Revoke marca a key como revogada. Operação idempotente (sem confirmação
// dupla porque é reversível só restaurando do backup; criar nova key é
// trivial).
func (h *AdminAPIKeys) Revoke(w http.ResponseWriter, r *http.Request) {
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
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	if err := h.repo.Revoke(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "revoke error: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
