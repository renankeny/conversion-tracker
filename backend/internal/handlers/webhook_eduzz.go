package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// eduzzPayload suporta tanto o Developer Hub v2 (myeduzz.invoice_paid / sun / nutror)
// quanto o Postback clássico v1 (trans_status=3, JSON ou x-www-form-urlencoded).
type eduzzPayload struct {
	ID    string `json:"id"`
	Event string `json:"event"`

	// Campos v2 (Developer Hub)
	Data struct {
		ID     any    `json:"id"`
		Status string `json:"status"`
		Price  struct {
			Currency string `json:"currency"`
			Value    any    `json:"value"`
		} `json:"price"`
		Buyer struct {
			Name              string `json:"name"`
			Email             string `json:"email"`
			CellphoneNullable string `json:"cellphoneNullable"`
			PhoneNullable     string `json:"phoneNullable"`
			Cellphone         string `json:"cellphone"`
			Phone             string `json:"phone"`
		} `json:"buyer"`
		Producer struct {
			OriginSecret string `json:"originSecret"`
		} `json:"producer"`
		Offer struct {
			NameNullable string `json:"nameNullable"`
			Name         string `json:"name"`
		} `json:"offer"`
		Items []struct {
			ProductID any    `json:"productId"`
			Name      string `json:"name"`
		} `json:"items"`
		UTMNullable map[string]any `json:"utmNullable"`
		UTM         map[string]any `json:"utm"`
	} `json:"data"`

	// Campos v1 (Postback clássico)
	TransCod          any    `json:"trans_cod"`
	TransStatus       any    `json:"trans_status"`
	TransPaid         any    `json:"trans_paid"`
	TransValue        any    `json:"trans_value"`
	TransCurrency     string `json:"trans_currency"`
	CusName           string `json:"cus_name"`
	CusEmail          string `json:"cus_email"`
	CusCel            string `json:"cus_cel"`
	CusTel            string `json:"cus_tel"`
	ProCod            any    `json:"pro_cod"`
	ProName           string `json:"pro_name"`
	TrackerUTMSource  string `json:"tracker_utm_source"`
	TrackerUTMCamp    string `json:"tracker_utm_campaign"`
	TrackerUTMMedium  string `json:"tracker_utm_medium"`
	TrackerUTMContent string `json:"tracker_utm_content"`
	UTMSource         string `json:"utm_source"`
	SCK               string `json:"sck"`
	SRC               string `json:"src"`
	Origin            string `json:"origin"`
	EduzzToken        string `json:"eduzz_token"`
	APIKey            string `json:"api_key"`
}

// Eduzz processa o webhook /webhooks/eduzz.
func (h *Webhooks) Eduzz(w http.ResponseWriter, r *http.Request) {
	rawRaw, _ := readWebhookBody(r)
	var p eduzzPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	ev := strings.ToLower(strings.TrimSpace(p.Event))
	v2Status := strings.ToLower(strings.TrimSpace(p.Data.Status))
	v1Status := strings.ToLower(anyString(p.TransStatus))

	approved := ev == "myeduzz.invoice_paid" ||
		ev == "invoice_paid" ||
		ev == "sun.invoice_paid" ||
		v2Status == "paid" ||
		v1Status == "3" ||
		v1Status == "paid"

	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: " + util.FirstNonEmpty(p.Event, p.Data.Status, v1Status),
		})
		return
	}

	accID := ExtractAccountID(r)
	hmacSig := util.FirstNonEmpty(r.Header.Get("X-Signature"), r.Header.Get("x-signature"))
	if !h.validateHMACOrToken(
		r.Context(), r, accID, "eduzz", rawRaw, hmacSig, "",
		p.Data.Producer.OriginSecret, p.EduzzToken, p.APIKey, p.Origin,
	) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	fullName := util.FirstNonEmpty(p.Data.Buyer.Name, p.CusName)
	first, last := util.SplitFullName(strings.ToLower(fullName))
	email := util.FirstNonEmpty(p.Data.Buyer.Email, p.CusEmail)
	phone := util.FirstNonEmpty(
		p.Data.Buyer.CellphoneNullable, p.Data.Buyer.PhoneNullable,
		p.Data.Buyer.Cellphone, p.Data.Buyer.Phone,
		p.CusCel, p.CusTel,
	)

	value := util.ParseMoneyAny(p.Data.Price.Value)
	if value == 0 {
		value = util.ParseMoneyAny(p.TransPaid)
	}
	if value == 0 {
		value = util.ParseMoneyAny(p.TransValue)
	}

	var productID, productName string
	if len(p.Data.Items) > 0 {
		productID = anyString(p.Data.Items[0].ProductID)
		productName = p.Data.Items[0].Name
	}
	productID = util.FirstNonEmpty(productID, anyString(p.ProCod))
	productName = util.FirstNonEmpty(productName, p.Data.Offer.NameNullable, p.Data.Offer.Name, p.ProName)

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Data.UTMNullable),
		util.ExtractUserIDFromMap(p.Data.UTM),
		util.ExtractUserIDFromAny(p.TrackerUTMSource, p.TrackerUTMCamp, p.TrackerUTMMedium, p.TrackerUTMContent, p.UTMSource, p.SCK, p.SRC),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "eduzz",
		ExternalID:  externalID,
		OrderID:     util.FirstNonEmpty(anyString(p.Data.ID), anyString(p.TransCod), p.ID),
		Value:       value,
		Currency:    util.FirstNonEmpty(p.Data.Price.Currency, p.TransCurrency, "BRL"),
		FirstName:   first,
		LastName:    last,
		Email:       email,
		Phone:       phone,
		ProductID:   productID,
		ProductName: productName,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

