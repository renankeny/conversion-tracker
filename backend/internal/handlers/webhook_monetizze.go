package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// monetizzePayload suporta tanto JSON quanto application/x-www-form-urlencoded
// (venda[codigo], venda[status], comprador[email], produto[nome], chave_unica).
type monetizzePayload struct {
	ChaveUnica   string `json:"chave_unica"`
	TipoPostback struct {
		Codigo any    `json:"codigo"`
		Nome   string `json:"nome"`
	} `json:"tipoPostback"`
	Venda struct {
		Codigo        any    `json:"codigo"`
		Status        string `json:"status"`
		CodigoStatus  any    `json:"codigo_status"`
		Valor         any    `json:"valor"`
		ValorRecebido any    `json:"valorRecebido"`
		SRC           string `json:"src"`
		SCK           string `json:"sck"`
		UTMSource     string `json:"utm_source"`
		UTMMedium     string `json:"utm_medium"`
		UTMCampaign   string `json:"utm_campaign"`
		UTMContent    string `json:"utm_content"`
		UTMTerm       string `json:"utm_term"`
	} `json:"venda"`
	Comprador struct {
		Nome     string `json:"nome"`
		Email    string `json:"email"`
		Telefone string `json:"telefone"`
		Celular  string `json:"celular"`
	} `json:"comprador"`
	Produto struct {
		Codigo any    `json:"codigo"`
		Chave  string `json:"chave"`
		Nome   string `json:"nome"`
	} `json:"produto"`
	UTMSource string `json:"utm_source"`
	SRC       string `json:"src"`
}

// Monetizze processa o postback/webhook /webhooks/monetizze.
func (h *Webhooks) Monetizze(w http.ResponseWriter, r *http.Request) {
	var p monetizzePayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	st := strings.ToLower(strings.TrimSpace(p.Venda.Status))
	stCode := anyString(p.Venda.CodigoStatus)
	postbackCode := anyString(p.TipoPostback.Codigo)

	approved := st == "finalizada" ||
		st == "completa" ||
		st == "aprovada" ||
		st == "pago" ||
		st == "paid" ||
		stCode == "2" ||
		stCode == "6" ||
		postbackCode == "2"

	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + util.FirstNonEmpty(p.Venda.Status, stCode),
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "monetizze", p.ChaveUnica) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Comprador.Nome))
	phone := util.FirstNonEmpty(p.Comprador.Celular, p.Comprador.Telefone)

	value := util.ParseMoneyAny(p.Venda.Valor)
	if value == 0 {
		value = util.ParseMoneyAny(p.Venda.ValorRecebido)
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromAny(
			p.Venda.UTMSource, p.Venda.UTMTerm, p.Venda.UTMContent,
			p.Venda.UTMMedium, p.Venda.UTMCampaign,
			p.Venda.SRC, p.Venda.SCK,
			p.UTMSource, p.SRC,
		),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "monetizze",
		ExternalID:  externalID,
		OrderID:     anyString(p.Venda.Codigo),
		Value:       value,
		Currency:    "BRL",
		FirstName:   first,
		LastName:    last,
		Email:       p.Comprador.Email,
		Phone:       phone,
		ProductID:   util.FirstNonEmpty(anyString(p.Produto.Codigo), p.Produto.Chave),
		ProductName: p.Produto.Nome,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

