package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type cartpandaOrder struct {
	ID              any    `json:"id"`
	Number          any    `json:"number"`
	Token           string `json:"token"`
	Status          string `json:"status"`
	FinancialStatus string `json:"financial_status"`
	Currency        string `json:"currency"`
	TotalPrice      any    `json:"total_price"`
	SubtotalPrice   any    `json:"subtotal_price"`
	LandingSite     string `json:"landing_site"`
	Customer        struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
	} `json:"customer"`
	LineItems []struct {
		ProductID any    `json:"product_id"`
		Title     string `json:"title"`
	} `json:"line_items"`
	ConversionSummary map[string]any `json:"conversion_summary"`
}

// cartpandaPayload suporta tanto envelope {event, order: {...}} quanto pedido direto na raiz.
type cartpandaPayload struct {
	Event string         `json:"event"`
	Order cartpandaOrder `json:"order"`
	cartpandaOrder
}

// Cartpanda processa o webhook /webhooks/cartpanda.
func (h *Webhooks) Cartpanda(w http.ResponseWriter, r *http.Request) {
	rawBody, _ := readWebhookBody(r)
	var p cartpandaPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ord := p.Order
	if anyString(ord.ID) == "" && anyString(p.cartpandaOrder.ID) != "" {
		ord = p.cartpandaOrder
	}

	ev := strings.ToLower(strings.TrimSpace(p.Event))
	finSt := strings.ToLower(strings.TrimSpace(util.FirstNonEmpty(ord.FinancialStatus, ord.Status)))
	approved := ev == "order.paid" || ev == "order_paid" || finSt == "paid"
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: " + util.FirstNonEmpty(p.Event, finSt),
		})
		return
	}

	accID := ExtractAccountID(r)
	hmacHeader := util.FirstNonEmpty(
		r.Header.Get("X-Cartpanda-Hmac-Sha256"),
		r.Header.Get("X-Shop-Hmac-Sha256"),
	)
	if !h.validateHMACOrToken(r.Context(), r, accID, "cartpanda", rawBody, hmacHeader, "", ord.Token) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first := strings.ToLower(ord.Customer.FirstName)
	last := strings.ToLower(ord.Customer.LastName)
	if first == "" && last == "" {
		first, last = util.SplitFullName(strings.ToLower(ord.Customer.Name))
	}

	value := util.ParseMoneyAny(ord.TotalPrice)
	if value == 0 {
		value = util.ParseMoneyAny(ord.SubtotalPrice)
	}

	var prodID, prodName string
	if len(ord.LineItems) > 0 {
		prodID = anyString(ord.LineItems[0].ProductID)
		prodName = ord.LineItems[0].Title
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(ord.ConversionSummary),
		util.ExtractUserIDFromUTM(ord.LandingSite),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "cartpanda",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(anyString(ord.ID), anyString(ord.Number)),
		Value:       value,
		Currency:    util.FirstNonEmpty(ord.Currency, "BRL"),
		FirstName:   first,
		LastName:    last,
		Email:       ord.Customer.Email,
		Phone:       ord.Customer.Phone,
		ProductID:   prodID,
		ProductName: prodName,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

