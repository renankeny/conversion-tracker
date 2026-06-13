package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// hotmartPayload reflete o JSON da Hotmart.
//
// O userId é extraído de `data.purchase.origin.xcod` no formato
// "<algo>|userId<ID>|<algo>". O campo `xcod` é a origem completa (UTMs
// concatenadas) que o Checkout repassa.
type hotmartPayload struct {
	ID   string `json:"id"`
	Data struct {
		Buyer struct {
			Name          string `json:"name"`
			FirstName     string `json:"first_name"`
			LastName      string `json:"last_name"`
			Email         string `json:"email"`
			CheckoutPhone string `json:"checkout_phone"`
			Document      string `json:"document"`
			Phone         struct {
				Number string `json:"number"`
			} `json:"phone"`
		} `json:"buyer"`
		Purchase struct {
			Transaction string `json:"transaction"`
			Origin      struct {
				Xcod string `json:"xcod"`
			} `json:"origin"`
			Price struct {
				Value         float64 `json:"value"`
				CurrencyValue string  `json:"currency_value"`
				CurrencyCode  string  `json:"currency_code"`
			} `json:"price"`
			FullPrice struct {
				Value         float64 `json:"value"`
				CurrencyValue string  `json:"currency_value"`
			} `json:"full_price"`
		} `json:"purchase"`
	} `json:"data"`
}

// Hotmart processa o webhook /webhooks/hotmart.
func (h *Webhooks) Hotmart(w http.ResponseWriter, r *http.Request) {
	var p hotmartPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	first, last := hotmartName(p)
	phone := util.FirstNonEmpty(p.Data.Buyer.CheckoutPhone, p.Data.Buyer.Phone.Number)
	value := p.Data.Purchase.Price.Value
	if value == 0 {
		value = p.Data.Purchase.FullPrice.Value
	}
	currency := util.FirstNonEmpty(
		p.Data.Purchase.Price.CurrencyValue,
		p.Data.Purchase.FullPrice.CurrencyValue,
		p.Data.Purchase.Price.CurrencyCode,
	)

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "hotmart",
		ExternalID:  util.ExtractUserIDFromUTM(p.Data.Purchase.Origin.Xcod),
		OrderID:     p.Data.Purchase.Transaction,
		Value:       value,
		Currency:    currency,
		FirstName:   first,
		LastName:    last,
		Email:       p.Data.Buyer.Email,
		Phone:       phone,
	})
}

// hotmartName escolhe o melhor par (first, last) considerando que o payload
// pode trazer `first_name`/`last_name` separados ou só o `name` completo.
func hotmartName(p hotmartPayload) (string, string) {
	first := strings.ToLower(p.Data.Buyer.FirstName)
	last := strings.ToLower(p.Data.Buyer.LastName)
	if first != "" || last != "" {
		return first, last
	}
	return util.SplitFullName(strings.ToLower(p.Data.Buyer.Name))
}
