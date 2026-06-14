package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// hotmartPayload reflete o JSON da Hotmart, suportando v1.x (legacy com
// `origin.xcod`) e v2.0 (atual, com `sckPaymentLink`, `offer.code` etc).
//
// Estrutura observada em produção (junho/2026, v2.0):
//
//	{
//	  "id":            "uuid",
//	  "event":         "PURCHASE_APPROVED",
//	  "version":       "2.0.0",
//	  "creation_date": <epoch ms>,
//	  "data": {
//	    "buyer": {
//	      "name":           "Teste Comprador",
//	      "first_name":     "Teste",
//	      "last_name":      "Comprador",
//	      "email":          "...",
//	      "checkout_phone": "99999999900",
//	      "document":       "...",
//	      "address":        { ... }
//	    },
//	    "purchase": {
//	      "transaction":     "HP1601...",
//	      "status":          "APPROVED",
//	      "approved_date":   <epoch ms>,
//	      "sckPaymentLink":  "<carrega utm_source / userId quando configurado>",
//	      "offer":           { code, coupon_code },
//	      "price":           { currency_value, value },
//	      "full_price":      { currency_value, value },
//	      "original_offer_price": { currency_value, value },
//	      "payment":         { installments_number, type },
//	      "business_model":  "I",
//	      "tracking":        { utm_source, ... }   // pode existir em algumas integrações
//	    },
//	    "affiliates": [{ "affiliate_code": "...", "name": "..." }],
//	    "producer":   { ... },
//	    "subscription": { ... }
//	  }
//	}
type hotmartPayload struct {
	ID    string `json:"id"`
	Event string `json:"event"`
	Data  struct {
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

			// v1 legado: origin.xcod com pipes "|userId<id>|".
			Origin map[string]any `json:"origin"`

			// v2.0: campos diretos no purchase.
			SCKPaymentLink string `json:"sckPaymentLink"`
			Offer          struct {
				Code       string `json:"code"`
				CouponCode string `json:"coupon_code"`
			} `json:"offer"`

			// Tracking opcional (algumas integrações enviam UTMs aqui).
			Tracking map[string]any `json:"tracking"`

			Price struct {
				Value         float64 `json:"value"`
				CurrencyValue string  `json:"currency_value"`
				CurrencyCode  string  `json:"currency_code"`
			} `json:"price"`
			FullPrice struct {
				Value         float64 `json:"value"`
				CurrencyValue string  `json:"currency_value"`
			} `json:"full_price"`
			OriginalOfferPrice struct {
				Value         float64 `json:"value"`
				CurrencyValue string  `json:"currency_value"`
			} `json:"original_offer_price"`
		} `json:"purchase"`
		Affiliates []struct {
			AffiliateCode string `json:"affiliate_code"`
		} `json:"affiliates"`
	} `json:"data"`
}

// Hotmart processa o webhook /webhooks/hotmart.
//
// Aceita só PURCHASE_APPROVED. Os demais (PURCHASE_REFUNDED, BILLET_PRINTED, etc)
// retornam 200 OK silenciosamente.
func (h *Webhooks) Hotmart(w http.ResponseWriter, r *http.Request) {
	var p hotmartPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if !strings.EqualFold(p.Event, "PURCHASE_APPROVED") {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: " + p.Event,
		})
		return
	}

	first, last := hotmartName(p)
	phone := util.FirstNonEmpty(p.Data.Buyer.CheckoutPhone, p.Data.Buyer.Phone.Number)

	// Valor: prioriza price > full_price > original_offer_price.
	value := p.Data.Purchase.Price.Value
	if value == 0 {
		value = p.Data.Purchase.FullPrice.Value
	}
	if value == 0 {
		value = p.Data.Purchase.OriginalOfferPrice.Value
	}

	currency := util.FirstNonEmpty(
		p.Data.Purchase.Price.CurrencyValue,
		p.Data.Purchase.FullPrice.CurrencyValue,
		p.Data.Purchase.OriginalOfferPrice.CurrencyValue,
		p.Data.Purchase.Price.CurrencyCode,
	)

	// userId pode chegar em vários campos:
	//   - origin.xcod (legacy v1)
	//   - sckPaymentLink (v2)
	//   - offer.code / offer.coupon_code (clientes que abusam desses campos)
	//   - tracking.utm_source etc (algumas integrações)
	//   - affiliate_code (uso adaptado)
	userIDCandidates := []string{
		p.Data.Purchase.SCKPaymentLink,
		p.Data.Purchase.Offer.Code,
		p.Data.Purchase.Offer.CouponCode,
	}
	for _, a := range p.Data.Affiliates {
		userIDCandidates = append(userIDCandidates, a.AffiliateCode)
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Data.Purchase.Origin),
		util.ExtractUserIDFromMap(p.Data.Purchase.Tracking),
		util.ExtractUserIDFromAny(userIDCandidates...),
	)

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "hotmart",
		ExternalID:  externalID,
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
