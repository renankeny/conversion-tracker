package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/uazapi"
)

func (h *AdminWhatsAppInstances) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accID := TenantFromContext(ctx).EffectiveAccountID(0)
	instanceID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	inst, err := h.repo.GetInstance(ctx, accID, instanceID)
	if err != nil || inst == nil {
		writeError(w, http.StatusNotFound, "instância não encontrada")
		return
	}

	var in instanceUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}

	if in.PixelID > 0 && in.PixelID != inst.PixelID {
		pixel, perr := h.pixels.FindByID(ctx, in.PixelID)
		if perr != nil || pixel == nil || pixel.AccountID != accID {
			writeError(w, http.StatusBadRequest, "Pixel inválido")
			return
		}
		inst.PixelID = pixel.ID
		inst.PixelName = pixel.Name
	}
	if strings.TrimSpace(in.Name) != "" {
		inst.Name = strings.TrimSpace(in.Name)
	}
	if strings.TrimSpace(in.Currency) != "" {
		inst.Currency = domain.NormalizeWhatsAppCurrency(in.Currency)
	}
	if inst.IsExternalUazapi() {
		if in.Provider == domain.WhatsAppProviderAPIPremium || in.Provider == domain.WhatsAppProviderUazapi {
			inst.Provider = in.Provider
		}
		inst.ExternalAPIURL = strings.TrimRight(strings.TrimSpace(in.ExternalAPIURL), "/")
		if strings.TrimSpace(in.InstanceToken) != "" {
			inst.InstanceToken = strings.TrimSpace(in.InstanceToken)
		}
	}
	inst.PurchaseTrigger = strings.TrimSpace(in.PurchaseTrigger)
	inst.LeadTrigger = strings.TrimSpace(in.LeadTrigger)
	inst.CustomEventName = strings.TrimSpace(in.CustomEventName)
	inst.CustomEventTrigger = strings.TrimSpace(in.CustomEventTrigger)
	if in.AutoCreateCRMLead != nil {
		inst.AutoCreateCRMLead = *in.AutoCreateCRMLead
	}
	inst.DefaultCRMStageID = in.DefaultCRMStageID
	if in.Enabled != nil {
		inst.Enabled = *in.Enabled
	}

	if err := h.repo.UpdateInstance(ctx, inst); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao atualizar instância")
		return
	}

	if inst.IsExternalUazapi() && inst.ExternalAPIURL != "" && inst.InstanceToken != "" {
		webhookURL := buildProviderWebhookURL(h.publicAPIURL, inst.Provider, inst.WebhookSecret)
		if werr := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken).ConfigureWebhook(ctx, webhookURL, inst.WebhookSecret); werr != nil {
			slog.Warn("uazapi webhook update warning", "instance_id", inst.ID, "err", werr)
		}
		_ = syncInstanceLiveStatus(ctx, h.repo, inst, h.publicAPIURL)
	}

	h.decorateInstance(inst)
	writeJSON(w, http.StatusOK, inst)
}

func syncInstanceLiveStatus(ctx context.Context, r *repo.WhatsAppRepo, inst *domain.WhatsAppInstance, publicAPIURL string) error {
	if inst.IsExternalUazapi() {
		if inst.ExternalAPIURL == "" || inst.InstanceToken == "" {
			return nil
		}
		uazCli := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken)
		if publicAPIURL != "" && inst.WebhookSecret != "" {
			whURL := buildProviderWebhookURL(publicAPIURL, inst.Provider, inst.WebhookSecret)
			_ = uazCli.ConfigureWebhook(ctx, whURL, inst.WebhookSecret)
		}
		st, err := uazCli.GetStatus(ctx)
		if err != nil {
			return err
		}
		if st.Connected {
			ownerBare := queue.NormalizeBareJID(st.OwnerJID)
			if ownerBare == "" {
				ownerBare = inst.OwnerJID
			}
			profName := st.ProfileName
			if profName == "" {
				profName = inst.ProfileName
			}
			profPic := st.ProfilePicURL
			if profPic == "" {
				profPic = inst.ProfilePicURL
			}
			_ = r.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, profPic)
			inst.ConnectionStatus = "OPEN"
			inst.OwnerJID = ownerBare
			inst.ProfileName = profName
			inst.ProfilePicURL = profPic
			return nil
		}
		status := "CLOSED"
		if st.Status == "connecting" || st.Qrcode != "" {
			status = "CONNECTING"
		}
		_ = r.ResetInstanceSession(ctx, inst.ID, status)
		inst.ConnectionStatus = status
		inst.OwnerJID = ""
		inst.ProfileName = ""
		inst.ProfilePicURL = ""
		return nil
	}

	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	st, err := cli.GetStatus(ctx, inst.InstanceToken)
	if err != nil {
		return err
	}

	target := inst.EvolutionUUID
	if target == "" {
		target = inst.InstanceName
	}
	info, _ := cli.GetInfo(ctx, target)

	isLoggedIn := st.LoggedIn || (info != nil && info.Connected)
	isConnected := st.Connected && isLoggedIn

	if isConnected {
		ownerBare := queue.NormalizeBareJID(st.MyJID)
		if ownerBare == "" && info != nil && info.JID != "" {
			ownerBare = queue.NormalizeBareJID(info.JID)
		}
		if ownerBare == "" {
			ownerBare = inst.OwnerJID
		}
		profName := strings.TrimSpace(st.Name)
		if info != nil && strings.TrimSpace(info.ProfileName) != "" {
			profName = strings.TrimSpace(info.ProfileName)
		}
		if profName == "" {
			profName = inst.ProfileName
		}
		profPic := ""
		if info != nil && strings.TrimSpace(info.ProfilePicURL) != "" {
			profPic = strings.TrimSpace(info.ProfilePicURL)
		}
		if profPic == "" && ownerBare != "" {
			profPic = cli.GetUserAvatar(ctx, inst.InstanceToken, ownerBare)
		}
		if profPic == "" {
			profPic = inst.ProfilePicURL
		}

		_ = r.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, profPic)
		inst.ConnectionStatus = "OPEN"
		inst.OwnerJID = ownerBare
		inst.ProfileName = profName
		inst.ProfilePicURL = profPic
		return nil
	}

	if isLoggedIn && !st.Connected {
		_ = r.UpdateInstanceStatus(ctx, inst.ID, "DISCONNECTED")
		inst.ConnectionStatus = "DISCONNECTED"
		return nil
	}

	status := "CLOSED"
	if st.Connected || (info != nil && info.Qrcode != "") {
		status = "CONNECTING"
	}
	_ = r.ResetInstanceSession(ctx, inst.ID, status)
	inst.ConnectionStatus = status
	inst.OwnerJID = ""
	inst.ProfileName = ""
	inst.ProfilePicURL = ""
	return nil
}
