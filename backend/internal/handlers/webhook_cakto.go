package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// caktoPayload reflete o JSON da Cakto.
//
// Cakto envia eventos como `pedido_aprovado`, `pedido_recusado`, etc. Tratamos
// só o `pedido_aprovado` como Purchase. Os demais não disparam evento.
//
// O `userId` vem geralmente em `tracking.utm_source` ou no campo legado `sck`.
type caktoPayload struct {
	Event    string `json:"event"`
	Order    struct {
		ID       string  `json:"id"`
		Total    float64 `json:"total"`     // já em reais
		Currency string  `json:"currency"`
	} `json:"order"`
	Customer struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"customer"`
	Tracking struct {
		UTMSource string `json:"utm_source"`
		SCK       string `json:"sck"`
	} `json:"tracking"`
}

// Cakto processa o webhook /webhooks/cakto.
func (h *Webhooks) Cakto(w http.ResponseWriter, r *http.Request) {
	var p caktoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Cakto só conta como Purchase quando aprova; os demais ignoramos.
	if !strings.EqualFold(p.Event, "pedido_aprovado") &&
		!strings.EqualFold(p.Event, "purchase_approved") &&
		!strings.EqualFold(p.Event, "approved") {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: " + p.Event,
		})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.Name))
	utmSource := util.FirstNonEmpty(p.Tracking.UTMSource, p.Tracking.SCK)

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "cakto",
		ExternalID:  util.ExtractUserIDFromUTM(utmSource),
		OrderID:     p.Order.ID,
		Value:       p.Order.Total,
		Currency:    p.Order.Currency,
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       p.Customer.Phone,
	})
}
