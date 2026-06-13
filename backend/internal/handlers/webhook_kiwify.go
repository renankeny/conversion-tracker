package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// kiwifyPayload reflete o JSON enviado pela Kiwify em compras.
//
// Importante:
//   - `Commissions.my_commission` vem em CENTAVOS (legado divide por 100).
//   - O userId é extraído de `TrackingParameters.utm_source` no formato
//     "<algo>|userId<ID>|<algo>".
type kiwifyPayload struct {
	OrderID  string `json:"order_id"`
	Customer struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Mobile   string `json:"mobile"`
	} `json:"Customer"`
	Commissions struct {
		Currency     string  `json:"currency"`
		MyCommission float64 `json:"my_commission"`
	} `json:"Commissions"`
	TrackingParameters struct {
		UTMSource string `json:"utm_source"`
	} `json:"TrackingParameters"`
}

// Kiwify processa o webhook /webhooks/kiwify.
func (h *Webhooks) Kiwify(w http.ResponseWriter, r *http.Request) {
	var p kiwifyPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.FullName))
	value := p.Commissions.MyCommission / 100.0 // centavos -> reais

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "kiwify",
		ExternalID:  util.ExtractUserIDFromUTM(p.TrackingParameters.UTMSource),
		OrderID:     p.OrderID,
		Value:       value,
		Currency:    p.Commissions.Currency,
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       p.Customer.Mobile,
	})
}
