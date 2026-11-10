package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// perfectPayPayload reflete o JSON oficial de webhooks da PerfectPay.
type perfectPayPayload struct {
	Token            string `json:"token"`
	Code             string `json:"code"`
	SaleAmount       any    `json:"sale_amount"`
	SaleStatusEnum   any    `json:"sale_status_enum"`
	SaleStatusDetail string `json:"sale_status_detail"`
	Product          struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"product"`
	Plan struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"plan"`
	Customer struct {
		FullName      string `json:"full_name"`
		Email         string `json:"email"`
		PhoneAreaCode string `json:"phone_area_code"`
		PhoneNumber   string `json:"phone_number"`
		PhoneFormated string `json:"phone_formated"`
	} `json:"customer"`
	Metadata map[string]any `json:"metadata"`
}

// PerfectPay processa o webhook /webhooks/perfectpay.
// sale_status_enum == 2 representa venda aprovada na PerfectPay.
func (h *Webhooks) PerfectPay(w http.ResponseWriter, r *http.Request) {
	var p perfectPayPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	stEnum := strings.TrimSpace(anyString(p.SaleStatusEnum))
	stDetail := strings.ToLower(strings.TrimSpace(p.SaleStatusDetail))
	approved := stEnum == "2" || stDetail == "approved" || stDetail == "aprovado" || stDetail == "paid"
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "status ignorado: " + util.FirstNonEmpty(p.SaleStatusDetail, stEnum),
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "perfectpay", p.Token) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	first, last := util.SplitFullName(strings.ToLower(p.Customer.FullName))
	phone := util.FirstNonEmpty(
		joinPhoneDDD("", p.Customer.PhoneAreaCode, p.Customer.PhoneNumber),
		p.Customer.PhoneFormated,
	)

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Metadata),
		util.ExtractUserIDDeep(jsonBytes),
	)

	in := purchaseInput{
		GatewayName: "perfectpay",
		ExternalID:  externalID,
		OrderID:     p.Code,
		Value:       util.ParseMoneyAny(p.SaleAmount),
		Currency:    "BRL",
		FirstName:   first,
		LastName:    last,
		Email:       p.Customer.Email,
		Phone:       phone,
		ProductID:   util.FirstNonEmpty(p.Product.Code, p.Plan.Code),
		ProductName: util.FirstNonEmpty(p.Product.Name, p.Plan.Name),
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

