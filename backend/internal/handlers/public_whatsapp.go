package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/uazapi"
)

type PublicWhatsApp struct {
	repo         *repo.WhatsAppRepo
	publicAPIURL string
}

func NewPublicWhatsApp(repo *repo.WhatsAppRepo, publicAPIURL string) *PublicWhatsApp {
	return &PublicWhatsApp{
		repo:         repo,
		publicAPIURL: strings.TrimRight(strings.TrimSpace(publicAPIURL), "/"),
	}
}

func (h *PublicWhatsApp) ConnectStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if token == "" {
		writeError(w, http.StatusBadRequest, "token inválido")
		return
	}
	inst, err := h.repo.GetInstanceByShareToken(ctx, token)
	if err != nil || inst == nil {
		writeError(w, http.StatusNotFound, "instância não encontrada")
		return
	}

	_ = syncInstanceLiveStatus(ctx, h.repo, inst, h.publicAPIURL)

	qrBase64 := ""
	if inst.ConnectionStatus != "OPEN" {
		if inst.IsExternalUazapi() {
			if inst.ExternalAPIURL != "" && inst.InstanceToken != "" {
				uazCli := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken)
				if st, serr := uazCli.GetStatus(ctx); serr == nil && st != nil && st.Qrcode != "" {
					qrBase64 = st.Qrcode
				} else if qr, cerr := uazCli.Connect(ctx); cerr == nil {
					qrBase64 = qr
				}
			}
		} else {
			cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
			if qrData, qerr := cli.GetQrCode(ctx, inst.InstanceToken); qerr == nil && qrData != nil && qrData.Qrcode != "" {
				qrBase64 = qrData.Qrcode
			} else {
				webhookURL := h.publicAPIURL + "/webhooks/evolution/" + inst.WebhookSecret
				_, _ = cli.ConnectInstance(ctx, inst.InstanceToken, webhookURL)
				if qrData2, qerr2 := cli.GetQrCode(ctx, inst.InstanceToken); qerr2 == nil && qrData2 != nil {
					qrBase64 = qrData2.Qrcode
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":         true,
		"instance_name":   inst.Name,
		"profile_name":    inst.ProfileName,
		"profile_pic_url": inst.ProfilePicURL,
		"owner_jid":       inst.OwnerJID,
		"status":          inst.ConnectionStatus,
		"qrcode":          qrBase64,
	})
}

func (h *PublicWhatsApp) InitConnect(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if token == "" {
		writeError(w, http.StatusBadRequest, "token inválido")
		return
	}
	inst, err := h.repo.GetInstanceByShareToken(r.Context(), token)
	if err != nil || inst == nil {
		writeError(w, http.StatusNotFound, "instância não encontrada")
		return
	}

	if inst.IsExternalUazapi() {
		if inst.ExternalAPIURL != "" && inst.InstanceToken != "" {
			uazCli := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken)
			whURL := buildProviderWebhookURL(h.publicAPIURL, inst.Provider, inst.WebhookSecret)
			_ = uazCli.ConfigureWebhook(r.Context(), whURL, inst.WebhookSecret)
			_, _ = uazCli.Connect(r.Context())
		}
		_ = h.repo.UpdateInstanceStatus(r.Context(), inst.ID, "CONNECTING")
		writeJSON(w, http.StatusOK, map[string]any{"success": true})
		return
	}

	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	webhookURL := h.publicAPIURL + "/webhooks/evolution/" + inst.WebhookSecret
	if _, err := cli.ConnectInstance(r.Context(), inst.InstanceToken, webhookURL); err != nil {
		writeError(w, http.StatusBadGateway, "erro ao iniciar conexão: "+err.Error())
		return
	}
	_ = h.repo.UpdateInstanceStatus(r.Context(), inst.ID, "CONNECTING")
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *PublicWhatsApp) PairPhone(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if token == "" {
		writeError(w, http.StatusBadRequest, "token inválido")
		return
	}
	inst, err := h.repo.GetInstanceByShareToken(ctx, token)
	if err != nil || inst == nil {
		writeError(w, http.StatusNotFound, "instância não encontrada")
		return
	}

	var in struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Phone) == "" {
		writeError(w, http.StatusBadRequest, "informe o número de telefone com DDI (ex: 5511999999999)")
		return
	}

	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	webhookURL := h.publicAPIURL + "/webhooks/evolution/" + inst.WebhookSecret
	_, _ = cli.ConnectInstance(ctx, inst.InstanceToken, webhookURL)

	code, err := cli.PairPhone(ctx, inst.InstanceToken, in.Phone)
	if err != nil {
		writeError(w, http.StatusBadGateway, "erro ao gerar código de pareamento: "+err.Error())
		return
	}
	_ = h.repo.UpdateInstanceStatus(ctx, inst.ID, "CONNECTING")
	writeJSON(w, http.StatusOK, map[string]any{"pairing_code": code})
}
