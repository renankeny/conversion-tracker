package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type yampiPayload struct {
	Event    string `json:"event"`
	Resource struct {
		ID              any     `json:"id"`
		Number          any     `json:"number"`
		ValueTotal      float64 `json:"value_total"`
		BuyerValueTotal float64 `json:"buyer_value_total"`
		UTMSource       string  `json:"utm_source"`
		UTMMedium       string  `json:"utm_medium"`
		UTMCampaign     string  `json:"utm_campaign"`
		UTMContent      string  `json:"utm_content"`
		UTMTerm         string  `json:"utm_term"`
		Customer        struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			CPF   string `json:"cpf"`
			Phone struct {
				FullNumber string `json:"full_number"`
			} `json:"phone"`
		} `json:"customer"`
		Items struct {
			Data []struct {
				ProductID any    `json:"product_id"`
				ItemSKU   string `json:"item_sku"`
				SKU       struct {
					Data struct {
						Title string `json:"title"`
					} `json:"data"`
				} `json:"sku"`
			} `json:"data"`
		} `json:"items"`
	} `json:"resource"`
}

// Yampi processa o webhook /webhooks/yampi.
// Suporta validação X-Yampi-Hmac-SHA256 tanto em Base64 (padrão oficial Yampi) quanto em Hex.
func (h *Webhooks) Yampi(w http.ResponseWriter, r *http.Request) {
	raw, err := readWebhookBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	var p yampiPayload
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if !strings.EqualFold(p.Event, "order.paid") {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: " + p.Event,
		})
		return
	}

	accID := ExtractAccountID(r)
	hmacHeader := r.Header.Get("X-Yampi-Hmac-SHA256")
	if !h.validateHMACOrToken(r.Context(), r, accID, "yampi", raw, hmacHeader, "") {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid hmac signature"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Resource.Customer.Name))

	value := p.Resource.ValueTotal
	if value == 0 {
		value = p.Resource.BuyerValueTotal
	}

	utmMap := map[string]any{
		"utm_source":   p.Resource.UTMSource,
		"utm_medium":   p.Resource.UTMMedium,
		"utm_campaign": p.Resource.UTMCampaign,
		"utm_content":  p.Resource.UTMContent,
		"utm_term":     p.Resource.UTMTerm,
	}
	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(utmMap),
		util.ExtractUserIDDeep(raw),
	)

	orderID := anyString(p.Resource.ID)
	if orderID == "" || orderID == "0" {
		orderID = anyString(p.Resource.Number)
	}

	var productID, productName string
	if len(p.Resource.Items.Data) > 0 {
		firstItem := p.Resource.Items.Data[0]
		productID = util.FirstNonEmpty(anyString(firstItem.ProductID), firstItem.ItemSKU)
		productName = firstItem.SKU.Data.Title
	}

	in := purchaseInput{
		GatewayName: "yampi",
		ExternalID:  externalID,
		OrderID:     orderID,
		Value:       value,
		Currency:    "BRL",
		FirstName:   first,
		LastName:    last,
		Email:       p.Resource.Customer.Email,
		Phone:       p.Resource.Customer.Phone.FullNumber,
		ProductID:   productID,
		ProductName: productName,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}

	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}
