package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// kirvanoPayload reflete o JSON REAL da Kirvano.
//
// Estrutura observada em produção (junho/2026):
//
//	{
//	  "event":  "SALE_APPROVED",        // pode ser SALE_APPROVED, SALE_CHARGEBACK...
//	  "type":   "ONE_TIME",
//	  "status": "APPROVED",
//	  "sale_id":     "ABC123",
//	  "checkout_id": "XYZ",
//	  "total_price": "R$ 329,65",       // string formatada
//	  "payment_method": "PIX",
//	  "fiscal": {
//	    "total_value":    329.65,       // valor pago (cobrado)
//	    "original_value": 347,          // preço da oferta antes de descontos
//	    "net_value":      329.65,
//	    "commission":     302.96,
//	    "fee":            26.69,
//	    ...
//	  },
//	  "customer": {
//	    "name":         "...",
//	    "email":        "...",
//	    "phone_number": "5584...",
//	    "document":     "...",
//	    "address":      { ... }
//	  },
//	  "products": [{ "id", "name", "price": "R$ 347,00", "offer_id", ... }],
//	  "utm": {
//	    "utm_source": "...", "utm_term": "...", "utm_medium": "...",
//	    "utm_content": "...", "utm_campaign": "...", "src": "..."
//	  },
//	  "cookies": {},                    // pode trazer fbc/fbp no futuro
//	  ...
//	}
//
// Tratamos só event=SALE_APPROVED OU status=APPROVED.
type kirvanoPayload struct {
	Event      string `json:"event"`
	Status     string `json:"status"`
	SaleID     string `json:"sale_id"`
	TotalPrice string `json:"total_price"` // sempre string "R$ 329,65"
	Fiscal     struct {
		TotalValue    float64 `json:"total_value"`
		OriginalValue float64 `json:"original_value"`
		NetValue      float64 `json:"net_value"`
	} `json:"fiscal"`
	Customer struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phone_number"`
		Document    string `json:"document"`
	} `json:"customer"`
	UTM map[string]any `json:"utm"`
}

// Kirvano processa o webhook /webhooks/kirvano.
func (h *Webhooks) Kirvano(w http.ResponseWriter, r *http.Request) {
	var p kirvanoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Filtra só vendas aprovadas. Aceita event ou status.
	ev := strings.ToUpper(p.Event)
	st := strings.ToUpper(p.Status)
	approved := ev == "SALE_APPROVED" || st == "APPROVED" || ev == "PURCHASE_APPROVED"
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: event=" + p.Event + " status=" + p.Status,
		})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.Name))

	// Valor: prioriza fiscal.total_value (numérico, valor pago).
	// Fallback: parse da string total_price ("R$ 329,65"). Original_value é
	// usado como último recurso (preço de tabela).
	value := p.Fiscal.TotalValue
	if value == 0 {
		value = util.ParseMoneyAny(p.TotalPrice)
	}
	if value == 0 {
		value = p.Fiscal.OriginalValue
	}

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "kirvano",
		ExternalID:  util.ExtractUserIDFromMap(p.UTM),
		OrderID:     p.SaleID,
		Value:       value,
		Currency:    "BRL",
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       p.Customer.PhoneNumber,
	})
}
