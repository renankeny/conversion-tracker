// Package meta encapsula chamadas à Meta Conversions API (Graph API /events).
//
// Spec oficial (consultada em 2026-06-13):
//   - Customer Information Parameters:
//     https://developers.facebook.com/docs/marketing-api/conversions-api/parameters/customer-information-parameters
//   - Custom Data Parameters:
//     https://developers.facebook.com/docs/marketing-api/conversions-api/parameters/custom-data
//   - Best Practices:
//     https://developers.facebook.com/docs/marketing-api/conversions-api/best-practices
//
// Para eventos web, os parâmetros OBRIGATÓRIOS são:
//   - action_source = "website"
//   - event_source_url
//   - client_user_agent (no user_data)
//
// PII (em, ph, fn, ln, ct, st, zp, country, external_id, db, ge) sempre vai
// hasheada SHA256 hex lowercase. Tokens browser (fbp, fbc, ip, UA) NÃO.
package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const baseURL = "https://graph.facebook.com"

type Client struct {
	apiVersion   string
	partnerAgent string // opcional, nome da plataforma agindo em nome do anunciante
	http         *http.Client
}

// New cria um cliente da CAPI. apiVersion deve ser do formato "v25.0".
// partnerAgent é opcional — preencha se você é uma agência/plataforma
// (ex: "trackfind-1.0.0").
func New(apiVersion, partnerAgent string) *Client {
	return &Client{
		apiVersion:   apiVersion,
		partnerAgent: partnerAgent,
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

// SendResult agrupa a resposta de um envio.
type SendResult struct {
	PixelID     string
	StatusCode  int
	Body        string // resposta da Meta (com tokens redatados antes de persistir)
	RequestBody string // payload JSON enviado (sem token — só headers/url tinham)
	Err         error
}

// Send envia o evento para todos os pixel_id/access_token configurados no domain.Pixel.
// Retorna um SendResult por pixel.
func (c *Client) Send(ctx context.Context, pixel *domain.Pixel, in domain.SendEventInput) []SendResult {
	results := make([]SendResult, 0, len(pixel.PixelIDs))
	for i, pixelID := range pixel.PixelIDs {
		token := ""
		testCode := ""
		if i < len(pixel.AccessTokens) {
			token = pixel.AccessTokens[i]
		}
		if i < len(pixel.TestCodes) {
			testCode = pixel.TestCodes[i]
		}
		results = append(results, c.sendOne(ctx, pixelID, token, testCode, in))
	}
	return results
}

func (c *Client) sendOne(ctx context.Context, pixelID, token, testCode string, in domain.SendEventInput) SendResult {
	res := SendResult{PixelID: pixelID}
	if token == "" {
		res.Err = fmt.Errorf("missing access token for pixel %s", pixelID)
		return res
	}

	// Eventos web exigem event_source_url. Se vier vazio, loga warn —
	// o Meta provavelmente vai descartar pelo critério dele.
	if isWebEvent(in.EventName) && in.SourceURL == "" {
		slog.Warn("missing event_source_url for web event",
			"event", in.EventName,
			"pixel_id", pixelID,
			"event_id", in.EventID,
		)
	}

	body := buildPayload(in, testCode, c.partnerAgent)
	raw, err := json.Marshal(body)
	if err != nil {
		res.Err = err
		return res
	}
	res.RequestBody = string(raw)

	url := fmt.Sprintf("%s/%s/%s/events?access_token=%s", baseURL, c.apiVersion, pixelID, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		res.Err = err
		return res
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		res.Err = err
		return res
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	res.StatusCode = resp.StatusCode
	res.Body = string(respBody)
	if resp.StatusCode >= 300 {
		res.Err = fmt.Errorf("meta returned %d: %s", resp.StatusCode, res.Body)
	}
	return res
}

// buildPayload monta o JSON enviado pra Graph API /events. Aplica
// normalização e hash SHA256 nas PIIs conforme spec.
func buildPayload(in domain.SendEventInput, testCode, partnerAgent string) map[string]any {
	userData := map[string]any{}

	// PII hasheada (formatos exigidos pela spec).
	setHashed(userData, "em", util.NormalizeEmail(in.Email))
	setHashed(userData, "ph", util.NormalizePhone(in.Phone))
	setHashed(userData, "fn", util.NormalizeName(in.FirstName))
	setHashed(userData, "ln", util.NormalizeName(in.LastName))
	setHashed(userData, "ct", util.NormalizeCity(in.City))
	setHashed(userData, "st", util.NormalizeStateISO(in.State))
	setHashed(userData, "zp", util.NormalizeZip(in.PostalCode))
	setHashed(userData, "country", util.NormalizeCountry(in.Country))
	// external_id: hash recomendado pela spec, mantemos hasheado.
	setHashed(userData, "external_id", in.ExternalID)

	// Tokens NÃO hasheados.
	setIfNotEmpty(userData, "client_ip_address", in.ClientIP)
	setIfNotEmpty(userData, "client_user_agent", in.ClientUserAgent)
	setIfNotEmpty(userData, "fbp", in.FBP)
	setIfNotEmpty(userData, "fbc", in.FBC)

	customData := map[string]any{}
	if in.ContentID != "" {
		customData["content_ids"] = []string{in.ContentID}
		ct := in.ContentType
		if ct == "" {
			ct = "product"
		}
		customData["content_type"] = ct
	}
	if in.ContentName != "" {
		customData["content_name"] = in.ContentName
	}
	if in.Value != nil {
		customData["value"] = *in.Value
		customData["currency"] = orDefault(in.Currency, "BRL")
	}
	if in.OrderID != "" {
		customData["order_id"] = in.OrderID
	}
	if in.NumItems != nil && in.EventName == domain.EventInitiateCheckout {
		customData["num_items"] = *in.NumItems
	}

	event := map[string]any{
		"event_name":    in.EventName,
		"event_time":    in.EventTime,
		"event_id":      in.EventID,
		"action_source": "website",
		"user_data":     userData,
	}
	if in.SourceURL != "" {
		event["event_source_url"] = in.SourceURL
	}
	if len(customData) > 0 {
		event["custom_data"] = customData
	}

	payload := map[string]any{
		"data": []any{event},
	}
	if testCode != "" {
		payload["test_event_code"] = testCode
	}
	if partnerAgent != "" {
		payload["partner_agent"] = partnerAgent
	}
	return payload
}

// isWebEvent retorna true pros eventos disparados de browser (que exigem
// event_source_url e client_user_agent na spec). Eventos custom também
// são disparados de web no nosso fluxo.
func isWebEvent(name string) bool {
	switch name {
	case domain.EventPageView, domain.EventViewContent, domain.EventLead,
		domain.EventContact, domain.EventInitiateCheckout, domain.EventAddPaymentInfo,
		domain.EventAddToCart, domain.EventAddToWishlist,
		domain.EventCompleteRegistration, domain.EventPurchase,
		domain.EventScroll25, domain.EventScroll50, domain.EventScroll75, domain.EventScroll90,
		domain.EventTimer1Min, domain.EventPlayVideo,
		domain.EventViewVideo25, domain.EventViewVideo50, domain.EventViewVideo75, domain.EventViewVideo90:
		return true
	}
	return false
}

func setHashed(m map[string]any, key, val string) {
	if h := util.SHA256Hex(val); h != "" {
		m[key] = []string{h}
	}
}

func setIfNotEmpty(m map[string]any, key, val string) {
	if val != "" {
		m[key] = val
	}
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
