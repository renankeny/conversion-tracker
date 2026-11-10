package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// guruPayload reflete o JSON oficial de webhooks de transação da Digital Manager Guru.
type guruPayload struct {
	APIToken string `json:"api_token"`
	ID       any    `json:"id"`
	Event    string `json:"event"`
	Status   string `json:"status"`
	Contact  struct {
		Name           string `json:"name"`
		Email          string `json:"email"`
		PhoneLocalCode string `json:"phone_local_code"`
		PhoneNumber    string `json:"phone_number"`
		Phone          string `json:"phone"`
	} `json:"contact"`
	Payment struct {
		Status   string `json:"status"`
		Total    any    `json:"total"`
		Value    any    `json:"value"`
		Amount   any    `json:"amount"`
		Currency string `json:"currency"`
		TID      string `json:"tid"`
	} `json:"payment"`
	Product struct {
		ID   any    `json:"id"`
		Name string `json:"name"`
	} `json:"product"`
	Items []struct {
		ID   any    `json:"id"`
		Name string `json:"name"`
	} `json:"items"`
	Source    map[string]any `json:"source"`
	Trackings map[string]any `json:"trackings"`
}

// Guru processa o webhook /webhooks/guru (Digital Manager Guru).
func (h *Webhooks) Guru(w http.ResponseWriter, r *http.Request) {
	var p guruPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	st := strings.ToLower(strings.TrimSpace(p.Status))
	paySt := strings.ToLower(strings.TrimSpace(p.Payment.Status))
	ev := strings.ToLower(strings.TrimSpace(p.Event))

	approved := st == "approved" ||
		st == "complete" ||
		st == "paid" ||
		paySt == "approved" ||
		paySt == "paid" ||
		ev == "order_approved"

	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + util.FirstNonEmpty(p.Status, p.Payment.Status, p.Event),
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "guru", p.APIToken) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Contact.Name))
	phone := util.FirstNonEmpty(
		joinPhoneDDD("", p.Contact.PhoneLocalCode, p.Contact.PhoneNumber),
		p.Contact.Phone,
	)

	value := util.ParseMoneyAny(p.Payment.Total)
	if value == 0 {
		value = util.ParseMoneyAny(p.Payment.Value)
	}
	if value == 0 {
		value = util.ParseMoneyAny(p.Payment.Amount)
	}

	prodID := anyString(p.Product.ID)
	prodName := p.Product.Name
	if prodID == "" && len(p.Items) > 0 {
		prodID = anyString(p.Items[0].ID)
		prodName = p.Items[0].Name
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Source),
		util.ExtractUserIDFromMap(p.Trackings),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "guru",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(anyString(p.ID), p.Payment.TID),
		Value:       value,
		Currency:    util.FirstNonEmpty(p.Payment.Currency, "BRL"),
		FirstName:   first,
		LastName:    last,
		Email:       p.Contact.Email,
		Phone:       phone,
		ProductID:   prodID,
		ProductName: prodName,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

