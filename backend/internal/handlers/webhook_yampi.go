package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// yampiPayload reflete o JSON enviado pela Yampi nos eventos transacionais.
//
// Doc oficial: https://docs.yampi.com.br/webhooks
//
// Importante:
//   - Yampi envia HMAC-SHA256 do body no header `X-Yampi-Hmac-SHA256`.
//     Validamos quando `Webhooks.secrets.Yampi` está configurado.
//   - O event "order.paid" mapeia pra Purchase. "order.created" mapeia pra
//     AddPaymentInfo (carrinho criado, pagamento iniciado mas não pago ainda).
//   - O `userId` vem em `customer.notes` (placeholder do Yampi pra metadados),
//     `customer.metadata`, ou no UTM via `analytics.utm_source`.
type yampiPayload struct {
	Event    string `json:"event"`
	Resource struct {
		ID       int     `json:"id"`
		Number   string  `json:"number"`
		Total    float64 `json:"total"`
		Currency string  `json:"currency"`
		Customer struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Phone    struct {
				Number string `json:"number"`
			} `json:"phone"`
			Notes    string `json:"notes"`
			Metadata string `json:"metadata"`
		} `json:"customer"`
		Analytics struct {
			UTMSource string `json:"utm_source"`
		} `json:"analytics"`
	} `json:"resource"`
}

// Yampi processa o webhook /webhooks/yampi com validação HMAC opcional.
func (h *Webhooks) Yampi(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read error")
		return
	}

	// Validação HMAC quando o secret está configurado.
	// Se o secret estiver vazio, aceita sem checar (modo dev/teste).
	if h.secrets.Yampi != "" {
		got := r.Header.Get("X-Yampi-Hmac-SHA256")
		if got == "" {
			writeError(w, http.StatusUnauthorized, "missing X-Yampi-Hmac-SHA256")
			return
		}
		mac := hmac.New(sha256.New, []byte(h.secrets.Yampi))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(got), []byte(expected)) {
			writeError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
	}

	var p yampiPayload
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Resource.Customer.Name))
	utmSource := util.FirstNonEmpty(
		p.Resource.Analytics.UTMSource,
		p.Resource.Customer.Notes,
		p.Resource.Customer.Metadata,
	)

	// Mapeia event do Yampi pro nosso domínio.
	eventName := domain.EventPurchase
	if strings.Contains(p.Event, "created") || strings.Contains(p.Event, "pending") {
		eventName = domain.EventInitiateCheckout
	}

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "yampi",
		EventName:   eventName,
		ExternalID:  util.ExtractUserIDFromUTM(utmSource),
		OrderID:     p.Resource.Number,
		Value:       p.Resource.Total,
		Currency:    p.Resource.Currency,
		FirstName:   first,
		LastName:    last,
		Email:       p.Resource.Customer.Email,
		Phone:       p.Resource.Customer.Phone.Number,
	})
}
