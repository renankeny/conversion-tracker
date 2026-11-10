package handlers

import (
	"encoding/json"
	"net/http"

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
	scope := TenantFromContext(r.Context())
	list, err := h.repo.ListByAccount(r.Context(), scope.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *AdminCustomWebhooks) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scope := TenantFromContext(r.Context())
	
	t, err := h.repo.GetByAccount(r.Context(), id, scope.AccountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "webhook nao encontrado")
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
	var body CustomWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "nome é obrigatório")
		return
	}
	if len(body.MappingConfig) == 0 {
		body.MappingConfig = []byte("{}")
	}

	scope := TenantFromContext(r.Context())
	var accountID *int64
	// Se for owner e marcou is_global, cria como null
	if scope.IsOwner && body.IsGlobal {
		accountID = nil
	} else {
		acc := scope.AccountID
		accountID = &acc
	}

	t := &repo.CustomWebhookTemplate{
		AccountID:     accountID,
		Name:          body.Name,
		MappingConfig: body.MappingConfig,
	}

	if err := h.repo.Create(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, "create error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *AdminCustomWebhooks) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body CustomWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	
	scope := TenantFromContext(r.Context())
	
	// Valida acesso primeiro
	existing, err := h.repo.GetByAccount(r.Context(), id, scope.AccountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	
	// Apenas owner pode editar globais
	if existing.AccountID == nil && !scope.IsOwner {
		writeError(w, http.StatusForbidden, "apenas superadmin pode editar templates globais")
		return
	}

	if body.Name != "" {
		existing.Name = body.Name
	}
	if len(body.MappingConfig) > 0 {
		existing.MappingConfig = body.MappingConfig
	}

	if err := h.repo.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *AdminCustomWebhooks) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scope := TenantFromContext(r.Context())
	
	existing, err := h.repo.GetByAccount(r.Context(), id, scope.AccountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	
	if existing.AccountID == nil && !scope.IsOwner {
		writeError(w, http.StatusForbidden, "apenas superadmin pode deletar templates globais")
		return
	}

	if err := h.repo.Delete(r.Context(), id, scope.AccountID); err != nil {
		writeError(w, http.StatusInternalServerError, "delete error: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Logs retorna o histórico de auditoria de uma integração
func (h *AdminCustomWebhooks) Logs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scope := TenantFromContext(r.Context())

	logs, err := h.logRepo.ListByCustomWebhook(r.Context(), id, scope.AccountID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar logs: "+err.Error())
		return
	}
	if logs == nil {
		logs = []repo.CustomWebhookAuditLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}
