package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// doppusPayload reflete o JSON de webhooks da Doppus.
type doppusPayload struct {
	ID              any             `json:"id"`
	Event           string          `json:"event"`
	TransactionCode string          `json:"transaction_code"`
	Token           string          `json:"token"`
	StatusRaw       json.RawMessage `json:"status"` // string OU objeto {code, message}
	Customer        struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"customer"`
	Transaction struct {
		TotalValue any    `json:"total_value"`
		SaleValue  any    `json:"sale_value"`
		Status     string `json:"status"`
	} `json:"transaction"`
	Items []struct {
		ItemCode string `json:"item_code"`
		ItemName string `json:"item_name"`
		Code     string `json:"code"`
		Name     string `json:"name"`
	} `json:"items"`
	UTM      map[string]any `json:"utm"`
	Tracking map[string]any `json:"tracking"`
}

func parseDoppusStatus(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		Code   string `json:"code"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return util.FirstNonEmpty(obj.Code, obj.Status)
	}
	return ""
}

// Doppus processa o webhook /webhooks/doppus.
func (h *Webhooks) Doppus(w http.ResponseWriter, r *http.Request) {
	var p doppusPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	st := strings.ToLower(strings.TrimSpace(util.FirstNonEmpty(parseDoppusStatus(p.StatusRaw), p.Transaction.Status)))
	ev := strings.ToLower(strings.TrimSpace(p.Event))

	value := util.ParseMoneyAny(p.Transaction.TotalValue)
	if value == 0 {
		value = util.ParseMoneyAny(p.Transaction.SaleValue)
	}

	// Na Doppus, quando o webhook é disparado apenas na rota de venda aprovada, status pode vir omitido.
	approved := st == "approved" ||
		st == "paid" ||
		st == "aprovado" ||
		st == "complete" ||
		ev == "purchase_approved" ||
		ev == "sale_approved" ||
		(st == "" && ev == "" && p.TransactionCode != "" && value > 0)

	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + util.FirstNonEmpty(st, p.Event),
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "doppus", p.Token) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.Name))

	var prodID, prodName string
	if len(p.Items) > 0 {
		prodID = util.FirstNonEmpty(p.Items[0].ItemCode, p.Items[0].Code)
		prodName = util.FirstNonEmpty(p.Items[0].ItemName, p.Items[0].Name)
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.UTM),
		util.ExtractUserIDFromMap(p.Tracking),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "doppus",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(p.TransactionCode, anyString(p.ID)),
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

