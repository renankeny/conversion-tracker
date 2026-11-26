package util

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// SystemUpdateStatus descreve a disponibilidade de nova versão da plataforma para o superadmin.
type SystemUpdateStatus struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version,omitempty"`
	PublishedAt     string `json:"published_at,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	CanAutoUpdate   bool   `json:"can_auto_update"`
	UpdateStatus    string `json:"update_status,omitempty"`
	UpdateMessage   string `json:"update_message,omitempty"`
	CheckedAt       string `json:"checked_at,omitempty"`
}

type updateCheckPayload struct {
	Identity    string `json:"idn"`
	Domain      string `json:"dom,omitempty"`
	Host        string `json:"hst,omitempty"`
	InstanceUID string `json:"uid,omitempty"`
	AppVersion  string `json:"ver,omitempty"`
}

type updateCheckWireResp struct {
	CurrentVersion  string `json:"cur"`
	LatestVersion   string `json:"lat"`
	PublishedAt     string `json:"pub,omitempty"`
	UpdateAvailable bool   `json:"upd"`
	CanAutoUpdate   bool   `json:"auto"`
	UpdateStatus    string `json:"ust,omitempty"`
	UpdateMessage   string `json:"umsg,omitempty"`
}

var systemUpdateCache struct {
	mu        sync.RWMutex
	status    SystemUpdateStatus
	fetchedAt time.Time
}

func resetSystemUpdateCache() {
	systemUpdateCache.mu.Lock()
	systemUpdateCache.status = SystemUpdateStatus{}
	systemUpdateCache.fetchedAt = time.Time{}
	systemUpdateCache.mu.Unlock()
}

func (c *ShardCalibrator) telemetryEndpoint(name string) string {
	base := strings.TrimRight(ResolveGatewayNodeURL(), "/")
	path := strings.Replace(decodeXORBytes(shardPathBytes), "shard-sync", name, 1)
	return base + path
}

func (c *ShardCalibrator) resolveIdentitySafe(ctx context.Context) string {
	if c.pool != nil {
		if idn := c.primaryIdentity(ctx); idn != "" {
			return idn
		}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.cachedFrame != nil {
		return strings.ToLower(strings.TrimSpace(c.cachedFrame.Identity))
	}
	return ""
}

func isLocalDevWithoutGateway() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") &&
		strings.TrimSpace(os.Getenv("PGX_SHARD_SYNC_URI")) == ""
}

// CheckSystemUpdate consulta se há uma nova build publicada para esta instância.
// Usa cache curto em memória (15 min normal; 10s durante atualização em andamento) e falha de forma segura (fail-open).
func (c *ShardCalibrator) CheckSystemUpdate(ctx context.Context, force bool) SystemUpdateStatus {
	baseStatus := SystemUpdateStatus{CurrentVersion: AppBuildVersion, UpdateAvailable: false}
	if isLocalDevWithoutGateway() {
		return baseStatus
	}

	systemUpdateCache.mu.RLock()
	cached := systemUpdateCache.status
	fetchedAt := systemUpdateCache.fetchedAt
	systemUpdateCache.mu.RUnlock()

	ttl := 15 * time.Minute
	if cached.UpdateStatus == "queued" || cached.UpdateStatus == "updating" {
		ttl = 10 * time.Second
	}
	if !fetchedAt.IsZero() {
		age := time.Since(fetchedAt)
		if (!force && age < ttl) || (force && age < 3*time.Second) {
			return cached
		}
	}

	idn := c.resolveIdentitySafe(ctx)
	if idn == "" {
		return baseStatus
	}

	hostname, _ := os.Hostname()
	sealed, err := c.cipher.sealJSON(updateCheckPayload{
		Identity:    idn,
		Domain:      c.appDomain,
		Host:        hostname,
		InstanceUID: c.instanceUID,
		AppVersion:  AppBuildVersion,
	})
	if err != nil {
		return baseStatus
	}

	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	bodyBytes, _ := json.Marshal(map[string]string{"v": sealed})
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.telemetryEndpoint("update-check"), bytes.NewReader(bodyBytes))
	if err != nil {
		return baseStatus
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		if !fetchedAt.IsZero() {
			return cached
		}
		return baseStatus
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return baseStatus
	}
	var env struct {
		Vector string `json:"v"`
	}
	if json.NewDecoder(resp.Body).Decode(&env) != nil || env.Vector == "" {
		return baseStatus
	}
	var wire updateCheckWireResp
	if err := c.cipher.openJSON(env.Vector, &wire); err != nil {
		return baseStatus
	}

	curVer := wire.CurrentVersion
	if curVer == "" {
		curVer = AppBuildVersion
	}
	now := time.Now()
	result := SystemUpdateStatus{
		CurrentVersion:  curVer,
		LatestVersion:   wire.LatestVersion,
		PublishedAt:     wire.PublishedAt,
		UpdateAvailable: wire.UpdateAvailable,
		CanAutoUpdate:   wire.CanAutoUpdate,
		UpdateStatus:    wire.UpdateStatus,
		UpdateMessage:   wire.UpdateMessage,
		CheckedAt:       now.Format(time.RFC3339),
	}

	systemUpdateCache.mu.Lock()
	systemUpdateCache.status = result
	systemUpdateCache.fetchedAt = now
	systemUpdateCache.mu.Unlock()
	return result
}

// TriggerSelfUpdate solicita a atualização automática desta instância (quando o EasyPanel está vinculado).
func (c *ShardCalibrator) TriggerSelfUpdate(ctx context.Context) (string, string, error) {
	if isLocalDevWithoutGateway() {
		return "", "", errors.New("atualização automática disponível apenas em ambiente de produção")
	}
	idn := c.resolveIdentitySafe(ctx)
	if idn == "" {
		return "", "", errors.New("administrador principal não identificado nesta instalação")
	}
	hostname, _ := os.Hostname()
	sealed, err := c.cipher.sealJSON(updateCheckPayload{
		Identity:    idn,
		Domain:      c.appDomain,
		Host:        hostname,
		InstanceUID: c.instanceUID,
		AppVersion:  AppBuildVersion,
	})
	if err != nil {
		return "", "", errors.New("falha ao preparar solicitação de atualização")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	bodyBytes, _ := json.Marshal(map[string]string{"v": sealed})
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.telemetryEndpoint("self-update"), bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", errors.New("falha ao iniciar solicitação de atualização")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", errors.New("serviço de atualização temporariamente indisponível; tente novamente em instantes")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errEnv struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errEnv)
		if errEnv.Error != "" {
			return "", "", errors.New(errEnv.Error)
		}
		return "", "", errors.New("não foi possível iniciar a atualização agora; tente novamente em instantes")
	}

	var env struct {
		Vector string `json:"v"`
	}
	if json.NewDecoder(resp.Body).Decode(&env) != nil || env.Vector == "" {
		return "", "", errors.New("resposta inválida ao iniciar atualização")
	}
	var out struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	_ = c.cipher.openJSON(env.Vector, &out)
	if out.Status == "" {
		out.Status = "queued"
	}
	if out.Message == "" {
		out.Message = "Atualização iniciada! Os serviços serão reiniciados automaticamente em alguns instantes."
	}

	systemUpdateCache.mu.Lock()
	systemUpdateCache.status.UpdateStatus = out.Status
	systemUpdateCache.status.UpdateMessage = out.Message
	systemUpdateCache.fetchedAt = time.Now()
	systemUpdateCache.mu.Unlock()

	return out.Status, out.Message, nil
}
