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

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/uazapi"
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

func buildProviderWebhookURL(publicAPIURL, provider, secret string) string {
	if secret == "" {
		return ""
	}
	switch domain.NormalizeWhatsAppProvider(provider) {
	case domain.WhatsAppProviderAPIPremium:
		return publicAPIURL + "/webhooks/api-premium/" + secret
	case domain.WhatsAppProviderUazapi:
		return publicAPIURL + "/webhooks/uazapi/" + secret
	default:
		return publicAPIURL + "/webhooks/evolution/" + secret
	}
}

func (h *AdminWhatsAppInstances) buildWebhookURL(secret string) string {
	return buildProviderWebhookURL(h.publicAPIURL, domain.WhatsAppProviderEvolutionGo, secret)
}

func (h *AdminWhatsAppInstances) decorateInstance(inst *domain.WhatsAppInstance) {
	if inst == nil {
		return
	}
	inst.Provider = domain.NormalizeWhatsAppProvider(inst.Provider)
	inst.WebhookURL = buildProviderWebhookURL(h.publicAPIURL, inst.Provider, inst.WebhookSecret)
}

func (h *AdminWhatsAppInstances) List(w http.ResponseWriter, r *http.Request) {
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	instances, err := h.repo.ListInstances(r.Context(), accID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar instâncias: "+err.Error())
		return
	}
	for i := range instances {
		if instances[i].ShareToken == nil || strings.TrimSpace(*instances[i].ShareToken) == "" {
			tok := randomHexToken(12)
			_ = h.repo.EnsureInstanceShareToken(r.Context(), instances[i].ID, tok)
			instances[i].ShareToken = &tok
		}
		if instances[i].ConnectionStatus == "CONNECTING" || (instances[i].ConnectionStatus == "OPEN" && (instances[i].OwnerJID == "" || instances[i].ProfilePicURL == "")) {
			_ = syncInstanceLiveStatus(r.Context(), h.repo, &instances[i], h.publicAPIURL)
		}
		h.decorateInstance(&instances[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": instances})
}

type instanceUpsertInput struct {
	Name               string `json:"name"`
	PixelID            int64  `json:"pixel_id"`
	Provider           string `json:"provider"`
	ExternalAPIURL     string `json:"external_api_url"`
	Currency           string `json:"currency"`
	InstanceToken      string `json:"instance_token"`
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

	provider := domain.NormalizeWhatsAppProvider(in.Provider)
	slug := slugifyInstanceName(in.Name, accID)
	webhookSecret := randomHexToken(20)
	shareToken := randomHexToken(12)
	webhookURL := buildProviderWebhookURL(h.publicAPIURL, provider, webhookSecret)

	var srvID int64
	var evoName, evoUUID, instToken string

	if provider == domain.WhatsAppProviderEvolutionGo {
		server, serr := h.repo.GetAccountServer(ctx, accID)
		if serr != nil || server == nil {
			writeError(w, http.StatusBadRequest, "configure a URL e API Key da Evolution Go antes de criar uma conexão via QR Code")
			return
		}
		srvID = server.ID
		instToken = "tok_" + randomHexToken(20)
		cli := evolution.NewClient(server.URL, server.APIKey)
		createdResp, cerr := cli.CreateInstance(ctx, slug, instToken)
		if cerr != nil {
			writeError(w, http.StatusBadGateway, "erro ao criar instância na Evolution Go: "+cerr.Error())
			return
		}
		evoName = createdResp.Data.Name
		if evoName == "" {
			evoName = slug
		}
		evoUUID = createdResp.Data.ID
		if _, connErr := cli.ConnectInstance(ctx, instToken, webhookURL); connErr != nil {
			slog.Warn("initial evolution connect warning", "instance", evoName, "err", connErr)
		}
	} else {
		evoName = slug
		instToken = strings.TrimSpace(in.InstanceToken)
	}

	autoCRM := true
	if in.AutoCreateCRMLead != nil {
		autoCRM = *in.AutoCreateCRMLead
	}

	inst := &domain.WhatsAppInstance{
		AccountID:          accID,
		PixelID:            pixel.ID,
		PixelName:          pixel.Name,
		ServerID:           srvID,
		Provider:           provider,
		ExternalAPIURL:     strings.TrimRight(strings.TrimSpace(in.ExternalAPIURL), "/"),
		Currency:           domain.NormalizeWhatsAppCurrency(in.Currency),
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
		if provider == domain.WhatsAppProviderEvolutionGo && srvID > 0 {
			if srv, _ := h.repo.GetAccountServer(ctx, accID); srv != nil {
				_ = evolution.NewClient(srv.URL, srv.APIKey).DeleteInstance(ctx, evoUUID, evoName)
			}
		}
		writeError(w, http.StatusInternalServerError, "erro ao salvar instância: "+err.Error())
		return
	}

	if inst.IsExternalUazapi() && inst.ExternalAPIURL != "" && inst.InstanceToken != "" {
		uazCli := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken)
		if werr := uazCli.ConfigureWebhook(ctx, webhookURL, webhookSecret); werr != nil {
			slog.Warn("initial uazapi webhook config warning", "instance_id", inst.ID, "err", werr)
		}
		_ = syncInstanceLiveStatus(ctx, h.repo, inst, h.publicAPIURL)
	}

	h.decorateInstance(inst)
	writeJSON(w, http.StatusCreated, inst)
}
