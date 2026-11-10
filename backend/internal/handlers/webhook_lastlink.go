package handlers

import (
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// lastlinkNode agrupa campos de pedido/cliente/produto aceitos tanto na raiz quanto em Data.
type lastlinkNode struct {
	Buyer struct {
		Name        string `json:"Name"`
		Email       string `json:"Email"`
		PhoneNumber string `json:"PhoneNumber"`
		Phone       string `json:"Phone"`
	} `json:"Buyer"`
	Customer struct {
		Name  string `json:"Name"`
		Email string `json:"Email"`
		Phone string `json:"Phone"`
	} `json:"Customer"`
	Purchase struct {
		PaymentID string `json:"PaymentId"`
		ID        string `json:"Id"`
		Price     any    `json:"Price"`
		Amount    any    `json:"Amount"`
	} `json:"Purchase"`
	Order struct {
		ID          string `json:"Id"`
		Status      string `json:"Status"`
		Amount      any    `json:"Amount"`
		TotalAmount any    `json:"TotalAmount"`
		Currency    string `json:"Currency"`
	} `json:"Order"`
	Product struct {
		ID   any    `json:"Id"`
		Name string `json:"Name"`
	} `json:"Product"`
	Offer struct {
		ID   any    `json:"Id"`
		Name string `json:"Name"`
	} `json:"Offer"`
	Products []struct {
		ID    any    `json:"Id"`
		Name  string `json:"Name"`
		Price any    `json:"Price"`
	} `json:"Products"`
	Utm              map[string]any `json:"Utm"`
	CustomParameters map[string]any `json:"CustomParameters"`
}

// lastlinkPayload suporta tanto PascalCase (.NET padrão Lastlink com Data) quanto chaves na raiz.
// Nota: encoding/json em Go faz match case-insensitive automaticamente nos nomes das chaves!
type lastlinkPayload struct {
	ID     string `json:"Id"`
	Event  string `json:"Event"`
	Status string `json:"Status"`
	Token  string `json:"Token"`
	IsTest bool   `json:"IsTest"`
	Data   lastlinkNode
	lastlinkNode
}

// Lastlink processa o webhook /webhooks/lastlink.
func (h *Webhooks) Lastlink(w http.ResponseWriter, r *http.Request) {
	var p lastlinkPayload
	jsonBytes, err := decodeWebhookPayload(r, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if p.IsTest {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento de teste ignorado (IsTest=true)",
		})
		return
	}

	ev := strings.ToLower(strings.TrimSpace(p.Event))
	st := strings.ToLower(strings.TrimSpace(util.FirstNonEmpty(p.Status, p.Data.Order.Status, p.Order.Status)))
	approved := ev == "purchase_order_confirmed" || st == "paid" || st == "approved"
	if !approved {
		writeJSON(w, http.StatusOK, map[string]any{
			"received": true,
			"queued":   false,
			"reason":   "evento ignorado: " + util.FirstNonEmpty(p.Event, st),
		})
		return
	}

	accID := ExtractAccountID(r)
	if !h.ValidatePlatformToken(r.Context(), r, accID, "lastlink", p.Token) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"reason": "invalid platform token"})
		return
	}

	name := util.FirstNonEmpty(p.Data.Buyer.Name, p.Buyer.Name, p.Data.Customer.Name, p.Customer.Name)
	first, last := util.SplitFullName(strings.ToLower(name))
	email := util.FirstNonEmpty(p.Data.Buyer.Email, p.Buyer.Email, p.Data.Customer.Email, p.Customer.Email)
	phone := util.FirstNonEmpty(
		p.Data.Buyer.PhoneNumber, p.Data.Buyer.Phone,
		p.Buyer.PhoneNumber, p.Buyer.Phone,
		p.Data.Customer.Phone, p.Customer.Phone,
	)

	value := util.ParseMoneyAny(p.Data.Purchase.Price)
	for _, cand := range []any{p.Data.Purchase.Amount, p.Data.Order.TotalAmount, p.Data.Order.Amount, p.Order.TotalAmount, p.Order.Amount} {
		if value == 0 {
			value = util.ParseMoneyAny(cand)
		}
	}

	var prodID, prodName string
	if len(p.Data.Products) > 0 {
		prodID = anyString(p.Data.Products[0].ID)
		prodName = p.Data.Products[0].Name
		if value == 0 {
			value = util.ParseMoneyAny(p.Data.Products[0].Price)
		}
	} else if len(p.Products) > 0 {
		prodID = anyString(p.Products[0].ID)
		prodName = p.Products[0].Name
	}
	prodID = util.FirstNonEmpty(prodID, anyString(p.Data.Product.ID), anyString(p.Product.ID), anyString(p.Data.Offer.ID), anyString(p.Offer.ID))
	prodName = util.FirstNonEmpty(prodName, p.Data.Product.Name, p.Product.Name, p.Data.Offer.Name, p.Offer.Name)

	externalID := util.FirstNonEmpty(
		util.ExtractUserIDFromMap(p.Data.Utm),
		util.ExtractUserIDFromMap(p.Utm),
		util.ExtractUserIDFromMap(p.Data.CustomParameters),
		util.ExtractUserIDFromMap(p.CustomParameters),
		util.ExtractUserIDDeep(jsonBytes),
	)

	orderID := util.FirstNonEmpty(p.Data.Purchase.PaymentID, p.Data.Purchase.ID, p.Data.Order.ID, p.Order.ID, p.ID)
	in := purchaseInput{
		GatewayName: "lastlink",
		ExternalID:  externalID,
		OrderID:     orderID,
		Value:       value,
		Currency:    util.FirstNonEmpty(p.Data.Order.Currency, p.Order.Currency, "BRL"),
		FirstName:   first,
		LastName:    last,
		Email:       email,
		Phone:       phone,
		ProductID:   prodID,
		ProductName: prodName,
		AccountID:   accID,
		PixelQuery:  webhookPixelQuery(r),
	}
	h.processGenericPurchase(r.Context(), w, in, fromPurchaseInput(in))
}

