package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type appmaxData struct {
	ID         any    `json:"id"`
	OrderID    any    `json:"order_id"`
	Status     string `json:"status"`
	StatusCode any    `json:"status_code"`
	Total      any    `json:"total"`
	Subtotal   any    `json:"subtotal"`
	Customer   struct {
		Name      string `json:"name"`
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
		Email     string `json:"email"`
		Telephone string `json:"telephone"`
		Phone     string `json:"phone"`
	} `json:"customer"`
	Products []struct {
		ID   any    `json:"id"`
		SKU  string `json:"sku"`
		Name string `json:"name"`
	} `json:"products"`
	Tracking map[string]any `json:"tracking"`
	Meta     map[string]any `json:"meta"`
}

// appmaxPayload suporta tanto Modelo Padrão ({event, data: {...}}) quanto modelo plano.
type appmaxPayload struct {
	Event       string     `json:"event"`
	Token       string     `json:"token"`
	AccessToken string     `json:"access_token"`
	Data        appmaxData `json:"data"`
	appmaxData
}

// Appmax processa o webhook /webhooks/appmax (Apphooks).
func (h *Webhooks) Appmax(w http.ResponseWriter, r *http.Request) {
	var p appmaxPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	d := p.Data
	if anyString(d.ID) == "" && anyString(d.OrderID) == "" {
		d = p.appmaxData
	}

	ev := strings.ToLower(strings.TrimSpace(p.Event))
	st := strings.ToLower(strings.TrimSpace(d.Status))
	approved := ev == "orderpaid" ||
		ev == "orderapproved" ||
		ev == "order.paid" ||
		ev == "order_paid" ||
		ev == "order_approved" ||
		st == "aprovado" ||
		st == "integrado" ||
		st == "pago" ||
		st == "paid"

	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: " + util.FirstNonEmpty(p.Event, d.Status),
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "appmax", p.Token, p.AccessToken) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first := strings.ToLower(d.Customer.Firstname)
	last := strings.ToLower(d.Customer.Lastname)
	if first == "" && last == "" {
		first, last = util.SplitFullName(strings.ToLower(d.Customer.Name))
	}

	value := util.ParseMoneyAny(d.Total)
	if value == 0 {
		value = util.ParseMoneyAny(d.Subtotal)
	}

	var prodID, prodName string
	if len(d.Products) > 0 {
		prodID = util.FirstNonEmpty(anyString(d.Products[0].ID), d.Products[0].SKU)
		prodName = d.Products[0].Name
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(d.Tracking),
		util.ExtractUserIDFromMap(d.Meta),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "appmax",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(anyString(d.OrderID), anyString(d.ID)),
		Value:       value,
		Currency:    "BRL",
		FirstName:   first,
		LastName:    last,
		Email:       d.Customer.Email,
		Phone:       util.FirstNonEmpty(d.Customer.Telephone, d.Customer.Phone),
		ProductID:   prodID,
		ProductName: prodName,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

