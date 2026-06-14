package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// rkCursosPayload reflete o JSON REAL da RKCursos.
//
// Estrutura observada em produção (junho/2026):
//
//	{
//	  "event": "checkout.purchased",
//	  "timestamp": "2026-06-11T15:35:07.599Z",
//	  "data": {
//	    "order_id":   "uuid",
//	    "user_id":    "uuid",          // user da plataforma RKCursos, NÃO o tracked_user
//	    "course_id":  "uuid",
//	    "full_name":  "Nome Completo",
//	    "email":      "...",
//	    "phone":      "5585...",       // já com 55
//	    "cpf":        "00000000000",
//	    "amount":     347,             // valor cobrado
//	    "base_amount": 347,            // preço original da oferta
//	    "gateway":    "pagouai",
//	    "payment_method": "credit_card",
//	    "installments": 2,
//	    "discount_amount": 0,
//	    "coupon_code": null,
//	    "paid_at":    "...",
//	    "utm_source": "...", "utm_term": null, "utm_medium": null,
//	    "utm_content": null, "utm_campaign": null,
//	    "sck": null, "src": null,
//	    ...
//	  }
//	}
//
// O legado PHP ignorava utm_source e fazia lookup só por phone/email. A
// versão atual aceita TAMBÉM utm_source (e variações) como fonte do userId,
// caindo no fallback de phone+email se vier vazio.
type rkCursosPayload struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Data      struct {
		OrderID    string  `json:"order_id"`
		UserID     string  `json:"user_id"` // não usado (é id interno da RKCursos)
		CourseID   string  `json:"course_id"`
		FullName   string  `json:"full_name"`
		Email      string  `json:"email"`
		Phone      string  `json:"phone"`
		CPF        string  `json:"cpf"`
		Amount     float64 `json:"amount"`
		BaseAmount float64 `json:"base_amount"`

		// Tracking — pode ou não vir preenchido
		UTMSource   string `json:"utm_source"`
		UTMTerm     string `json:"utm_term"`
		UTMMedium   string `json:"utm_medium"`
		UTMContent  string `json:"utm_content"`
		UTMCampaign string `json:"utm_campaign"`
		SCK         string `json:"sck"`
		SRC         string `json:"src"`
	} `json:"data"`
}

// RkCursos processa o webhook /webhooks/rkcursos.
//
// Aceita o evento `checkout.purchased`. Os demais são ignorados.
func (h *Webhooks) RkCursos(w http.ResponseWriter, r *http.Request) {
	var p rkCursosPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if p.Event != "checkout.purchased" {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: " + p.Event,
		})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Data.FullName))

	// Valor: prioriza base_amount sobre amount; fallback no contrário.
	value := p.Data.BaseAmount
	if value == 0 {
		value = p.Data.Amount
	}

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "rkcursos",
		ExternalID: util.ExtractUserIDFromAny(
			p.Data.UTMSource, p.Data.UTMTerm, p.Data.UTMContent,
			p.Data.UTMMedium, p.Data.UTMCampaign,
			p.Data.SCK, p.Data.SRC,
		),
		OrderID:   p.Data.OrderID,
		Value:     value,
		Currency:  "BRL",
		FirstName: first,
		LastName:  last,
		Email:     p.Data.Email,
		Phone:     p.Data.Phone,
		// Quando o userId não vier nas UTMs (caso default da RKCursos),
		// caímos no fallback por phone (com variantes 55) e por email.
		AlsoSearchByEmail: true,
	})
}
