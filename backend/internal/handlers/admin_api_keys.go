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
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminAPIKeys gerencia o ciclo completo de chaves S2S e MCP (listar com plaintext cifrado,
// criar, editar permissões/nome, regerar segredo e revogar).
type AdminAPIKeys struct {
	repo   *repo.APIKeyRepo
	notify *notify.Service
}

func NewAdminAPIKeys(r *repo.APIKeyRepo) *AdminAPIKeys {
	return &AdminAPIKeys{repo: r}
}

func (h *AdminAPIKeys) SetNotify(n *notify.Service) { h.notify = n }

func requireAPIKeysManager(w http.ResponseWriter, r *http.Request) bool {
	if !auth.CanManageUsers(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem gerenciar API Keys")
		return false
	}
	return true
}

func (h *AdminAPIKeys) List(w http.ResponseWriter, r *http.Request) {
	if !requireAPIKeysManager(w, r) {
		return
	}
	scope := TenantFromContext(r.Context())
	list, err := h.repo.ListByAccount(r.Context(), scope.EffectiveAccountID(0))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar chaves de API")
		return
	}
	if list == nil {
		list = []domain.APIKey{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

type apiKeyUpsertInput struct {
	Name      string   `json:"name"`
	AccountID *int64   `json:"account_id"`
	Scopes    []string `json:"scopes"`
	ReadOnly  bool     `json:"read_only"`
}

func (h *AdminAPIKeys) Create(w http.ResponseWriter, r *http.Request) {
	if !requireAPIKeysManager(w, r) {
		return
	}
	var in apiKeyUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "nome da chave é obrigatório")
		return
	}
	scope := TenantFromContext(r.Context())
	override := int64(0)
	if in.AccountID != nil {
		override = *in.AccountID
	}
	accountID := scope.EffectiveAccountID(override)
	if !scope.IsOwner && accountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a essa empresa")
		return
	}

	key, plaintext, err := h.repo.Create(r.Context(), accountID, name, in.Scopes, in.ReadOnly)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	emitAPIKeyLifecycleNotification(h.notify, accountID, key.Name, true)
	writeJSON(w, http.StatusCreated, map[string]any{
		"key":       key,
		"plaintext": plaintext,
	})
}

func (h *AdminAPIKeys) Update(w http.ResponseWriter, r *http.Request) {
	if !requireAPIKeysManager(w, r) {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "chave não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar chave")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a essa chave")
		return
	}

	var in apiKeyUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "nome da chave é obrigatório")
		return
	}

	updated, err := h.repo.Update(r.Context(), id, name, in.Scopes, in.ReadOnly)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"key": updated})
}

func (h *AdminAPIKeys) Regenerate(w http.ResponseWriter, r *http.Request) {
	if !requireAPIKeysManager(w, r) {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "chave não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar chave")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a essa chave")
		return
	}

	updated, plaintext, err := h.repo.RegenerateSecret(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key":       updated,
		"plaintext": plaintext,
	})
}

func (h *AdminAPIKeys) Revoke(w http.ResponseWriter, r *http.Request) {
	if !requireAPIKeysManager(w, r) {
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
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	if err := h.repo.Revoke(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "revoke error: "+err.Error())
		return
	}
	emitAPIKeyLifecycleNotification(h.notify, existing.AccountID, existing.Name, false)
	w.WriteHeader(http.StatusNoContent)
}
