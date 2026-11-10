package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// tictoPayload reflete o JSON oficial de webhooks v2.0 da Ticto.
type tictoPayload struct {
	Token           string `json:"token"`
	Status          string `json:"status"`
	TransactionHash string `json:"transaction_hash"`
	Order           struct {
		ID         any    `json:"id"`
		Hash       string `json:"hash"`
		PaidAmount any    `json:"paid_amount"` // em centavos
	} `json:"order"`
	Item struct {
		ID          any    `json:"id"`
		ProductID   any    `json:"product_id"`
		Name        string `json:"name"`
		ProductName string `json:"product_name"`
		Amount      any    `json:"amount"` // em centavos
		Price       any    `json:"price"`  // em centavos
	} `json:"item"`
	Customer struct {
		Name  string          `json:"name"`
		Email string          `json:"email"`
		Phone json.RawMessage `json:"phone"` // string OU objeto {ddi, ddd, number}
	} `json:"customer"`
	Tracking  map[string]any `json:"tracking"`
	URLParams struct {
		QueryParams map[string]any `json:"query_params"`
	} `json:"url_params"`
	QueryParams map[string]any `json:"query_params"`
}

func parseTictoPhone(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		DDI    string `json:"ddi"`
		DDD    string `json:"ddd"`
		Number string `json:"number"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return joinPhoneDDD(obj.DDI, obj.DDD, obj.Number)
	}
	return ""
}

// Ticto processa o webhook /webhooks/ticto.
func (h *Webhooks) Ticto(w http.ResponseWriter, r *http.Request) {
	var p tictoPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	st := strings.ToLower(strings.TrimSpace(p.Status))
	approved := st == "authorized" || st == "approved" || st == "paid" || st == "completed" || st == "venda_realizada"
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + p.Status,
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "ticto", p.Token) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.Name))
	phone := parseTictoPhone(p.Customer.Phone)

	value := util.ParseMoneyCents(p.Order.PaidAmount)
	if value == 0 {
		value = util.ParseMoneyCents(p.Item.Amount)
	}
	if value == 0 {
		value = util.ParseMoneyCents(p.Item.Price)
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Tracking),
		util.ExtractUserIDFromMap(p.URLParams.QueryParams),
		util.ExtractUserIDFromMap(p.QueryParams),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "ticto",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(p.Order.Hash, p.TransactionHash, anyString(p.Order.ID)),
		Value:       value,
		Currency:    "BRL",
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       phone,
		ProductID:   util.FirstNonEmpty(anyString(p.Item.ProductID), anyString(p.Item.ID)),
		ProductName: util.FirstNonEmpty(p.Item.ProductName, p.Item.Name),
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

