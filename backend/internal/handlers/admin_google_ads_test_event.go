package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
)

type googleAdsTestEventInput struct {
	EventType          string  `json:"event_type"`
	ConversionActionID string  `json:"conversion_action_id"`
	ActionType         string  `json:"action_type"`
	OwnerCustomerID    string  `json:"owner_customer_id"`
	CustomerID         string  `json:"customer_id"`
	LoginCustomerID    string  `json:"login_customer_id"`
	GCLID              string  `json:"gclid"`
	WBraid             string  `json:"wbraid"`
	GBraid             string  `json:"gbraid"`
	Value              float64 `json:"value"`
	Currency           string  `json:"currency"`
	OrderID            string  `json:"order_id"`
	Email              string  `json:"email"`
	Phone              string  `json:"phone"`
	FirstName          string  `json:"first_name"`
	LastName           string  `json:"last_name"`
	ValidateOnly       *bool   `json:"validate_only"`
}

func (h *AdminGoogleAdsAccounts) SendTestEvent(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para testar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}

	var in googleAdsTestEventInput
	_ = json.NewDecoder(r.Body).Decode(&in)

	accCopy := *a
	if cid := strings.TrimSpace(in.CustomerID); cid != "" {
		accCopy.CustomerID = cid
	}
	if lcid := strings.TrimSpace(in.LoginCustomerID); lcid != "" {
		accCopy.LoginCustomerID = lcid
	} else if strings.TrimSpace(in.CustomerID) != "" && in.CustomerID != a.CustomerID {
		accCopy.LoginCustomerID = ""
	}

	if strings.TrimSpace(accCopy.RefreshToken) == "" {
		writeError(w, http.StatusBadRequest, "conta Google Ads ainda não possui conexão OAuth ativa")
		return
	}
	if strings.TrimSpace(accCopy.CustomerID) == "" {
		writeError(w, http.StatusBadRequest, "selecione uma Conta Google Ads no Passo 2 antes de testar")
		return
	}

	in.EventType = strings.TrimSpace(in.EventType)
	if in.EventType == "" {
		in.EventType = domain.EventPurchase
	}
	in.GCLID = strings.TrimSpace(in.GCLID)
	in.WBraid = strings.TrimSpace(in.WBraid)
	in.GBraid = strings.TrimSpace(in.GBraid)

	action := accCopy.FindActionByIDOrEvent(in.ConversionActionID, in.EventType)
	if action == nil || strings.TrimSpace(action.ConversionActionID) == "" {
		writeError(w, http.StatusBadRequest, "nenhuma Conversion Action configurada para o evento '"+in.EventType+"'")
		return
	}
	actCopy := *action
	if tp := strings.TrimSpace(in.ActionType); tp != "" {
		actCopy.ActionType = tp
	}
	if own := strings.TrimSpace(in.OwnerCustomerID); own != "" {
		actCopy.OwnerCustomerID = own
	}
	action = &actCopy

	if in.GCLID == "" && in.WBraid == "" && in.GBraid == "" {
		in.GCLID = "Cj0KCQiTrackFindTESTgclid_" + strings.ReplaceAll(strings.ToUpper(action.ConversionActionID), " ", "")
	}

	validateOnly := true
	if in.ValidateOnly != nil {
		validateOnly = *in.ValidateOnly
	}

	var valuePtr *float64
	if in.Value > 0 {
		v := in.Value
		valuePtr = &v
	} else if in.EventType == domain.EventPurchase {
		v := 1.0
		valuePtr = &v
	}
	currency := strings.TrimSpace(in.Currency)
	if currency == "" {
		currency = "BRL"
	}

	input := domain.SendEventInput{
		EventName: in.EventType,
		EventID:   "test-" + strconv.FormatInt(time.Now().Unix(), 10),
		EventTime: time.Now().Unix(),
		Value:     valuePtr,
		Currency:  currency,
		OrderID:   strings.TrimSpace(in.OrderID),
		GCLID:     in.GCLID,
		WBraid:    in.WBraid,
		GBraid:    in.GBraid,
		Email:     strings.TrimSpace(in.Email),
		Phone:     strings.TrimSpace(in.Phone),
		FirstName: strings.TrimSpace(in.FirstName),
		LastName:  strings.TrimSpace(in.LastName),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	results := h.client.SendWithOptions(ctx, &accCopy, action, input, googleads.SendOptions{
		ValidateOnly: validateOnly,
	})

	out := make([]map[string]any, 0, len(results))
	for _, res := range results {
		entry := map[string]any{
			"customer_id":  res.CustomerID,
			"status_code":  res.StatusCode,
			"response":     res.Body,
			"request_body": res.RequestBody,
		}
		if res.Err != nil {
			entry["error"] = res.Err.Error()
		}
		out = append(out, entry)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"validate_only":        validateOnly,
		"event_type":           in.EventType,
		"conversion_action_id": action.ConversionActionID,
		"results":              out,
	})
}
