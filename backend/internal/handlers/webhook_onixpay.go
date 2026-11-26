package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type onixPayOrderItem struct {
	ID      any `json:"id"`
	Price   any `json:"price"`
	Product struct {
		ID         any    `json:"id"`
		Name       string `json:"name"`
		ExternalID string `json:"externalId"`
	} `json:"product"`
}

type onixPayAddress struct {
	Country      string `json:"country"`
	ZipCode      string `json:"zipCode"`
	State        string `json:"state"`
	City         string `json:"city"`
	Neighborhood string `json:"neighborhood"`
	Street       string `json:"street"`
	Number       string `json:"number"`
	Complement   string `json:"complement"`
}

// onixPayPayload reflete os eventos oficiais da Onix Pay (TRANSACTION_CREATED e TRANSACTION_PAID),
// suportando tanto orderItems/trackProps na raiz quanto aninhados dentro de transaction.
type onixPayPayload struct {
	Event       string `json:"event"`
	Token       string `json:"token"`
	Secret      string `json:"secret"`
	OfferCode   string `json:"offerCode"`
	CheckoutURL string `json:"checkoutUrl"`
	OrderID     string `json:"orderId"`
	OrderURL    string `json:"orderUrl"`
	Client      struct {
		ID      string          `json:"id"`
		Name    string          `json:"name"`
		Email   string          `json:"email"`
		Phone   string          `json:"phone"`
		CPF     string          `json:"cpf"`
		CNPJ    string          `json:"cnpj"`
		Address *onixPayAddress `json:"address"`
	} `json:"client"`
	Transaction struct {
		ID               string             `json:"id"`
		Identifier       string             `json:"identifier"`
		Status           string             `json:"status"`
		PaymentMethod    string             `json:"paymentMethod"`
		OriginalAmount   any                `json:"originalAmount"`
		Amount           any                `json:"amount"`
		CommissionAmount any                `json:"commissionAmount"`
		OriginalCurrency string             `json:"originalCurrency"`
		Currency         string             `json:"currency"`
		OrderItems       []onixPayOrderItem `json:"orderItems"`
		TrackProps       map[string]any     `json:"trackProps"`
	} `json:"transaction"`
	OrderItems []onixPayOrderItem `json:"orderItems"`
	TrackProps map[string]any     `json:"trackProps"`
}

// OnixPay processa o webhook /webhooks/onixpay.
// TRANSACTION_CREATED dispara AddPaymentInfo (e hidrata fbc/fbp/UTMs/IP/UA do lead) e TRANSACTION_PAID dispara Purchase.
func (h *Webhooks) OnixPay(w http.ResponseWriter, r *http.Request) {
	raw, err := readWebhookBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	itemsRaw, err := unwrapOnixPayPayloads(raw)
	if err != nil || len(itemsRaw) == 0 {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	accID := ExtractAccountID(r)
	pixelQuery := webhookPixelQuery(r)

	if len(itemsRaw) == 1 {
		body, status := h.processSingleOnixPay(r, itemsRaw[0], accID, pixelQuery)
		writeJSON(w, status, body)
		return
	}

	results := make([]map[string]any, 0, len(itemsRaw))
	highest := http.StatusOK
	for i, itemBytes := range itemsRaw {
		body, st := h.processSingleOnixPay(r, itemBytes, accID, pixelQuery)
		body["index"] = i
		results = append(results, body)
		if st > highest {
			highest = st
		}
	}
	writeJSON(w, highest, map[string]any{
		"received":  true,
		"processed": len(results),
		"results":   results,
	})
}

func (h *Webhooks) processSingleOnixPay(r *http.Request, itemRaw []byte, accID int64, pixelQuery string) (map[string]any, int) {
	var p onixPayPayload
	dec := json.NewDecoder(bytes.NewReader(itemRaw))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
		return map[string]any{"error": "invalid json"}, http.StatusBadRequest
	}

	eventName, ok := resolveOnixPayEventName(p.Event, p.Transaction.Status)
	if !ok {
		return map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: event=" + p.Event + " status=" + p.Transaction.Status,
		}, http.StatusOK
	}

	if !h.ValidatePlatformToken(r.Context(), r, accID, "onixpay", p.Token, p.Secret) {
		return map[string]any{"reason": "invalid platform token"}, http.StatusUnauthorized
	}

	in := buildOnixPayPurchaseInput(p, eventName, accID, pixelQuery, itemRaw)
	return h.enqueueGenericPurchase(r.Context(), in, fromPurchaseInput(in))
}

func resolveOnixPayEventName(rawEvent, rawStatus string) (string, bool) {
	ev := strings.ToUpper(strings.TrimSpace(rawEvent))
	st := strings.ToUpper(strings.TrimSpace(rawStatus))

	switch ev {
	case "TRANSACTION_PAID", "ORDER_PAID", "PURCHASE_APPROVED", "SALE_APPROVED":
		return domain.EventPurchase, true
	case "TRANSACTION_CREATED", "ORDER_CREATED", "PIX_GENERATED", "WAITING_PAYMENT":
		return domain.EventAddPaymentInfo, true
	default:
		if rev := detectRefundOrChargeback(ev, st); rev != "" {
			return rev, true
		}
		if ev == "" {
			if st == "COMPLETED" || st == "PAID" || st == "APPROVED" {
				return domain.EventPurchase, true
			}
			if st == "PENDING" || st == "WAITING_PAYMENT" {
				return domain.EventAddPaymentInfo, true
			}
		}
	}
	return "", false
}
