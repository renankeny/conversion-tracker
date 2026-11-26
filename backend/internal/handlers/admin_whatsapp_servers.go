package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type AdminWhatsAppServers struct {
	repo *repo.WhatsAppRepo
}

func NewAdminWhatsAppServers(repo *repo.WhatsAppRepo) *AdminWhatsAppServers {
	return &AdminWhatsAppServers{repo: repo}
}

func validateEvolutionURL(rawURL string) (string, bool) {
	clean, err := util.ValidateServiceEndpointURL(rawURL)
	return clean, err == nil
}

// Get retorna a configuração do servidor Evolution Go da conta atual (sem expor a api_key).
func (h *AdminWhatsAppServers) Get(w http.ResponseWriter, r *http.Request) {
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	srv, err := h.repo.GetAccountServer(r.Context(), accID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar servidor WhatsApp")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"server": srv})
}

// Upsert cria ou atualiza o servidor Evolution Go da conta atual.
func (h *AdminWhatsAppServers) Upsert(w http.ResponseWriter, r *http.Request) {
	if isEmployeeRole(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem configurar servidores WhatsApp")
		return
	}
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	var in struct {
		Name   string `json:"name"`
		URL    string `json:"url"`
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	cleanURL, ok := validateEvolutionURL(in.URL)
	if !ok {
		writeError(w, http.StatusBadRequest, "URL inválida ou bloqueada por segurança")
		return
	}

	existing, _ := h.repo.GetAccountServer(r.Context(), accID)
	if strings.TrimSpace(in.APIKey) == "" {
		if existing == nil || existing.URL != cleanURL {
			writeError(w, http.StatusBadRequest, "informe a API Key da Evolution Go ao cadastrar ou alterar a URL do servidor")
			return
		}
	}

	srv := &domain.WhatsAppServer{
		AccountID: accID,
		Name:      in.Name,
		URL:       cleanURL,
		APIKey:    in.APIKey,
		IsActive:  true,
	}
	if err := h.repo.UpsertAccountServer(r.Context(), srv); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar servidor: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"server": srv})
}

// Test testa a conexão e a API Key global com o servidor Evolution Go.
func (h *AdminWhatsAppServers) Test(w http.ResponseWriter, r *http.Request) {
	if isEmployeeRole(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem testar servidores WhatsApp")
		return
	}
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	var in struct {
		URL    string `json:"url"`
		APIKey string `json:"api_key"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)

	targetURL := strings.TrimRight(strings.TrimSpace(in.URL), "/")
	targetKey := strings.TrimSpace(in.APIKey)

	if targetURL == "" || targetKey == "" {
		saved, err := h.repo.GetAccountServer(r.Context(), accID)
		if err != nil || saved == nil {
			writeError(w, http.StatusBadRequest, "informe a URL e API Key ou salve o servidor primeiro")
			return
		}
		if targetURL == "" {
			targetURL = saved.URL
		} else if targetKey == "" && targetURL != strings.TrimRight(strings.TrimSpace(saved.URL), "/") {
			writeError(w, http.StatusBadRequest, "informe a API Key ao testar uma URL diferente da salva")
			return
		}
		if targetKey == "" {
			targetKey = saved.APIKey
		}
	}

	cleanURL, ok := validateEvolutionURL(targetURL)
	if !ok {
		writeError(w, http.StatusBadRequest, "URL inválida")
		return
	}

	cli := evolution.NewClient(cleanURL, targetKey)
	health, err := cli.HealthCheck(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "falha ao conectar na Evolution Go: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"healthy":        true,
		"version":        health.Version,
		"database":       health.Database,
		"instance_count": health.InstanceCount,
	})
}

func (h *AdminWhatsAppServers) Delete(w http.ResponseWriter, r *http.Request) {
	if isEmployeeRole(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem excluir servidores WhatsApp")
		return
	}
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	count, err := h.repo.CountInstances(r.Context(), accID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao verificar instâncias")
		return
	}
	if count > 0 {
		writeError(w, http.StatusConflict, "remova as instâncias conectadas antes de excluir o servidor")
		return
	}
	if err := h.repo.DeleteAccountServer(r.Context(), accID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir servidor")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}
