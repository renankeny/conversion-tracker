package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/waha"
)

type AdminWhatsAppServers struct {
	repo *repo.WhatsAppRepo
}

func NewAdminWhatsAppServers(repo *repo.WhatsAppRepo) *AdminWhatsAppServers {
	return &AdminWhatsAppServers{repo: repo}
}

func validateDirectServerURL(rawURL string) (string, bool) {
	clean, err := util.ValidateServiceEndpointURL(rawURL)
	return clean, err == nil
}

func canManageGlobalWhatsAppServer(ctx context.Context) bool {
	c, ok := auth.ClaimsFrom(ctx)
	if !ok || c == nil {
		return false
	}
	return c.IsOwner() || c.Role == "admin"
}

func providerDisplayName(provider string) string {
	if domain.NormalizeWhatsAppServerProvider(provider) == domain.WhatsAppProviderWAHA {
		return "WAHA API"
	}
	return "Evolution Go"
}

// Get retorna os servidores globais de Conexão Direta (Evolution Go e WAHA API), sem expor api_key.
func (h *AdminWhatsAppServers) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	evoSrv, err := h.repo.GetGlobalServerByProvider(ctx, domain.WhatsAppProviderEvolutionGo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar servidor Evolution Go")
		return
	}
	wahaSrv, err := h.repo.GetGlobalServerByProvider(ctx, domain.WhatsAppProviderWAHA)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar servidor WAHA")
		return
	}
	selected := evoSrv
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("provider")), domain.WhatsAppProviderWAHA) {
		selected = wahaSrv
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"server": selected,
		"servers": map[string]any{
			domain.WhatsAppProviderEvolutionGo: evoSrv,
			domain.WhatsAppProviderWAHA:        wahaSrv,
		},
	})
}

// Upsert cria ou atualiza o servidor global de Evolution Go ou WAHA API.
func (h *AdminWhatsAppServers) Upsert(w http.ResponseWriter, r *http.Request) {
	if !canManageGlobalWhatsAppServer(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores da plataforma podem configurar o servidor global WhatsApp")
		return
	}
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	var in struct {
		Provider string `json:"provider"`
		Name     string `json:"name"`
		URL      string `json:"url"`
		APIKey   string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	provider := domain.NormalizeWhatsAppServerProvider(in.Provider)
	cleanURL, ok := validateDirectServerURL(in.URL)
	if !ok {
		writeError(w, http.StatusBadRequest, "URL inválida ou bloqueada por segurança")
		return
	}

	existing, _ := h.repo.GetGlobalServerByProvider(r.Context(), provider)
	if strings.TrimSpace(in.APIKey) == "" {
		if existing == nil || existing.URL != cleanURL {
			writeError(w, http.StatusBadRequest, "informe a API Key da "+providerDisplayName(provider)+" ao cadastrar ou alterar a URL do servidor")
			return
		}
	}

	srv := &domain.WhatsAppServer{
		AccountID: accID,
		Provider:  provider,
		Name:      in.Name,
		URL:       cleanURL,
		APIKey:    in.APIKey,
		IsActive:  true,
	}
	if err := h.repo.UpsertGlobalServer(r.Context(), srv); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar servidor: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"server": srv})
}

// Test testa a conexão e a API Key global do servidor Evolution Go ou WAHA API.
func (h *AdminWhatsAppServers) Test(w http.ResponseWriter, r *http.Request) {
	if isEmployeeRole(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem testar o servidor WhatsApp")
		return
	}
	var in struct {
		Provider string `json:"provider"`
		URL      string `json:"url"`
		APIKey   string `json:"api_key"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)

	provider := domain.NormalizeWhatsAppServerProvider(in.Provider)
	targetURL := strings.TrimRight(strings.TrimSpace(in.URL), "/")
	targetKey := strings.TrimSpace(in.APIKey)

	if targetURL == "" || targetKey == "" {
		saved, err := h.repo.GetGlobalServerByProvider(r.Context(), provider)
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

	cleanURL, ok := validateDirectServerURL(targetURL)
	if !ok {
		writeError(w, http.StatusBadRequest, "URL inválida")
		return
	}

	if provider == domain.WhatsAppProviderWAHA {
		health, err := waha.NewClient(cleanURL, targetKey).HealthCheck(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "falha ao conectar na WAHA API: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"healthy":        true,
			"version":        health.Version,
			"engine":         health.Engine,
			"database":       health.Tier,
			"instance_count": health.InstanceCount,
		})
		return
	}

	health, err := evolution.NewClient(cleanURL, targetKey).HealthCheck(r.Context())
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
	if !canManageGlobalWhatsAppServer(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores da plataforma podem excluir o servidor global WhatsApp")
		return
	}
	provider := domain.NormalizeWhatsAppServerProvider(r.URL.Query().Get("provider"))
	count, err := h.repo.CountInstancesByServerProvider(r.Context(), provider)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao verificar instâncias")
		return
	}
	if count > 0 {
		writeError(w, http.StatusConflict, "remova as instâncias "+providerDisplayName(provider)+" conectadas nas empresas antes de excluir o servidor")
		return
	}
	if err := h.repo.DeleteGlobalServerByProvider(r.Context(), provider); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir servidor")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}
