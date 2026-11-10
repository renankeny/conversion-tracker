package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type pagtrustPayload struct {
	ID     string `json:"id"`
	Event  string `json:"event"`
	Hottok string `json:"hottok"`
	Token  string `json:"token"`
	Data   struct {
		Product struct {
			ID   any    `json:"id"`
			Name string `json:"name"`
		} `json:"product"`
		Buyer struct {
			Name              string `json:"name"`
			Email             string `json:"email"`
			CheckoutPhone     string `json:"checkout_phone"`
			CheckoutFullPhone string `json:"checkout_full_phone"`
		} `json:"buyer"`
		Purchase struct {
			Transaction string `json:"transaction"`
			Price       struct {
				Price         float64 `json:"price"`
				Value         float64 `json:"value"`
				CurrencyValue string  `json:"currency_value"`
			} `json:"price"`
			FullPrice struct {
				Value         float64 `json:"value"`
				CurrencyValue string  `json:"currency_value"`
			} `json:"full_price"`
			OriginalOfferPrice struct {
				Value         float64 `json:"value"`
				CurrencyValue string  `json:"currency_value"`
			} `json:"original_offer_price"`
			Origin struct {
				Xcod        string `json:"xcod"`
				Src         string `json:"src"`
				Sck         string `json:"sck"`
				UTMCampaign string `json:"utmcampaign"`
				UTMMedium   string `json:"utmmedium"`
				UTMSource   string `json:"utmsource"`
				Content     string `json:"content"`
				Term        string `json:"term"`
			} `json:"origin"`
		} `json:"purchase"`
	} `json:"data"`
}

// PagTrust processa o webhook /webhooks/pagtrust.
func (h *Webhooks) PagTrust(w http.ResponseWriter, r *http.Request) {
	raw, err := readWebhookBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var p pagtrustPayload
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
	if !h.ValidatePlatformToken(r.Context(), r, accID, "pagtrust", p.Hottok, p.Token) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(p.Data.Buyer.Name)
	phone := util.FirstNonEmpty(p.Data.Buyer.CheckoutFullPhone, p.Data.Buyer.CheckoutPhone)

	value := p.Data.Purchase.Price.Price
	if value == 0 {
		value = p.Data.Purchase.Price.Value
	}
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
	)

	o := p.Data.Purchase.Origin
	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromAny(o.Xcod, o.Src, o.Sck, o.UTMSource, o.UTMCampaign, o.UTMMedium, o.Content, o.Term),
		util.ExtractUserIDDeep(raw),
	)

	in := purchaseInput{
		GatewayName: "pagtrust",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(p.Data.Purchase.Transaction, p.ID),
		Value:       value,
		Currency:    currency,
		FirstName:   first,
		LastName:    last,
		Email:       p.Data.Buyer.Email,
		Phone:       phone,
		ProductID:   anyString(p.Data.Product.ID),
		ProductName: p.Data.Product.Name,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}
