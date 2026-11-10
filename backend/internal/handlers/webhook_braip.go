package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// braipPayload suporta tanto o Postback JSON quanto x-www-form-urlencoded da Braip.
type braipPayload struct {
	BasicAuth       string         `json:"basic_authentication"`
	Type            string         `json:"type"`
	TransKey        string         `json:"trans_key"`
	TransCode       string         `json:"trans_code"`
	TransID         any            `json:"trans_id"`
	TransStatus     any            `json:"trans_status"`
	TransStatusCode any            `json:"trans_status_code"`
	TransStatusName string         `json:"trans_status_name"`
	TransValue      any            `json:"trans_value"`
	TransTotalValue any            `json:"trans_total_value"`
	TransCurrency   string         `json:"trans_currency"`
	ClientName      string         `json:"client_name"`
	ClientEmail     string         `json:"client_email"`
	ClientCel       string         `json:"client_cel"`
	ClientPhone     string         `json:"client_phone"`
	ProdKey         string         `json:"prod_key"`
	ProductCode     string         `json:"product_code"`
	ProdID          any            `json:"prod_id"`
	ProdName        string         `json:"prod_name"`
	ProductName     string         `json:"product_name"`
	ParentSale      string         `json:"parent_sale"`
	Meta            map[string]any `json:"meta"`
	UTMSource       string         `json:"utm_source"`
	UTMCampaign     string         `json:"utm_campaign"`
	UTMMedium       string         `json:"utm_medium"`
	UTMContent      string         `json:"utm_content"`
	UTMTerm         string         `json:"utm_term"`
	SRC             string         `json:"src"`
	SCK             string         `json:"sck"`
}

// Braip processa o postback/webhook /webhooks/braip.
func (h *Webhooks) Braip(w http.ResponseWriter, r *http.Request) {
	var p braipPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	stText := strings.ToLower(strings.TrimSpace(util.FirstNonEmpty(anyString(p.TransStatus), p.TransStatusName)))
	stCode := strings.TrimSpace(anyString(p.TransStatusCode))

	approved := stCode == "2" ||
		stText == "2" ||
		strings.Contains(stText, "aprovad") ||
		stText == "pago" ||
		stText == "paid"

	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + util.FirstNonEmpty(stText, stCode),
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "braip", p.BasicAuth) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.ClientName))
	phone := util.FirstNonEmpty(p.ClientCel, p.ClientPhone)

	value := util.ParseMoneyAny(p.TransValue)
	if value == 0 {
		value = util.ParseMoneyAny(p.TransTotalValue)
	}

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Meta),
		util.ExtractUserIDFromAny(p.UTMSource, p.UTMTerm, p.UTMContent, p.UTMMedium, p.UTMCampaign, p.SRC, p.SCK),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "braip",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(p.TransKey, p.TransCode, anyString(p.TransID)),
		Value:       value,
		Currency:    util.FirstNonEmpty(p.TransCurrency, "BRL"),
		FirstName:   first,
		LastName:    last,
		Email:       p.ClientEmail,
		Phone:       phone,
		ProductID:   util.FirstNonEmpty(p.ProdKey, p.ProductCode, anyString(p.ProdID)),
		ProductName: util.FirstNonEmpty(p.ProdName, p.ProductName),
		IsOrderBump: p.ParentSale != "",
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

