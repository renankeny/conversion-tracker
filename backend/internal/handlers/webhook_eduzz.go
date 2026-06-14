package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// eduzzPayload reflete o JSON enviado pela Eduzz na compra (eventos
// "myeduzz.invoice_paid" e similares).
//
// Doc oficial: https://docs.eduzz.com/api/webhook/
//
// O `utm_source` chega em `trans_ref` ou em `tracking.utm_source`, dependendo
// do tipo de origem. Tentamos os dois.
type eduzzPayload struct {
	TransCod   string  `json:"trans_cod"`    // ID da transação (== order_id)
	TransRef   string  `json:"trans_ref"`    // pode trazer userId quando vinda de link com sck/utm
	Cus        string  `json:"cus_email"`    // legado
	CusEmail   string  `json:"customer_email"`
	CusName    string  `json:"customer_name"`
	CusPhone   string  `json:"customer_phone"`
	CusMobile  string  `json:"customer_mobile"`
	Amount     float64 `json:"trans_value"`  // já em reais
	Currency   string  `json:"trans_currency"`
	Tracking   struct {
		UTMSource string `json:"utm_source"`
	} `json:"tracking"`
}

// Eduzz processa o webhook /webhooks/eduzz.
func (h *Webhooks) Eduzz(w http.ResponseWriter, r *http.Request) {
	var p eduzzPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.CusName))
	phone := util.FirstNonEmpty(p.CusPhone, p.CusMobile)
	email := util.FirstNonEmpty(p.CusEmail, p.Cus)
	utmSource := util.FirstNonEmpty(p.Tracking.UTMSource, p.TransRef)

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "eduzz",
		ExternalID:  util.ExtractUserIDFromUTM(utmSource),
		OrderID:     p.TransCod,
		Value:       p.Amount,
		Currency:    p.Currency,
		FirstName:   first,
		LastName:    last,
		Email:       email,
		Phone:       phone,
	})
}
