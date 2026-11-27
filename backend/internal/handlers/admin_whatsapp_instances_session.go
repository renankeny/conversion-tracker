package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/uazapi"
)

func (h *AdminWhatsAppInstances) loadOwnedInstance(w http.ResponseWriter, r *http.Request) (*domain.WhatsAppInstance, bool) {
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	instanceID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return nil, false
	}
	inst, err := h.repo.GetInstance(r.Context(), accID, instanceID)
	if err != nil || inst == nil {
		writeError(w, http.StatusNotFound, "instância não encontrada")
		return nil, false
	}
	return inst, true
}

func (h *AdminWhatsAppInstances) Connect(w http.ResponseWriter, r *http.Request) {
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	if inst.IsWAHA() {
		whURL := buildProviderWebhookURL(h.publicAPIURL, domain.WhatsAppProviderWAHA, inst.WebhookSecret)
		if _, err := wahaClientForInstance(inst).EnsureSessionRunning(ctx, inst.InstanceName, whURL); err != nil {
			writeError(w, http.StatusBadGateway, "erro ao iniciar sessão na WAHA API: "+err.Error())
			return
		}
		_ = h.repo.UpdateInstanceStatus(ctx, inst.ID, "CONNECTING")
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "status": "CONNECTING"})
		return
	}
	if inst.IsExternalUazapi() {
		if inst.ExternalAPIURL != "" && inst.InstanceToken != "" {
			uazCli := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken)
			whURL := buildProviderWebhookURL(h.publicAPIURL, inst.Provider, inst.WebhookSecret)
			_ = uazCli.ConfigureWebhook(ctx, whURL, inst.WebhookSecret)
			_, _ = uazCli.Connect(ctx)
		}
		_ = h.repo.UpdateInstanceStatus(ctx, inst.ID, "CONNECTING")
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "status": "CONNECTING"})
		return
	}

	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	if _, err := cli.ConnectInstance(ctx, inst.InstanceToken, h.buildWebhookURL(inst.WebhookSecret)); err != nil {
		writeError(w, http.StatusBadGateway, "erro ao iniciar conexão na Evolution Go: "+err.Error())
		return
	}
	_ = h.repo.UpdateInstanceStatus(ctx, inst.ID, "CONNECTING")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "status": "CONNECTING"})
}

func (h *AdminWhatsAppInstances) QrCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
		return
	}

	if inst.IsWAHA() {
		qr, connected, err := fetchWAHAQRCode(ctx, h.repo, inst, h.publicAPIURL)
		if err != nil {
			writeError(w, http.StatusBadGateway, "erro ao gerar QR Code WAHA: "+err.Error())
			return
		}
		if connected {
			writeJSON(w, http.StatusOK, map[string]any{
				"connected": true, "status": "OPEN",
				"owner_jid": inst.OwnerJID, "profile_name": inst.ProfileName, "profile_pic_url": inst.ProfilePicURL,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"connected": false, "qrcode": qr})
		return
	}

	if inst.IsExternalUazapi() {
		if inst.ExternalAPIURL == "" || inst.InstanceToken == "" {
			writeError(w, http.StatusBadRequest, "informe a URL da API e o Token da Instância ou utilize o link de Webhook")
			return
		}
		uazCli := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken)
		whURL := buildProviderWebhookURL(h.publicAPIURL, inst.Provider, inst.WebhookSecret)
		_ = uazCli.ConfigureWebhook(ctx, whURL, inst.WebhookSecret)
		if st, err := uazCli.GetStatus(ctx); err == nil && st != nil {
			if st.Connected {
				_ = syncInstanceLiveStatus(ctx, h.repo, inst, h.publicAPIURL)
				writeJSON(w, http.StatusOK, map[string]any{
					"connected": true, "status": "OPEN",
					"owner_jid": inst.OwnerJID, "profile_name": inst.ProfileName, "profile_pic_url": inst.ProfilePicURL,
				})
				return
			}
			if st.Qrcode != "" {
				writeJSON(w, http.StatusOK, map[string]any{"connected": false, "qrcode": st.Qrcode})
				return
			}
		}
		qr, err := uazCli.Connect(ctx)
		if err != nil {
			writeError(w, http.StatusBadGateway, "erro ao gerar QR Code: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"connected": false, "qrcode": qr})
		return
	}

	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	_, _ = cli.ConnectInstance(ctx, inst.InstanceToken, h.buildWebhookURL(inst.WebhookSecret))
	qrData, err := cli.GetQrCode(ctx, inst.InstanceToken)
	if err != nil {
		if syncErr := syncInstanceLiveStatus(ctx, h.repo, inst, h.publicAPIURL); syncErr == nil && inst.ConnectionStatus == "OPEN" {
			writeJSON(w, http.StatusOK, map[string]any{
				"connected": true, "status": "OPEN",
				"owner_jid": inst.OwnerJID, "profile_name": inst.ProfileName, "profile_pic_url": inst.ProfilePicURL,
			})
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "already logged in") {
			_ = syncInstanceLiveStatus(ctx, h.repo, inst, h.publicAPIURL)
			writeJSON(w, http.StatusOK, map[string]any{"connected": true, "status": "OPEN"})
			return
		}
		writeError(w, http.StatusBadGateway, "erro ao gerar QR Code: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connected": false, "qrcode": qrData.Qrcode, "code": qrData.Code})
}

func (h *AdminWhatsAppInstances) Pair(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
		return
	}
	if inst.IsExternalUazapi() {
		writeError(w, http.StatusBadRequest, "pareamento por código telefônico é suportado apenas em conexões diretas (Evolution Go ou WAHA)")
		return
	}
	var in struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Phone) == "" {
		writeError(w, http.StatusBadRequest, "informe o número de telefone com DDI (ex: 5511999999999)")
		return
	}

	var code string
	var err error
	if inst.IsWAHA() {
		code, err = pairWAHAPhone(ctx, inst, h.publicAPIURL, in.Phone)
	} else {
		cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
		_, _ = cli.ConnectInstance(ctx, inst.InstanceToken, h.buildWebhookURL(inst.WebhookSecret))
		code, err = cli.PairPhone(ctx, inst.InstanceToken, in.Phone)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "erro ao gerar código de pareamento: "+err.Error())
		return
	}
	_ = h.repo.UpdateInstanceStatus(ctx, inst.ID, "CONNECTING")
	writeJSON(w, http.StatusOK, map[string]any{"pairing_code": code})
}

