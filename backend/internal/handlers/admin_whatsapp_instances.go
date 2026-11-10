package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminWhatsAppInstances struct {
	repo         *repo.WhatsAppRepo
	pixels       *repo.PixelRepo
	publicAPIURL string
}

func NewAdminWhatsAppInstances(repo *repo.WhatsAppRepo, pixels *repo.PixelRepo, publicAPIURL string) *AdminWhatsAppInstances {
	return &AdminWhatsAppInstances{
		repo:         repo,
		pixels:       pixels,
		publicAPIURL: strings.TrimRight(strings.TrimSpace(publicAPIURL), "/"),
	}
}

func randomHexToken(byteLen int) string {
	b := make([]byte, byteLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

var rxNonSlug = regexp.MustCompile(`[^a-z0-9-]+`)

func slugifyInstanceName(name string, accID int64) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = rxNonSlug.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "whatsapp"
	}
	return s + "-acc" + strconv.FormatInt(accID, 10) + "-" + strconv.FormatInt(time.Now().Unix()%100000, 10)
}

func (h *AdminWhatsAppInstances) buildWebhookURL(secret string) string {
	return h.publicAPIURL + "/webhooks/evolution/" + secret
}

func (h *AdminWhatsAppInstances) List(w http.ResponseWriter, r *http.Request) {
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	instances, err := h.repo.ListInstances(r.Context(), accID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar instâncias: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": instances})
}

type instanceUpsertInput struct {
	Name               string `json:"name"`
	PixelID            int64  `json:"pixel_id"`
	PurchaseTrigger    string `json:"purchase_trigger"`
	LeadTrigger        string `json:"lead_trigger"`
	CustomEventName    string `json:"custom_event_name"`
	CustomEventTrigger string `json:"custom_event_trigger"`
	AutoCreateCRMLead  *bool  `json:"auto_create_crm_lead"`
	DefaultCRMStageID  *int64 `json:"default_crm_stage_id"`
	Enabled            *bool  `json:"enabled"`
}

func (h *AdminWhatsAppInstances) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accID := TenantFromContext(ctx).EffectiveAccountID(0)

	var in instanceUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, http.StatusBadRequest, "nome da instância é obrigatório")
		return
	}

	pixel, err := h.pixels.FindByID(ctx, in.PixelID)
	if err != nil || pixel == nil || pixel.AccountID != accID {
		writeError(w, http.StatusBadRequest, "selecione um Pixel válido desta conta")
		return
	}

	count, err := h.repo.CountInstances(ctx, accID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if count >= 5 {
		writeError(w, http.StatusBadRequest, "limite máximo de 5 conexões de WhatsApp por conta atingido")
		return
	}

	server, err := h.repo.GetAccountServer(ctx, accID)
	if err != nil || server == nil {
		writeError(w, http.StatusBadRequest, "configure a URL e API Key da Evolution Go antes de criar uma instância")
		return
	}

	slug := slugifyInstanceName(in.Name, accID)
	instToken := "tok_" + randomHexToken(20)
	webhookSecret := randomHexToken(20)
	shareToken := randomHexToken(12)

	cli := evolution.NewClient(server.URL, server.APIKey)
	createdResp, err := cli.CreateInstance(ctx, slug, instToken)
	if err != nil {
		writeError(w, http.StatusBadGateway, "erro ao criar instância na Evolution Go: "+err.Error())
		return
	}

	evoName := createdResp.Data.Name
	if evoName == "" {
		evoName = slug
	}
	evoUUID := createdResp.Data.ID

	// Passo 2 obrigatório da Evolution Go: configurar webhookUrl + eventos via /instance/connect
	webhookURL := h.buildWebhookURL(webhookSecret)
	if _, connErr := cli.ConnectInstance(ctx, instToken, webhookURL); connErr != nil {
		slog.Warn("initial evolution connect warning", "instance", evoName, "err", connErr)
	}

	autoCRM := true
	if in.AutoCreateCRMLead != nil {
		autoCRM = *in.AutoCreateCRMLead
	}

	inst := &domain.WhatsAppInstance{
		AccountID:          accID,
		PixelID:            pixel.ID,
		PixelName:          pixel.Name,
		ServerID:           server.ID,
		Name:               strings.TrimSpace(in.Name),
		InstanceName:       evoName,
		EvolutionUUID:      evoUUID,
		InstanceToken:      instToken,
		WebhookSecret:      webhookSecret,
		ShareToken:         &shareToken,
		ConnectionStatus:   "CONNECTING",
		PurchaseTrigger:    strings.TrimSpace(in.PurchaseTrigger),
		LeadTrigger:        strings.TrimSpace(in.LeadTrigger),
		CustomEventName:    strings.TrimSpace(in.CustomEventName),
		CustomEventTrigger: strings.TrimSpace(in.CustomEventTrigger),
		AutoCreateCRMLead:  autoCRM,
		DefaultCRMStageID:  in.DefaultCRMStageID,
		Enabled:            true,
	}

	if err := h.repo.CreateInstance(ctx, inst); err != nil {
		_ = cli.DeleteInstance(ctx, evoUUID, evoName)
		writeError(w, http.StatusInternalServerError, "erro ao salvar instância: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, inst)
}

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
	writeJSON(w, http.StatusOK, inst)
}
