package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// kirvanoPayload reflete o JSON oficial da Kirvano.
type kirvanoPayload struct {
	Event         string `json:"event"`
	Status        string `json:"status"`
	SaleID        string `json:"sale_id"`
	SecurityToken string `json:"security_token"`
	Token         string `json:"token"`
	TotalPrice    string `json:"total_price"` // string "R$ 329,65"
	Fiscal        struct {
		TotalValue    float64 `json:"total_value"`
		OriginalValue float64 `json:"original_value"`
		NetValue      float64 `json:"net_value"`
	} `json:"fiscal"`
	Customer struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phone_number"`
		Document    string `json:"document"`
	} `json:"customer"`
	Products []struct {
		ID          any    `json:"id"`
		Name        string `json:"name"`
		OfferID     string `json:"offer_id"`
		OfferName   string `json:"offer_name"`
		IsOrderBump bool   `json:"is_order_bump"`
	} `json:"products"`
	UTM map[string]any `json:"utm"`
}

// Kirvano processa o webhook /webhooks/kirvano.
func (h *Webhooks) Kirvano(w http.ResponseWriter, r *http.Request) {
	raw, err := readWebhookBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var p kirvanoPayload
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ev := strings.ToUpper(p.Event)
	st := strings.ToUpper(p.Status)
	approved := ev == "SALE_APPROVED" || st == "APPROVED" || ev == "PURCHASE_APPROVED"
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: event=" + p.Event + " status=" + p.Status,
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "kirvano", p.SecurityToken, p.Token) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.Name))

	value := p.Fiscal.TotalValue
	if value == 0 {
		value = util.ParseMoneyAny(p.TotalPrice)
	}
	if value == 0 {
		value = p.Fiscal.OriginalValue
	}

	var productID, productName string
	var isOrderBump bool
	if len(p.Products) > 0 {
		productID = util.FirstNonEmpty(anyString(p.Products[0].ID), p.Products[0].OfferID)
		productName = util.FirstNonEmpty(p.Products[0].Name, p.Products[0].OfferName)
		isOrderBump = p.Products[0].IsOrderBump
	}

	in := purchaseInput{
		GatewayName: "kirvano",
		ExternalID: util.FirstNonEmpty(
			util.ExtractUserIDFromMap(p.UTM),
			util.ExtractUserIDDeep(raw),
		),
		OrderID:     p.SaleID,
		Value:       value,
		Currency:    "BRL",
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       p.Customer.PhoneNumber,
		ProductID:   productID,
		ProductName: productName,
		IsOrderBump: isOrderBump,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}
