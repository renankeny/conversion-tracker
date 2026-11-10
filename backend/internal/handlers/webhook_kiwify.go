package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type kiwifyPayload struct {
	OrderID          string `json:"order_id"`
	OrderRef         string `json:"order_ref"`
	OrderStatus      string `json:"order_status"`
	WebhookEventType string `json:"webhook_event_type"`
	Signature        string `json:"signature"`
	Type             string `json:"type"`
	Product          struct {
		ProductID   string `json:"product_id"`
		ProductName string `json:"product_name"`
	} `json:"Product"`
	Customer struct {
		FullName  string `json:"full_name"`
		FirstName string `json:"first_name"`
		Email     string `json:"email"`
		Mobile    string `json:"mobile"`
	} `json:"Customer"`
	Commissions struct {
		ChargeAmount     any    `json:"charge_amount"`      // centavos (int ou string "9700")
		ProductBasePrice any    `json:"product_base_price"` // centavos
		Currency         string `json:"currency"`
	} `json:"Commissions"`
	TrackingParameters map[string]any `json:"TrackingParameters"`
}

// Kiwify processa o webhook /webhooks/kiwify.
// Valida HMAC-SHA1 (?signature=) sobre o raw body quando o token está configurado no painel.
func (h *Webhooks) Kiwify(w http.ResponseWriter, r *http.Request) {
	raw, err := readWebhookBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	var p kiwifyPayload
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ev := strings.ToLower(strings.TrimSpace(p.WebhookEventType))
	st := strings.ToLower(strings.TrimSpace(p.OrderStatus))
	approved := ev == "order_approved" || ev == "compra_aprovada" || (ev == "" && st == "paid")
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: " + util.FirstNonEmpty(p.WebhookEventType, p.OrderStatus),
		})
		return
	}

	accID := ExtractAccountID(r)
	sigSHA1 := util.FirstNonEmpty(r.URL.Query().Get("signature"), r.Header.Get("X-Kiwify-Signature"), p.Signature)
	if !h.validateHMACOrToken(r.Context(), r, accID, "kiwify", raw, "", sigSHA1, p.Signature) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(p.Customer.FullName)
	if p.Customer.FirstName != "" && first == "" {
		first = p.Customer.FirstName
	}

	value := util.ParseMoneyCents(p.Commissions.ChargeAmount)
	if value == 0 {
		value = util.ParseMoneyCents(p.Commissions.ProductBasePrice)
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.TrackingParameters),
		util.ExtractUserIDDeep(raw),
	)

	in := purchaseInput{
		GatewayName: "kiwify",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(p.OrderID, p.OrderRef),
		Value:       value,
		Currency:    util.FirstNonEmpty(p.Commissions.Currency, "BRL"),
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       p.Customer.Mobile,
		ProductID:   p.Product.ProductID,
		ProductName: p.Product.ProductName,
		IsOrderBump: strings.EqualFold(p.Type, "order_bump"),
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}

	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}
