package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type metaAccountTestInput struct {
	AdAccountID   string  `json:"ad_account_id"`
	PixelID       string  `json:"pixel_id"`
	EventName     string  `json:"event_name"`
	TestEventCode string  `json:"test_event_code"`
	Value         float64 `json:"value"`
	Currency      string  `json:"currency"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
}

func (h *AdminMetaAccounts) SendTestEvent(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para testar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	if a.IsTokenExpired() {
		writeError(w, http.StatusBadRequest, "o token OAuth desta conta expirou. Clique em Reconectar no Passo 1")
		return
	}
	if !a.IsConnected() {
		writeError(w, http.StatusBadRequest, "conta Meta Ads ainda não possui conexão OAuth ativa")
		return
	}

	var in metaAccountTestInput
	_ = json.NewDecoder(r.Body).Decode(&in)

	accCopy := *a
	if adAcc := strings.TrimSpace(in.AdAccountID); adAcc != "" {
		accCopy.AdAccountID = adAcc
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	pixelID := strings.TrimSpace(in.PixelID)
	if pixelID == "" && accCopy.AdAccountID != "" {
		if pxRes, err := h.client.ListPixels(ctx, &accCopy, accCopy.AdAccountID); err == nil {
			pixelID = extractFirstPixelID(pxRes["pixels"])
		}
	}

	if pixelID == "" {
		adRes, err := h.client.ListAdAccounts(ctx, &accCopy)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"mode":        "connection_check",
				"status_code": 502,
				"error":       err.Error(),
			})
			return
		}
		count := 0
		if arr, ok := adRes["ad_accounts"].([]any); ok {
			count = len(arr)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"mode":        "connection_check",
			"status_code": 200,
			"response":    fmt.Sprintf(`{"ok":true,"ad_accounts_accessible":%d,"note":"Conexão OAuth validada com sucesso na Graph API (nenhum Dataset/Pixel encontrado nesta conta para disparo CAPI)."}`, count),
		})
		return
	}

	evName := strings.TrimSpace(in.EventName)
	if evName == "" {
		evName = domain.EventLead
	}
	currency := strings.TrimSpace(in.Currency)
	if currency == "" {
		currency = "BRL"
	}
	var valPtr *float64
	if in.Value > 0 {
		v := in.Value
		valPtr = &v
	} else if evName == domain.EventPurchase {
		v := 97.0
		valPtr = &v
	}

	email := strings.TrimSpace(in.Email)
	if email == "" {
		email = "teste.capi@exemplo.com"
	}
	phone := strings.TrimSpace(in.Phone)
	if phone == "" {
		phone = "5511999999999"
	}

	sendIn := domain.SendEventInput{
		EventName:       evName,
		EventID:         "TEST_" + util.GenerateID10(),
		EventTime:       time.Now().Unix(),
		SourceURL:       strings.TrimRight(h.frontHost, "/") + "/integracoes/meta-ads",
		Value:           valPtr,
		Currency:        currency,
		ExternalID:      "test-" + util.GenerateID10(),
		ClientIP:        extractTestClientIP(r),
		ClientUserAgent: orTestUA(r.UserAgent()),
		Email:           email,
		Phone:           phone,
	}

	rawPayload, err := meta.BuildCAPIPayload(sendIn, strings.TrimSpace(in.TestEventCode), h.metaPartner)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao montar payload CAPI: "+err.Error())
		return
	}

	code, body, sendErr := h.client.SendCAPIEvent(ctx, &accCopy, pixelID, rawPayload)
	errStr := ""
	if sendErr != nil {
		errStr = sendErr.Error()
	} else if code >= 300 {
		errStr = fmt.Sprintf("Meta CAPI retornou status %d", code)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"mode":         "capi_event",
		"event_name":   evName,
		"pixel_id":     pixelID,
		"status_code":  code,
		"response":     util.RedactSecrets(body),
		"request_body": string(rawPayload),
		"error":        errStr,
	})
}

func extractFirstPixelID(raw any) string {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	if m, ok := items[0].(map[string]any); ok {
		id, _ := m["id"].(string)
		return strings.TrimSpace(id)
	}
	return ""
}

func orTestUA(ua string) string {
	if strings.TrimSpace(ua) == "" {
		return "TrackFind-TestEvent/1.0"
	}
	return ua
}
