package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type googleAdsTestEventInput struct {
	EventType    string  `json:"event_type"`
	GCLID        string  `json:"gclid"`
	WBraid       string  `json:"wbraid"`
	GBraid       string  `json:"gbraid"`
	Value        float64 `json:"value"`
	Currency     string  `json:"currency"`
	OrderID      string  `json:"order_id"`
	Email        string  `json:"email"`
	Phone        string  `json:"phone"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	ValidateOnly *bool   `json:"validate_only"` // default true
}

// SendTestEvent dispara uma conversao sincronamente pra checar a conexao
// + Conversion Action. Por padrao usa validateOnly=true.
func (h *AdminGoogleAdsAccounts) SendTestEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	a, err := h.accounts.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conta nao encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && a.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	if !a.IsConnected() {
		writeError(w, http.StatusBadRequest, "conta nao conectada (faltando refresh_token, customer_id ou conversion action ativa)")
		return
	}

	var in googleAdsTestEventInput
	_ = json.NewDecoder(r.Body).Decode(&in)
	in.EventType = strings.TrimSpace(in.EventType)
	if in.EventType == "" {
		in.EventType = domain.EventPurchase
	}
	in.GCLID = strings.TrimSpace(in.GCLID)
	in.WBraid = strings.TrimSpace(in.WBraid)
	in.GBraid = strings.TrimSpace(in.GBraid)

	action := a.FindActionForEvent(in.EventType)
	if action == nil {
		writeError(w, http.StatusBadRequest, "nenhuma Conversion Action ativa configurada para event_type='"+in.EventType+"'")
		return
	}

	if in.GCLID == "" && in.WBraid == "" && in.GBraid == "" {
		in.GCLID = "Cj0KCQiTrackFindTESTgclid_" + strings.ReplaceAll(strings.ToUpper(action.ConversionActionID), " ", "")
	}

	validateOnly := true
	if in.ValidateOnly != nil {
		validateOnly = *in.ValidateOnly
	}

	value := in.Value
	var valuePtr *float64
	if value > 0 {
		valuePtr = &value
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
		EventID:   "test-" + strconv.FormatInt(int64(time.Now().Unix()), 10),
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

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	results := h.client.SendWithOptions(ctx, a, action, input, googleads.SendOptions{
		ValidateOnly: validateOnly,
	})

	out := make([]map[string]any, 0, len(results))
	for _, r := range results {
		entry := map[string]any{
			"customer_id":  r.CustomerID,
			"status_code":  r.StatusCode,
			"response":     r.Body,
			"request_body": r.RequestBody,
		}
		if r.Err != nil {
			entry["error"] = r.Err.Error()
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
