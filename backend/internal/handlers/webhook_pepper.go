package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// pepperPayload reflete o JSON de webhooks da Pepper (v2).
type pepperPayload struct {
	Event         string `json:"event"`
	Hash          string `json:"hash"`
	Transaction   string `json:"transaction"`
	PaymentStatus string `json:"payment_status"`
	Status        string `json:"status"`
	Token         string `json:"token"`
	Secret        string `json:"secret"`
	Amount        any    `json:"amount"` // em centavos (ex: 10000 = R$ 100,00)
	UTMSource     string `json:"utm_source"`
	UTMCampaign   string `json:"utm_campaign"`
	UTMContent    string `json:"utm_content"`
	UTMTerm       string `json:"utm_term"`
	UTMMedium     string `json:"utm_medium"`
	SRC           string `json:"src"`
	SCK           string `json:"sck"`
	Customer      struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phone_number"`
		Phone       string `json:"phone"`
	} `json:"customer"`
	Offer struct {
		Hash  string `json:"hash"`
		Title string `json:"title"`
		Price any    `json:"price"` // em centavos
	} `json:"offer"`
	Product struct {
		Hash  string `json:"hash"`
		ID    any    `json:"id"`
		Title string `json:"title"`
		Name  string `json:"name"`
	} `json:"product"`
}

// Pepper processa o webhook /webhooks/pepper.
func (h *Webhooks) Pepper(w http.ResponseWriter, r *http.Request) {
	var p pepperPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	st := strings.ToLower(strings.TrimSpace(util.FirstNonEmpty(p.PaymentStatus, p.Status)))
	ev := strings.ToLower(strings.TrimSpace(p.Event))
	approved := st == "paid" || st == "approved" || st == "pago" || ev == "purchase_approved" || ev == "transaction.paid"
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + util.FirstNonEmpty(p.PaymentStatus, p.Status, p.Event),
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "pepper", p.Token, p.Secret) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.Name))
	phone := util.FirstNonEmpty(p.Customer.PhoneNumber, p.Customer.Phone)

	value := util.ParseMoneyCents(p.Amount)
	if value == 0 {
		value = util.ParseMoneyCents(p.Offer.Price)
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromAny(p.UTMSource, p.UTMTerm, p.UTMContent, p.UTMMedium, p.UTMCampaign, p.SRC, p.SCK),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "pepper",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(p.Transaction, p.Hash),
		Value:       value,
		Currency:    "BRL",
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       phone,
		ProductID:   util.FirstNonEmpty(p.Product.Hash, anyString(p.Product.ID), p.Offer.Hash),
		ProductName: util.FirstNonEmpty(p.Product.Title, p.Product.Name, p.Offer.Title),
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

