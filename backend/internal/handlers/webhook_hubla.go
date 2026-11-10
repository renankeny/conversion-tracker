package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// hublaPayload reflete o Webhook 2.0 oficial da Hubla (invoice.status_updated / invoice.payment_succeeded / NewSale).
type hublaPayload struct {
	Type    string `json:"type"`
	Version string `json:"version"`
	Token   string `json:"token"`
	Event   struct {
		Product struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"product"`
		Products []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"products"`
		Invoice struct {
			ID               string `json:"id"`
			OrderID          string `json:"orderId"`
			Status           string `json:"status"`
			AmountCents      any    `json:"amountCents"`
			TotalAmountCents any    `json:"totalAmountCents"`
			Currency         string `json:"currency"`
		} `json:"invoice"`
		Payer struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Name      string `json:"name"`
			Email     string `json:"email"`
			Phone     string `json:"phone"`
		} `json:"payer"`
		User struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Name      string `json:"name"`
			Email     string `json:"email"`
			Phone     string `json:"phone"`
		} `json:"user"`
		UTM map[string]any `json:"utm"`
	} `json:"event"`
}

// Hubla processa o webhook /webhooks/hubla.
func (h *Webhooks) Hubla(w http.ResponseWriter, r *http.Request) {
	var p hublaPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	evType := strings.ToLower(strings.TrimSpace(p.Type))
	invStatus := strings.ToLower(strings.TrimSpace(p.Event.Invoice.Status))

	approved := (evType == "invoice.status_updated" && invStatus == "paid") ||
		evType == "invoice.payment_succeeded" ||
		evType == "newsale" ||
		(evType == "" && invStatus == "paid")

	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: type=" + p.Type + " status=" + p.Event.Invoice.Status,
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "hubla", p.Token) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first := strings.ToLower(util.FirstNonEmpty(p.Event.Payer.FirstName, p.Event.User.FirstName))
	last := strings.ToLower(util.FirstNonEmpty(p.Event.Payer.LastName, p.Event.User.LastName))
	if first == "" && last == "" {
		first, last = util.SplitFullName(strings.ToLower(util.FirstNonEmpty(p.Event.Payer.Name, p.Event.User.Name)))
	}

	value := util.ParseMoneyCents(p.Event.Invoice.AmountCents)
	if value == 0 {
		value = util.ParseMoneyCents(p.Event.Invoice.TotalAmountCents)
	}

	prodID := p.Event.Product.ID
	prodName := p.Event.Product.Name
	if prodID == "" && len(p.Event.Products) > 0 {
		prodID = p.Event.Products[0].ID
		prodName = p.Event.Products[0].Name
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Event.UTM),
		util.ExtractUserIDDeep(jsonBytes),
	)

	orderID := util.FirstNonEmpty(
		p.Event.Invoice.OrderID,
		p.Event.Invoice.ID,
		r.Header.Get("X-Hubla-Idempotency"),
	)

	in := purchaseInput{
		GatewayName: "hubla",
		ExternalID:  externalID,
		OrderID:     orderID,
		Value:       value,
		Currency:    util.FirstNonEmpty(p.Event.Invoice.Currency, "BRL"),
		FirstName:   first,
		LastName:    last,
		Email:       util.FirstNonEmpty(p.Event.Payer.Email, p.Event.User.Email),
		Phone:       util.FirstNonEmpty(p.Event.Payer.Phone, p.Event.User.Phone),
		ProductID:   prodID,
		ProductName: prodName,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

