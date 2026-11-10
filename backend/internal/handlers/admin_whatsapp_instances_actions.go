package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
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
	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	webhookURL := h.buildWebhookURL(inst.WebhookSecret)
	if _, err := cli.ConnectInstance(r.Context(), inst.InstanceToken, webhookURL); err != nil {
		writeError(w, http.StatusBadGateway, "erro ao iniciar conexão na Evolution Go: "+err.Error())
		return
	}
	_ = h.repo.UpdateInstanceStatus(r.Context(), inst.ID, "CONNECTING")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "status": "CONNECTING"})
}

func (h *AdminWhatsAppInstances) QrCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
		return
	}

	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	webhookURL := h.buildWebhookURL(inst.WebhookSecret)
	_, _ = cli.ConnectInstance(ctx, inst.InstanceToken, webhookURL)

	qrData, err := cli.GetQrCode(ctx, inst.InstanceToken)
	if err != nil {
		if st, stErr := cli.GetStatus(ctx, inst.InstanceToken); stErr == nil && st != nil && (st.Connected || st.LoggedIn) {
			ownerBare := queue.NormalizeBareJID(st.MyJID)
			_ = h.repo.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, st.Name, "")
			writeJSON(w, http.StatusOK, map[string]any{"connected": true, "status": "OPEN", "owner_jid": ownerBare})
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "already logged in") {
			_ = h.repo.UpdateInstanceStatus(ctx, inst.ID, "OPEN")
			writeJSON(w, http.StatusOK, map[string]any{"connected": true, "status": "OPEN"})
			return
		}
		writeError(w, http.StatusBadGateway, "erro ao gerar QR Code: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"connected": false,
		"qrcode":    qrData.Qrcode,
		"code":      qrData.Code,
	})
}

func (h *AdminWhatsAppInstances) Pair(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
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
	webhookURL := h.buildWebhookURL(inst.WebhookSecret)
	_, _ = cli.ConnectInstance(ctx, inst.InstanceToken, webhookURL)

	code, err := cli.PairPhone(ctx, inst.InstanceToken, in.Phone)
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
	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	st, err := cli.GetStatus(ctx, inst.InstanceToken)
	if err != nil {
		writeError(w, http.StatusBadGateway, "erro ao consultar status: "+err.Error())
		return
	}

	status := "CLOSED"
	if st.Connected || st.LoggedIn {
		status = "OPEN"
	}
	ownerBare := queue.NormalizeBareJID(st.MyJID)
	profName := st.Name
	profPic := ""
	target := inst.EvolutionUUID
	if target == "" {
		target = inst.InstanceName
	}
	if info, ierr := cli.GetInfo(ctx, target); ierr == nil && info != nil {
		if info.ProfileName != "" {
			profName = info.ProfileName
		}
		profPic = info.ProfilePicURL
	}
	_ = h.repo.UpdateInstanceOwnerAndProfile(ctx, inst.ID, status, ownerBare, profName, profPic)
	updated, _ := h.repo.GetInstance(ctx, inst.AccountID, inst.ID)
	writeJSON(w, http.StatusOK, updated)
}

func (h *AdminWhatsAppInstances) Disconnect(w http.ResponseWriter, r *http.Request) {
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
		return
	}
	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	_ = cli.LogoutInstance(r.Context(), inst.InstanceToken)
	_ = h.repo.UpdateInstanceStatus(r.Context(), inst.ID, "CLOSED")
	writeJSON(w, http.StatusOK, map[string]any{"disconnected": true})
}

func (h *AdminWhatsAppInstances) Delete(w http.ResponseWriter, r *http.Request) {
	inst, ok := h.loadOwnedInstance(w, r)
	if !ok {
		return
	}
	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	_ = cli.DeleteInstance(r.Context(), inst.EvolutionUUID, inst.InstanceName)

	if err := h.repo.DeleteInstance(r.Context(), inst.AccountID, inst.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir instância: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

