package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminWhatsAppServers struct {
	repo *repo.WhatsAppRepo
}

func NewAdminWhatsAppServers(repo *repo.WhatsAppRepo) *AdminWhatsAppServers {
	return &AdminWhatsAppServers{repo: repo}
}

func validateEvolutionURL(rawURL string) (string, bool) {
	s := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}
	return s, true
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
		writeError(w, http.StatusBadRequest, "URL inválida (informe http:// ou https://)")
		return
	}

	existing, _ := h.repo.GetAccountServer(r.Context(), accID)
	if existing == nil && strings.TrimSpace(in.APIKey) == "" {
		writeError(w, http.StatusBadRequest, "API Key da Evolution Go é obrigatória")
		return
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
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	var in struct {
		URL    string `json:"url"`
		APIKey string `json:"api_key"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)

	targetURL := strings.TrimSpace(in.URL)
	targetKey := strings.TrimSpace(in.APIKey)

	if targetURL == "" || targetKey == "" {
		saved, err := h.repo.GetAccountServer(r.Context(), accID)
		if err != nil || saved == nil {
			writeError(w, http.StatusBadRequest, "informe a URL e API Key ou salve o servidor primeiro")
			return
		}
		if targetURL == "" {
			targetURL = saved.URL
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
