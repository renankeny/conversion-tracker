package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/uazapi"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func (h *AdminWhatsAppInstances) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if isEmployeeRole(ctx) {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem editar conexões WhatsApp")
		return
	}
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
		if extURL := strings.TrimRight(strings.TrimSpace(in.ExternalAPIURL), "/"); extURL != "" {
			cleanExt, uerr := util.ValidateServiceEndpointURL(extURL)
			if uerr != nil {
				writeError(w, http.StatusBadRequest, uerr.Error())
				return
			}
			if cleanExt != inst.ExternalAPIURL && strings.TrimSpace(in.InstanceToken) == "" && inst.InstanceToken != "" {
				writeError(w, http.StatusBadRequest, "informe o Token da Instância ao alterar a URL da API")
				return
			}
			inst.ExternalAPIURL = cleanExt
		}
		if strings.TrimSpace(in.InstanceToken) != "" {
			inst.InstanceToken = strings.TrimSpace(in.InstanceToken)
		}
	}
	applyInstanceTriggersInput(inst, in)
	if in.AutoCreateCRMLead != nil {
		inst.AutoCreateCRMLead = *in.AutoCreateCRMLead
	}
	inst.DefaultCRMStageID = in.DefaultCRMStageID
	if in.CtwaMinInboundMessages != nil {
		inst.CtwaMinInboundMessages = domain.NormalizeCtwaMinMessages(*in.CtwaMinInboundMessages)
	}
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

func applyInstanceTriggersInput(inst *domain.WhatsAppInstance, in instanceUpsertInput) {
	if in.AverageTicket != nil {
		if *in.AverageTicket > 0 {
			inst.AverageTicket = *in.AverageTicket
		} else {
			inst.AverageTicket = 0
		}
	}
	if in.EventTriggers != nil {
		inst.EventTriggers = domain.NormalizeWhatsAppEventTriggers(*in.EventTriggers)
		domain.SyncLegacyTriggerColumns(inst)
		return
	}
	inst.PurchaseTrigger = strings.TrimSpace(in.PurchaseTrigger)
	inst.LeadTrigger = strings.TrimSpace(in.LeadTrigger)
	inst.CustomEventName = strings.TrimSpace(in.CustomEventName)
	inst.CustomEventTrigger = strings.TrimSpace(in.CustomEventTrigger)
	inst.EventTriggers = domain.LegacyTriggersToEventTriggers(
		inst.PurchaseTrigger, inst.LeadTrigger, inst.CustomEventName, inst.CustomEventTrigger,
	)
}
