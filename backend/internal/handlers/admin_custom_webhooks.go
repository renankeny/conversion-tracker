package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminCustomWebhooks struct {
	repo    *repo.CustomWebhookRepo
	logRepo *repo.WebhookLogRepo
}

func NewAdminCustomWebhooks(r *repo.CustomWebhookRepo, logRepo *repo.WebhookLogRepo) *AdminCustomWebhooks {
	return &AdminCustomWebhooks{repo: r, logRepo: logRepo}
}

func (h *AdminCustomWebhooks) List(w http.ResponseWriter, r *http.Request) {
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	list, err := h.repo.ListByAccount(r.Context(), accID)
	if err != nil {
		slog.Error("list custom webhooks", "err", err)
		writeError(w, http.StatusInternalServerError, "erro ao listar webhooks customizados")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *AdminCustomWebhooks) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)

	t, err := h.repo.GetByAccount(r.Context(), id, accID)
	if err != nil {
		writeError(w, http.StatusNotFound, "webhook não encontrado")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

type CustomWebhookPayload struct {
	Name          string          `json:"name"`
	MappingConfig json.RawMessage `json:"mapping_config"`
	IsGlobal      bool            `json:"is_global"`
}

func (h *AdminCustomWebhooks) Create(w http.ResponseWriter, r *http.Request) {
	if isEmployeeRole(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem criar integrações")
		return
	}
	var body CustomWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "nome é obrigatório")
		return
	}
	if len(body.MappingConfig) == 0 {
		body.MappingConfig = []byte("{}")
	}

	scope := TenantFromContext(r.Context())
	var accountID *int64
	if scope.IsOwner && body.IsGlobal {
		accountID = nil
	} else {
		acc := scope.EffectiveAccountID(0)
		accountID = &acc
	}

	t := &repo.CustomWebhookTemplate{
		AccountID:     accountID,
		Name:          body.Name,
		MappingConfig: body.MappingConfig,
	}

	if err := h.repo.Create(r.Context(), t); err != nil {
		slog.Error("create custom webhook", "err", err)
		writeError(w, http.StatusInternalServerError, "erro ao criar integração customizada")
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *AdminCustomWebhooks) Update(w http.ResponseWriter, r *http.Request) {
	if isEmployeeRole(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem editar integrações")
		return
	}
	id := chi.URLParam(r, "id")
	var body CustomWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	scope := TenantFromContext(r.Context())
	accID := scope.EffectiveAccountID(0)

	existing, err := h.repo.GetByAccount(r.Context(), id, accID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if existing.AccountID == nil && !scope.IsOwner {
		writeError(w, http.StatusForbidden, "apenas superadmin pode editar templates globais")
		return
	}

	if strings.TrimSpace(body.Name) != "" {
		existing.Name = strings.TrimSpace(body.Name)
	}
	if len(body.MappingConfig) > 0 {
		existing.MappingConfig = body.MappingConfig
	}

	if err := h.repo.Update(r.Context(), existing); err != nil {
		slog.Error("update custom webhook", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "erro ao atualizar integração customizada")
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *AdminCustomWebhooks) Delete(w http.ResponseWriter, r *http.Request) {
	if isEmployeeRole(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem excluir integrações")
		return
	}
	id := chi.URLParam(r, "id")
	scope := TenantFromContext(r.Context())
	accID := scope.EffectiveAccountID(0)

	existing, err := h.repo.GetByAccount(r.Context(), id, accID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if existing.AccountID == nil && !scope.IsOwner {
		writeError(w, http.StatusForbidden, "apenas superadmin pode deletar templates globais")
		return
	}

	if err := h.repo.Delete(r.Context(), id, existing.AccountID); err != nil {
		slog.Error("delete custom webhook", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "erro ao excluir integração customizada")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Logs retorna o histórico de auditoria de uma integração
func (h *AdminCustomWebhooks) Logs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)

	logs, err := h.logRepo.ListByCustomWebhook(r.Context(), id, accID, 50)
	if err != nil {
		slog.Error("list custom webhook logs", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "erro ao buscar logs")
		return
	}
	if logs == nil {
		logs = []repo.CustomWebhookAuditLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}
