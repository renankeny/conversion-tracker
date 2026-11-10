package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// greennPayload reflete o JSON de webhooks da Greenn (sale / saleUpdated).
type greennPayload struct {
	Type          string `json:"type"`
	Event         string `json:"event"`
	CurrentStatus string `json:"currentStatus"`
	Status        string `json:"status"`
	Token         string `json:"token"`
	Sale          struct {
		ID              any    `json:"id"`
		TransactionCode string `json:"transaction_code"`
		Status          string `json:"status"`
		Amount          any    `json:"amount"`
		TotalAmount     any    `json:"total_amount"`
		Currency        string `json:"currency"`
		Type            string `json:"type"`
	} `json:"sale"`
	Client struct {
		Name      string `json:"name"`
		Email     string `json:"email"`
		Cellphone string `json:"cellphone"`
		Phone     string `json:"phone"`
		Whatsapp  string `json:"whatsapp"`
	} `json:"client"`
	Product struct {
		ID   any    `json:"id"`
		Name string `json:"name"`
	} `json:"product"`
	Metas     map[string]any `json:"metas"`
	SaleMetas any            `json:"saleMetas"`
	Lead      map[string]any `json:"lead"`
}

// Greenn processa o webhook /webhooks/greenn.
func (h *Webhooks) Greenn(w http.ResponseWriter, r *http.Request) {
	var p greennPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	st := strings.ToLower(strings.TrimSpace(util.FirstNonEmpty(p.CurrentStatus, p.Sale.Status, p.Status)))
	approved := st == "paid" || st == "approved" || st == "pago"
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + st,
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "greenn", p.Token) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Client.Name))
	phone := util.FirstNonEmpty(p.Client.Cellphone, p.Client.Phone, p.Client.Whatsapp)

	value := util.ParseMoneyAny(p.Sale.Amount)
	if value == 0 {
		value = util.ParseMoneyAny(p.Sale.TotalAmount)
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Metas),
		util.ExtractUserIDFromMap(p.Lead),
		util.ExtractUserIDDeep(jsonBytes),
	)

	saleType := strings.ToLower(p.Sale.Type)
	in := purchaseInput{
		GatewayName: "greenn",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(p.Sale.TransactionCode, anyString(p.Sale.ID)),
		Value:       value,
		Currency:    util.FirstNonEmpty(p.Sale.Currency, "BRL"),
		FirstName:   first,
		LastName:    last,
		Email:       p.Client.Email,
		Phone:       phone,
		ProductID:   anyString(p.Product.ID),
		ProductName: p.Product.Name,
		IsOrderBump: strings.Contains(saleType, "bump"),
		IsUpsell:    strings.Contains(saleType, "upsell"),
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

