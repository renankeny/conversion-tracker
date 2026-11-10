package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// hotmartPayload reflete o JSON da Hotmart, suportando v1.x (legacy com
// `origin.xcod`) e v2.0 (atual, com `sckPaymentLink`, `offer.code`, `product`, `order_bump`).
type hotmartPayload struct {
	ID     string `json:"id"`
	Event  string `json:"event"`
	Hottok string `json:"hottok"`
	Data   struct {
		Product struct {
			ID    any    `json:"id"`
			Ucode string `json:"ucode"`
			Name  string `json:"name"`
		} `json:"product"`
		Buyer struct {
			Name              string `json:"name"`
			FirstName         string `json:"first_name"`
			LastName          string `json:"last_name"`
			Email             string `json:"email"`
			CheckoutPhone     string `json:"checkout_phone"`
			CheckoutPhoneCode string `json:"checkout_phone_code"`
			Document          string `json:"document"`
			Phone             struct {
				LocalCode string `json:"local_code"`
				Number    string `json:"number"`
			} `json:"phone"`
		} `json:"buyer"`
		Purchase struct {
			Transaction    string         `json:"transaction"`
			Origin         map[string]any `json:"origin"`
			SCKPaymentLink string         `json:"sckPaymentLink"`
			Offer          struct {
				Code       string `json:"code"`
				CouponCode string `json:"coupon_code"`
			} `json:"offer"`
			OrderBump struct {
				IsOrderBump bool `json:"is_order_bump"`
			} `json:"order_bump"`
			Tracking map[string]any `json:"tracking"`
			Price    struct {
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
// Aceita PURCHASE_APPROVED (aprovação imediata da venda).
func (h *Webhooks) Hotmart(w http.ResponseWriter, r *http.Request) {
	raw, err := readWebhookBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var p hotmartPayload
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
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

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "hotmart", p.Hottok) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := hotmartName(p)
	phone := util.FirstNonEmpty(
		joinPhoneDDD("", p.Data.Buyer.CheckoutPhoneCode, p.Data.Buyer.CheckoutPhone),
		joinPhoneDDD("", p.Data.Buyer.Phone.LocalCode, p.Data.Buyer.Phone.Number),
	)

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
		util.ExtractUserIDDeep(raw),
	)

	productID := util.FirstNonEmpty(anyString(p.Data.Product.ID), p.Data.Product.Ucode)
	in := purchaseInput{
		GatewayName: "hotmart",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(p.Data.Purchase.Transaction, p.ID),
		Value:       value,
		Currency:    currency,
		FirstName:   first,
		LastName:    last,
		Email:       p.Data.Buyer.Email,
		Phone:       phone,
		ProductID:   productID,
		ProductName: p.Data.Product.Name,
		IsOrderBump: p.Data.Purchase.OrderBump.IsOrderBump,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

func hotmartName(p hotmartPayload) (string, string) {
	first := strings.ToLower(p.Data.Buyer.FirstName)
	last := strings.ToLower(p.Data.Buyer.LastName)
	if first != "" || last != "" {
		return first, last
	}
	return util.SplitFullName(strings.ToLower(p.Data.Buyer.Name))
}