func (h *AdminWhatsAppInstances) RefreshStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
		return
	}
	wasOpen := isWhatsAppInstanceConnected(inst.ConnectionStatus)
	if err := syncInstanceLiveStatus(ctx, h.repo, inst, h.publicAPIURL); err != nil {
		writeError(w, http.StatusBadGateway, "erro ao consultar status: "+err.Error())
		return
	}
	nowOpen := isWhatsAppInstanceConnected(inst.ConnectionStatus)
	if wasOpen != nowOpen {
		emitWhatsAppStatusNotification(h.notify, inst, nowOpen)
	}
	updated, _ := h.repo.GetInstance(ctx, inst.AccountID, inst.ID)
	if updated == nil {
		updated = inst
	}
	h.decorateInstance(updated)
	writeJSON(w, http.StatusOK, updated)
}

func (h *AdminWhatsAppInstances) Disconnect(w http.ResponseWriter, r *http.Request) {
	if isEmployeeRole(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem desconectar instâncias")
		return
	}
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
		return
	}
	wasOpen := isWhatsAppInstanceConnected(inst.ConnectionStatus)
	if inst.IsWAHA() {
		if inst.ServerURL != "" {
			_ = wahaClientForInstance(inst).LogoutSession(r.Context(), inst.InstanceName)
		}
	} else if inst.IsExternalUazapi() {
		if inst.ExternalAPIURL != "" && inst.InstanceToken != "" {
			_ = uazapi.New(inst.ExternalAPIURL, inst.InstanceToken).Disconnect(r.Context())
		}
	} else {
		_ = evolution.NewClient(inst.ServerURL, inst.ServerAPIKey).LogoutInstance(r.Context(), inst.InstanceToken)
	}
	_ = h.repo.ResetInstanceSession(r.Context(), inst.ID, "CLOSED")
	if wasOpen {
		emitWhatsAppStatusNotification(h.notify, inst, false)
	}
	writeJSON(w, http.StatusOK, map[string]any{"disconnected": true})
}

func (h *AdminWhatsAppInstances) Delete(w http.ResponseWriter, r *http.Request) {
	if isEmployeeRole(r.Context()) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem excluir instâncias")
		return
	}
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
		return
	}
	if inst.IsWAHA() && inst.ServerURL != "" {
		_ = wahaClientForInstance(inst).DeleteSession(r.Context(), inst.InstanceName)
	} else if inst.IsEvolutionGo() && inst.ServerURL != "" {
		_ = evolution.NewClient(inst.ServerURL, inst.ServerAPIKey).DeleteInstance(r.Context(), inst.EvolutionUUID, inst.InstanceName)
	}
	if err := h.repo.DeleteInstance(r.Context(), inst.AccountID, inst.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir instância: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}
