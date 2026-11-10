package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// caktoItem é um pedido individual da Cakto (suporta v1 objeto único e v2 array).
type caktoItem struct {
	ID         string  `json:"id"`
	Amount     float64 `json:"amount"`     // valor cobrado (com juros)
	BaseAmount float64 `json:"baseAmount"` // preço original da oferta
	Status     string  `json:"status"`
	Type       string  `json:"type"` // main, order_bump, upsell
	Customer   struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"customer"`
	Product struct {
		ID   any    `json:"id"`
		Name string `json:"name"`
	} `json:"product"`
	Offer struct {
		ID       string  `json:"id"`
		Name     string  `json:"name"`
		Price    float64 `json:"price"`
		Currency string  `json:"currency"`
	} `json:"offer"`
	FBC         string `json:"fbc"`
	FBP         string `json:"fbp"`
	SCK         string `json:"sck"`
	SRC         string `json:"src"`
	UTMSource   string `json:"utm_source"`
	UTMTerm     string `json:"utm_term"`
	UTMMedium   string `json:"utm_medium"`
	UTMContent  string `json:"utm_content"`
	UTMCampaign string `json:"utm_campaign"`
}

type caktoPayload struct {
	Event  string      `json:"event"`
	Secret string      `json:"secret"`
	Data   []caktoItem `json:"-"`
}

func (p *caktoPayload) UnmarshalJSON(data []byte) error {
	type alias struct {
		Event  string          `json:"event"`
		Secret string          `json:"secret"`
		Data   json.RawMessage `json:"data"`
	}
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.Event = raw.Event
	p.Secret = raw.Secret

	if len(raw.Data) == 0 {
		return nil
	}

	for _, b := range raw.Data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '[':
			return json.Unmarshal(raw.Data, &p.Data)
		case '{':
			var single caktoItem
			if err := json.Unmarshal(raw.Data, &single); err != nil {
				return err
			}
			p.Data = []caktoItem{single}
			return nil
		default:
		}
		break
	}
	return nil
}

// Cakto processa o webhook /webhooks/cakto.
func (h *Webhooks) Cakto(w http.ResponseWriter, r *http.Request) {
	raw, err := readWebhookBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var p caktoPayload
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ev := strings.ToLower(p.Event)
	approvedEvent := ev == "purchase_approved" || ev == "pedido_aprovado" || ev == "approved"

	if len(p.Data) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"reason":   "no data items",
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "cakto", p.Secret) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	deepUID := util.ExtractUserIDDeep(raw)
	results := make([]map[string]any, 0, len(p.Data))
	highestStatus := http.StatusOK

	pixelQuery := webhookPixelQuery(r)
	for i := range p.Data {
		item := &p.Data[i]
		body, status := h.processCaktoItem(r.Context(), item, approvedEvent, pixelQuery, accID, deepUID)
		body["index"] = i
		results = append(results, body)
		if status > highestStatus {
			highestStatus = status
		}
	}

	writeJSON(w, highestStatus, map[string]any{
		"received":  true,
		"processed": len(results),
		"event":     p.Event,
		"results":   results,
	})
}

func (h *Webhooks) processCaktoItem(ctx context.Context, item *caktoItem, approvedEvent bool, pixelQuery string, accountID int64, deepUID string) (map[string]any, int) {
	st := strings.ToLower(item.Status)
	approved := approvedEvent || st == "paid" || st == "approved"
	if !approved {
		return map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + item.Status,
		}, http.StatusOK
	}

	first, last := util.SplitFullName(strings.ToLower(item.Customer.Name))

	value := item.BaseAmount
	if value == 0 {
		value = item.Amount
	}
	if value == 0 {
		value = item.Offer.Price
	}

	offerType := strings.ToLower(strings.TrimSpace(item.Type))
	in := purchaseInput{
		GatewayName: "cakto",
		ExternalID: util.FirstNonEmpty(
			util.ExtractUserIDFromAny(
				item.UTMSource, item.UTMTerm, item.UTMContent,
				item.UTMMedium, item.UTMCampaign,
				item.SCK, item.SRC,
			),
			deepUID,
		),
		OrderID:     item.ID,
		Value:       value,
		Currency:    util.FirstNonEmpty(item.Offer.Currency, "BRL"),
		FirstName:   first,
		LastName:    last,
		Email:       item.Customer.Email,
		Phone:       item.Customer.Phone,
		ProductID:   util.FirstNonEmpty(anyString(item.Product.ID), item.Offer.ID),
		ProductName: util.FirstNonEmpty(item.Product.Name, item.Offer.Name),
		IsOrderBump: offerType == "order_bump" || offerType == "orderbump",
		IsUpsell:    offerType == "upsell",
		OfferType:   offerType,
		AccountID:   accountID,
		PixelQuery:  pixelQuery,
	}
	return h.enqueueGenericPurchase(ctx, in, fromPurchaseInput(in))
}
