package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// monetizzePayload reflete o JSON da Monetizze.
//
// Monetizze envia eventos como "venda" com `status_descricao` (ex: "Aprovada",
// "Cancelada", "Reembolsada"). Tratamos só "Aprovada" como Purchase.
//
// O `userId` aparece em `venda.utmSource` ou em `cliente.observacoes`.
type monetizzePayload struct {
	Venda struct {
		Codigo            string  `json:"codigo"`
		ValorReal         float64 `json:"valor_real"`     // valor líquido em reais
		Valor             float64 `json:"valor"`          // valor bruto em reais
		Moeda             string  `json:"moeda"`
		StatusDescricao   string  `json:"status_descricao"`
		UTMSource         string  `json:"utmSource"`
		UTMSourceLegacy   string  `json:"utm_source"`
	} `json:"venda"`
	Cliente struct {
		Nome     string `json:"nome"`
		Email    string `json:"email"`
		Telefone string `json:"telefone"`
		Celular  string `json:"celular"`
		Observ   string `json:"observacoes"`
	} `json:"cliente"`
}

// Monetizze processa o webhook /webhooks/monetizze.
func (h *Webhooks) Monetizze(w http.ResponseWriter, r *http.Request) {
	var p monetizzePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	st := strings.ToLower(p.Venda.StatusDescricao)
	if !strings.Contains(st, "aprovada") && !strings.Contains(st, "approved") && !strings.Contains(st, "paid") {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + p.Venda.StatusDescricao,
		})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Cliente.Nome))
	phone := util.FirstNonEmpty(p.Cliente.Celular, p.Cliente.Telefone)
	utmSource := util.FirstNonEmpty(p.Venda.UTMSource, p.Venda.UTMSourceLegacy, p.Cliente.Observ)
	value := p.Venda.Valor
	if value == 0 {
		value = p.Venda.ValorReal
	}

	h.processGenericPurchase(r.Context(), w, purchaseInput{
		GatewayName: "monetizze",
		ExternalID:  util.ExtractUserIDFromUTM(utmSource),
		OrderID:     p.Venda.Codigo,
		Value:       value,
		Currency:    p.Venda.Moeda,
		FirstName:   first,
		LastName:    last,
		Email:       p.Cliente.Email,
		Phone:       phone,
	})
}
