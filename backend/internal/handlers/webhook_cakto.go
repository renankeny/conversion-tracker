package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// caktoItem é um pedido individual da Cakto.
//
// Formatos aceitos pelo wrapper caktoPayload (v1 single-object e
// v2 array — ambos vistos em produção):
//
//  v1 (single):
//	{
//	  "event": "purchase_approved",
//	  "data": { "id": "...", "customer": {...}, ... }
//	}
//
//  v2 (array):
//	{
//	  "event": "purchase_approved",
//	  "data": [
//	    { "id": "...", "customer": {...}, ... },
//	    ... possivelmente mais items
//	  ]
//	}
type caktoItem struct {
	ID         string  `json:"id"`
	Amount     float64 `json:"amount"`     // valor cobrado (com juros)
	BaseAmount float64 `json:"baseAmount"` // preço original da oferta
	Status     string  `json:"status"`
	Customer   struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"customer"`
	Offer struct {
		ID       string  `json:"id"`
		Name     string  `json:"name"`
		Price    float64 `json:"price"`
		Currency string  `json:"currency"`
	} `json:"offer"`
	FBC         string `json:"fbc"`
	FBP         string `json:"fbp"`
	SCK         string `json:"sck"`
	UTMSource   string `json:"utm_source"`
	UTMTerm     string `json:"utm_term"`
	UTMMedium   string `json:"utm_medium"`
	UTMContent  string `json:"utm_content"`
	UTMCampaign string `json:"utm_campaign"`
}

// caktoPayload aceita data como objeto OU array. O Decode customizado
// abaixo normaliza pra slice.
type caktoPayload struct {
	Event  string      `json:"event"`
	Secret string      `json:"secret"`
	Data   []caktoItem `json:"-"` // populado em UnmarshalJSON
}

// UnmarshalJSON aceita {data: {}} (single) e {data: [...]} (array).
func (p *caktoPayload) UnmarshalJSON(data []byte) error {
	// Decoda primeiro envelope com `data` como RawMessage pra inspecionar
	// o tipo antes de decidir.
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

	// Inspeciona o primeiro byte significativo: '[' = array, '{' = objeto.
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
			break
		}
		break
	}

	// data é null ou tipo inesperado — trata como vazio.
	return nil
}

// Cakto processa o webhook /webhooks/cakto.
//
// Aceita o evento `purchase_approved` ou status `paid`. Os demais são
// ignorados retornando 200 OK pra evitar retry desnecessário do gateway.
//
// Suporta `data` tanto como objeto único quanto como array de pedidos
// (formato batch que a Cakto introduziu em meados de 2026).
func (h *Webhooks) Cakto(w http.ResponseWriter, r *http.Request) {
	var p caktoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Filtra evento global (vale pra todos os items do batch).
	ev := strings.ToLower(p.Event)
	approvedEvent := ev == "purchase_approved" || ev == "pedido_aprovado" || ev == "approved"

	if len(p.Data) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"reason":   "no data items",
		})
		return
	}

	results := make([]map[string]any, 0, len(p.Data))
	highestStatus := http.StatusOK

	for i := range p.Data {
		item := &p.Data[i]
		body, status := h.processCaktoItem(r.Context(), item, approvedEvent)
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

// processCaktoItem aplica o filtro de status e enfileira um único item.
// Retorna body+statusCode pra agregação.
func (h *Webhooks) processCaktoItem(ctx context.Context, item *caktoItem, approvedEvent bool) (map[string]any, int) {
	st := strings.ToLower(item.Status)
	approved := approvedEvent || st == "paid"
	if !approved {
		return map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + item.Status,
		}, http.StatusOK
	}

	first, last := util.SplitFullName(strings.ToLower(item.Customer.Name))

	// Valor: prioriza baseAmount (preço original) sobre amount (com juros).
	value := item.BaseAmount
	if value == 0 {
		value = item.Amount
	}
	if value == 0 {
		value = item.Offer.Price
	}

	currency := item.Offer.Currency
	if currency == "" {
		currency = "BRL"
	}

	in := purchaseInput{
		GatewayName: "cakto",
		ExternalID: util.ExtractUserIDFromAny(
			item.UTMSource, item.UTMTerm, item.UTMContent,
			item.UTMMedium, item.UTMCampaign,
			item.SCK,
		),
		OrderID:   item.ID,
		Value:     value,
		Currency:  currency,
		FirstName: first,
		LastName:  last,
		Email:     item.Customer.Email,
		Phone:     item.Customer.Phone,
	}
	return h.enqueueGenericPurchase(ctx, in)
}
