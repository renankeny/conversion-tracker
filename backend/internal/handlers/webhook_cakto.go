package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// caktoPayload reflete o JSON REAL da Cakto.
//
// Estrutura observada em produção (junho/2026):
//
//	{
//	  "event": "purchase_approved",
//	  "secret": "uuid",                  // opcional, valida via env CAKTO_WEBHOOK_SECRET
//	  "data": {
//	    "id": "uuid",
//	    "amount": 421.28,                // valor cobrado (com juros do parcelamento)
//	    "baseAmount": 347,               // preço original da oferta (mais útil pra Meta)
//	    "status": "paid",
//	    "customer": { name, email, phone, ... },
//	    "offer":    { id, name, price, currency, ... },
//	    "fbc": null, "fbp": null,        // capturado pela Cakto, useful pra atribuição
//	    "sck": null,
//	    "utm_source":   null,
//	    "utm_term":     null,
//	    "utm_medium":   null,
//	    "utm_content":  null,
//	    "utm_campaign": null,
//	    ...
//	  }
//	}
//
// Tratamos só status=paid OU event=purchase_approved como Purchase.
type caktoPayload struct {
	Event  string `json:"event"`
	Secret string `json:"secret"`
	Data   struct {
		ID         string  `json:"id"`
		Amount     float64 `json:"amount"`     // valor cobrado (com juros)
		BaseAmount float64 `json:"baseAmount"` // preço original da oferta
		Status     string  `json:"status"`
		Customer   struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Phone string `json:"phone"`
		} `json:"customer"`
		Offer struct {
			ID       string  `json:"id"`
			Name     string  `json:"name"`
			Price    float64 `json:"price"`
			Currency string  `json:"currency"`
		} `json:"offer"`
		FBC         string `json:"fbc"`
		FBP         string `json:"fbp"`
		SCK         string `json:"sck"`
		UTMSource   string `json:"utm_source"`
		UTMTerm     string `json:"utm_term"`
		UTMMedium   string `json:"utm_medium"`
		UTMContent  string `json:"utm_content"`
		UTMCampaign string `json:"utm_campaign"`
	} `json:"data"`
}

// Cakto processa o webhook /webhooks/cakto.
//
// Aceita o evento `purchase_approved` ou status `paid`. Os demais são
// ignorados retornando 200 OK pra evitar retry desnecessário do gateway.
func (h *Webhooks) Cakto(w http.ResponseWriter, r *http.Request) {
	var p caktoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Filtra só compras aprovadas. status=paid OU event=purchase_approved.
	st := strings.ToLower(p.Data.Status)
	ev := strings.ToLower(p.Event)
	approved := st == "paid" || ev == "purchase_approved" || ev == "pedido_aprovado" || ev == "approved"
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: event=" + p.Event + " status=" + p.Data.Status,
		})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Data.Customer.Name))

	// Valor: prioriza baseAmount (preço original da oferta) sobre amount
	// (que inclui juros do parcelamento). Em fallback, usa offer.price.
	value := p.Data.BaseAmount
	if value == 0 {
		value = p.Data.Amount
	}
	if value == 0 {
		value = p.Data.Offer.Price
	}

	currency := p.Data.Offer.Currency
	if currency == "" {
		currency = "BRL"
	}

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "cakto",
		ExternalID: util.ExtractUserIDFromAny(
			p.Data.UTMSource, p.Data.UTMTerm, p.Data.UTMContent,
			p.Data.UTMMedium, p.Data.UTMCampaign,
			p.Data.SCK,
		),
		OrderID:   p.Data.ID,
		Value:     value,
		Currency:  currency,
		FirstName: first,
		LastName:  last,
		Email:     p.Data.Customer.Email,
		Phone:     p.Data.Customer.Phone,
	})
}
