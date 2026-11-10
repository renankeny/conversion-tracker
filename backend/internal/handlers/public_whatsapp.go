package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
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

	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	qrBase64 := ""
	if inst.ConnectionStatus != "OPEN" {
		if qrData, qerr := cli.GetQrCode(r.Context(), inst.InstanceToken); qerr == nil && qrData != nil {
			qrBase64 = qrData.Qrcode
		} else if st, stErr := cli.GetStatus(r.Context(), inst.InstanceToken); stErr == nil && st != nil && (st.Connected || st.LoggedIn) {
			ownerBare := queue.NormalizeBareJID(st.MyJID)
			_ = h.repo.UpdateInstanceOwnerAndProfile(r.Context(), inst.ID, "OPEN", ownerBare, st.Name, "")
			inst.ConnectionStatus = "OPEN"
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"instance_name": inst.Name,
		"status":        inst.ConnectionStatus,
		"qrcode":        qrBase64,
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

	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	webhookURL := h.publicAPIURL + "/webhooks/evolution/" + inst.WebhookSecret
	if _, err := cli.ConnectInstance(r.Context(), inst.InstanceToken, webhookURL); err != nil {
		writeError(w, http.StatusBadGateway, "erro ao iniciar conexão: "+err.Error())
		return
	}
	_ = h.repo.UpdateInstanceStatus(r.Context(), inst.ID, "CONNECTING")
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
