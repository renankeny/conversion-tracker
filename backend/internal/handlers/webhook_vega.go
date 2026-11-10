package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// vegaPayload reflete o JSON de Webhooks 1.0 da Vega Checkout.
type vegaPayload struct {
	ID            any    `json:"id"`
	TransactionID string `json:"transaction_id"`
	OrderID       string `json:"order_id"`
	CheckoutID    string `json:"checkout_id"`
	Event         string `json:"event"`
	Status        string `json:"status"`
	PaymentStatus string `json:"payment_status"`
	Token         string `json:"token"`
	Secret        string `json:"secret"`
	TotalPrice    any    `json:"total_price"`
	Amount        any    `json:"amount"`
	Customer      struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"customer"`
	Plans []struct {
		ID       any    `json:"id"`
		Name     string `json:"name"`
		Value    any    `json:"value"`
		Products []struct {
			ID   any    `json:"id"`
			Name string `json:"name"`
		} `json:"products"`
	} `json:"plans"`
	Products []struct {
		ID   any    `json:"id"`
		Name string `json:"name"`
	} `json:"products"`
	Checkout map[string]any `json:"checkout"`
	UTMs     map[string]any `json:"utms"`
}

// Vega processa o webhook /webhooks/vega (Vega Checkout).
func (h *Webhooks) Vega(w http.ResponseWriter, r *http.Request) {
	var p vegaPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	st := strings.ToLower(strings.TrimSpace(util.FirstNonEmpty(p.Status, p.PaymentStatus)))
	ev := strings.ToLower(strings.TrimSpace(p.Event))
	approved := st == "approved" ||
		st == "paid" ||
		st == "pago" ||
		st == "aprovado" ||
		ev == "order.approved" ||
		ev == "sale.approved" ||
		ev == "purchase_approved"

	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + util.FirstNonEmpty(st, p.Event),
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "vega", p.Token, p.Secret) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.Name))

	value := util.ParseMoneyAny(p.TotalPrice)
	if value == 0 {
		value = util.ParseMoneyAny(p.Amount)
	}

	var prodID, prodName string
	if len(p.Plans) > 0 {
		plan := p.Plans[0]
		if value == 0 {
			value = util.ParseMoneyAny(plan.Value)
		}
		if len(plan.Products) > 0 {
			prodID = anyString(plan.Products[0].ID)
			prodName = plan.Products[0].Name
		}
		prodID = util.FirstNonEmpty(prodID, anyString(plan.ID))
		prodName = util.FirstNonEmpty(prodName, plan.Name)
	} else if len(p.Products) > 0 {
		prodID = anyString(p.Products[0].ID)
		prodName = p.Products[0].Name
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Checkout),
		util.ExtractUserIDFromMap(p.UTMs),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "vega",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(p.TransactionID, p.OrderID, anyString(p.ID), p.CheckoutID),
		Value:       value,
		Currency:    "BRL",
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       p.Customer.Phone,
		ProductID:   prodID,
		ProductName: prodName,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

