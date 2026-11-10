package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oliveagle/jsonpath"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type CustomWebhookHandler struct {
	repo       *repo.CustomWebhookRepo
	base       *Webhooks
	webhookLog *repo.WebhookLogRepo
}

func NewCustomWebhookHandler(r *repo.CustomWebhookRepo, base *Webhooks, logRepo *repo.WebhookLogRepo) *CustomWebhookHandler {
	return &CustomWebhookHandler{repo: r, base: base, webhookLog: logRepo}
}

type CustomMappingConfig struct {
	FirstNamePath     string            `json:"first_name"`
	LastNamePath      string            `json:"last_name"`
	EmailPath         string            `json:"email"`
	PhonePath         string            `json:"phone"`
	TransactionIDPath string            `json:"transaction_id"`
	AmountPath        string            `json:"amount"`
	StatusPath        string            `json:"status"`
	UTMSourcePath     string            `json:"utm_source"`
	UTMMediumPath     string            `json:"utm_medium"`
	UTMCampaignPath   string            `json:"utm_campaign"`
	PixelPath         string            `json:"pixel"`
	StatusMap         map[string]string `json:"status_map"`
}

func (h *CustomWebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id obrigatorio")
		return
	}

	template, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "integração não encontrada")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "erro lendo body")
		return
	}

	var jsonBody any
	if err := json.Unmarshal(bodyBytes, &jsonBody); err != nil {
		writeError(w, http.StatusBadRequest, "body não é json válido")
		return
	}

	var mapping CustomMappingConfig
	if err := json.Unmarshal(template.MappingConfig, &mapping); err != nil {
		writeError(w, http.StatusInternalServerError, "configuração de mapeamento inválida")
		return
	}

	lookupRaw := func(path string) any {
		if path == "" {
			return nil
		}
		val, err := jsonpath.JsonPathLookup(jsonBody, path)
		if err != nil {
			return nil
		}
		return val
	}
	extractString := func(path string) string {
		return anyString(lookupRaw(path))
	}

	firstName := extractString(mapping.FirstNamePath)
	lastName := extractString(mapping.LastNamePath)
	email := extractString(mapping.EmailPath)
	phone := util.NormalizePhone(extractString(mapping.PhonePath))
	txnID := extractString(mapping.TransactionIDPath)
	valFloat := util.ParseMoneyAny(lookupRaw(mapping.AmountPath))
	statusRaw := strings.TrimSpace(extractString(mapping.StatusPath))

	if firstName != "" && (firstName == lastName || lastName == "") {
		firstName, lastName = util.SplitFullName(firstName)
	}

	utmSource := extractString(mapping.UTMSourcePath)
	externalID := util.ExtractUserIDFromAny(utmSource)
	if externalID == "" {
		externalID = util.ExtractUserIDDeep(bodyBytes)
	}

	pixel := extractString(mapping.PixelPath)
	pixelQuery := util.FirstNonEmpty(r.URL.Query().Get("pixel"), r.URL.Query().Get("token"), pixel)

	var accountID int64
	if template.AccountID != nil {
		accountID = *template.AccountID
	}
	if accountID == 0 {
		accountID = ExtractAccountID(r)
	}

	mappedEvent := resolveCustomMappedEvent(mapping.StatusMap, statusRaw)
	if !isAllowedCustomWebhookEvent(mappedEvent) {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"ignored":  true,
			"status":   statusRaw,
		})
		return
	}

	rawInput := &domain.WebhookPurchaseInput{
		GatewayName: template.Name,
		EventName:   mappedEvent,
		ExternalID:  externalID,
		OrderID:     txnID,
		Value:       valFloat,
		Currency:    "BRL",
		FirstName:   firstName,
		LastName:    lastName,
		Email:       email,
		Phone:       phone,
		AccountID:   accountID,
		PixelQuery:  pixelQuery,
	}

	in := purchaseInput{
		GatewayName:     template.Name,
		EventName:       mappedEvent,
		ExternalID:      externalID,
		OrderID:         txnID,
		Value:           valFloat,
		Currency:        "BRL",
		FirstName:       firstName,
		LastName:        lastName,
		Email:           email,
		Phone:           phone,
		AccountID:       accountID,
		PixelQuery:      pixelQuery,
		RawPayload:      bodyBytes,
		CustomWebhookID: template.ID,
	}

	if !h.base.ValidatePlatformToken(r.Context(), r, accountID, "custom_webhook") {
		writeError(w, http.StatusUnauthorized, "invalid platform token")
		return
	}
	result, status := h.base.enqueueGenericPurchase(r.Context(), in, rawInput)
	writeJSON(w, status, result)
}

func resolveCustomMappedEvent(statusMap map[string]string, statusRaw string) string {
	if statusMap != nil {
		if ev, ok := statusMap[statusRaw]; ok && ev != "" {
			return strings.TrimSpace(ev)
		}
		lower := strings.ToLower(statusRaw)
		for k, ev := range statusMap {
			if strings.ToLower(strings.TrimSpace(k)) == lower && ev != "" {
				return strings.TrimSpace(ev)
			}
		}
	}
	if isAllowedCustomWebhookEvent(statusRaw) {
		return statusRaw
	}
	return ""
}

func isAllowedCustomWebhookEvent(ev string) bool {
	switch ev {
	case domain.EventPurchase, domain.EventAddPaymentInfo, domain.EventInitiateCheckout,
		domain.EventAddToCart, domain.EventLead, domain.EventCompleteRegistration,
		domain.EventContact, "Subscribe", "StartTrial":
		return true
	}
	return false
}
