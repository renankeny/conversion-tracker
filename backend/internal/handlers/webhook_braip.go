package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// braipPayload reflete o JSON da Braip.
//
// Braip envia o status em `trans_status` (ex: "paid", "approved", "refused").
// Só consideramos Purchase quando aprovado.
//
// O `userId` chega em `trans_originalsource` (que é o utm_source de origem)
// ou no campo `sck`/`source`.
type braipPayload struct {
	TransCod    string  `json:"trans_cod"`
	TransStatus string  `json:"trans_status"`
	TransValue  float64 `json:"trans_value"`     // já em reais
	TransCurrency string `json:"trans_currency"`
	Customer    struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"customer"`
	TransOriginalSource string `json:"trans_originalsource"`
	SCK                 string `json:"sck"`
	Source              string `json:"source"`
}

// Braip processa o webhook /webhooks/braip.
func (h *Webhooks) Braip(w http.ResponseWriter, r *http.Request) {
	var p braipPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	st := strings.ToLower(p.TransStatus)
	if st != "paid" && st != "approved" && st != "aprovado" {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + p.TransStatus,
		})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.Name))
	utmSource := util.FirstNonEmpty(p.TransOriginalSource, p.SCK, p.Source)

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "braip",
		ExternalID:  util.ExtractUserIDFromUTM(utmSource),
		OrderID:     p.TransCod,
		Value:       p.TransValue,
		Currency:    p.TransCurrency,
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       p.Customer.Phone,
	})
}
