// Package tiktok encapsula chamadas à TikTok Events API V1.3.
//
// Spec oficial (consultada em 2026-06-15):
//   - Events API Reference:
//     https://business-api.tiktok.com/portal/docs?id=1771101303285761
//   - User identifier hashing:
//     https://business-api.tiktok.com/portal/docs?id=1739585702641154
//
// PII (email, phone, external_id) sempre vai hasheada SHA256 hex lowercase.
// Tokens browser (ttclid, ttp, ip, user_agent) NÃO. Mesmo padrão do Meta.
//
// Endpoint: POST https://business-api.tiktok.com/open_api/v1.3/event/track/
// Header: Access-Token: <access_token>
//
// Eventos web standard suportados pela V1.3:
//   - AddPaymentInfo, AddToCart, AddToWishlist, ClickButton,
//   - CompletePayment, CompleteRegistration, Contact, Download,
//   - InitiateCheckout, PlaceAnOrder, Search, SubmitForm,
//   - Subscribe, ViewContent
//
// Mapeamento Meta -> TikTok (decisões deliberadas):
//   PageView              -> ViewContent (TikTok V1.3 nao tem PageView
//                                        server-side; ViewContent e o
//                                        equivalente aceito pra "viu pagina")
//   ViewContent           -> ViewContent
//   AddToCart             -> AddToCart
//   AddToWishlist         -> AddToWishlist
//   InitiateCheckout      -> InitiateCheckout
//   AddPaymentInfo        -> AddPaymentInfo
//   Purchase              -> CompletePayment (equivalente direto)
//   Lead                  -> SubmitForm (Lead = captura de form geralmente)
//   Contact               -> Contact (clique em WhatsApp/tel/email)
//   CompleteRegistration  -> CompleteRegistration
//
// Eventos custom (Scroll_*, Timer_*, ViewVideo_*, PlayVideo) -> NAO envia.
// Sao metricas de engajamento que o TikTok nao usa pra otimizacao e
// apenas poluem o feed se forwardados.
//
// Init / UpdateUserData -> NAO envia (eventos internos da plataforma).
package tiktok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const (
	baseURL  = "https://business-api.tiktok.com/open_api/v1.3/event/track/"
	eventSrc = "web"
)

type Client struct {
	partnerName string
	http        *http.Client
}

// New cria um cliente da TikTok Events API. partnerName é opcional
// (usado pra identificar a plataforma agindo em nome do anunciante).
func New(partnerName string) *Client {
	return &Client{
		partnerName: partnerName,
		http:        &http.Client{Timeout: 10 * time.Second},
	}
}

// SendResult agrupa a resposta de um envio TikTok.
type SendResult struct {
	PixelID     string
	StatusCode  int
	Body        string
	RequestBody string
	Err         error
}

// Send envia o evento para todos os tiktok_pixel_id/access_token configurados
// no domain.Pixel. Retorna um SendResult por pixel TikTok.
//
// Se o evento Meta não tiver mapeamento pro TikTok (PageView, custom events,
// Init etc), retorna lista vazia — não conta como envio nem como falha.
func (c *Client) Send(ctx context.Context, pixel *domain.Pixel, in domain.SendEventInput) []SendResult {
	tiktokEvent := mapEventName(in.EventName)
	if tiktokEvent == "" {
		return nil // evento ignorado pra TikTok
	}
	results := make([]SendResult, 0, len(pixel.TikTokPixelIDs))
	for i, pixelID := range pixel.TikTokPixelIDs {
		token := ""
		testCode := ""
		if i < len(pixel.TikTokAccessTokens) {
			token = pixel.TikTokAccessTokens[i]
		}
		if i < len(pixel.TikTokTestCodes) {
			testCode = pixel.TikTokTestCodes[i]
		}
		results = append(results, c.sendOne(ctx, pixelID, token, testCode, tiktokEvent, in))
	}
	return results
}

func (c *Client) sendOne(ctx context.Context, pixelID, token, testCode, tiktokEvent string, in domain.SendEventInput) SendResult {
	res := SendResult{PixelID: pixelID}
	if token == "" {
		res.Err = fmt.Errorf("missing access token for tiktok pixel %s", pixelID)
		return res
	}

	body := buildPayload(pixelID, testCode, c.partnerName, tiktokEvent, in)
	raw, err := json.Marshal(body)
	if err != nil {
		res.Err = err
		return res
	}
	res.RequestBody = string(raw)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(raw))
	if err != nil {
		res.Err = err
		return res
	}
	req.Header.Set("Access-Token", token)
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

	// TikTok pode retornar 200 com code != 0 no body indicando erro logico.
	// Detectamos isso e marcamos como erro pra o worker registrar como failed.
	if resp.StatusCode >= 300 {
		res.Err = fmt.Errorf("tiktok returned %d: %s", resp.StatusCode, res.Body)
		return res
	}
	if logicalErr := extractTikTokError(respBody); logicalErr != "" {
		res.Err = fmt.Errorf("tiktok logical error: %s", logicalErr)
	}
	return res
}

// extractTikTokError olha o JSON de resposta. TikTok retorna sempre
// {"code": 0, "message": "OK", "request_id": "...", "data": {...}}
// quando ok. Code != 0 é erro logico.
func extractTikTokError(body []byte) string {
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return ""
	}
	if out.Code == 0 {
		return ""
	}
	return fmt.Sprintf("code=%d message=%s", out.Code, out.Message)
}

// buildPayload monta o JSON enviado pra TikTok Events API V1.3.
func buildPayload(pixelID, testCode, partnerName, eventName string, in domain.SendEventInput) map[string]any {
	user := map[string]any{}

	// PII hasheada (TikTok aceita SHA256 hex lowercase como Meta).
	setHashed(user, "email", util.NormalizeEmail(in.Email))
	setHashed(user, "phone", util.NormalizePhone(in.Phone))
	setHashed(user, "external_id", in.ExternalID)
	setHashedFirst(user, "first_name", util.NormalizeName(in.FirstName))
	setHashedFirst(user, "last_name", util.NormalizeName(in.LastName))
	setHashedFirst(user, "city", util.NormalizeCity(in.City))
	setHashedFirst(user, "state", util.NormalizeStateISO(in.State))
	setHashedFirst(user, "zip_code", util.NormalizeZip(in.PostalCode))
	setHashedFirst(user, "country_code", util.NormalizeCountry(in.Country))

	// Tokens NAO hasheados.
	setIfNotEmpty(user, "ttclid", in.TTClid)
	setIfNotEmpty(user, "ttp", in.TTP)
	setIfNotEmpty(user, "ip", in.ClientIP)
	setIfNotEmpty(user, "user_agent", in.ClientUserAgent)
	user["lang"] = "pt"

	page := map[string]any{}
	if in.SourceURL != "" {
		page["url"] = in.SourceURL
	}
	if in.Referrer != "" {
		page["referrer"] = in.Referrer
	}

	properties := map[string]any{}
	if in.ContentID != "" {
		properties["content_id"] = in.ContentID
		properties["content_type"] = orDefault(in.ContentType, "product")
		// O array "contents" e o local oficial pro detalhe do produto/item
		// segundo a doc V1.3. Os campos top-level (content_id, content_type)
		// sao mantidos por compat.
		item := map[string]any{
			"content_id":   in.ContentID,
			"content_type": orDefault(in.ContentType, "product"),
			"content_name": firstNonEmpty(in.ContentName, in.PixelName),
		}
		if in.Value != nil {
			item["price"] = *in.Value
			// Quantity opcional - se NumItems veio pro evento, usa, senao 1.
			if in.NumItems != nil {
				item["quantity"] = *in.NumItems
			} else {
				item["quantity"] = 1
			}
		}
		properties["contents"] = []map[string]any{item}
	}
	if in.ContentName != "" {
		properties["content_name"] = in.ContentName
	}
	if in.Value != nil {
		properties["value"] = *in.Value
		properties["currency"] = orDefault(in.Currency, "BRL")
	}
	if in.OrderID != "" {
		properties["order_id"] = in.OrderID
	}
	if in.NumItems != nil && (eventName == "InitiateCheckout" || eventName == "AddToCart") {
		properties["num_items"] = *in.NumItems
	}

	data := map[string]any{
		"event":      eventName,
		"event_time": in.EventTime,
		"event_id":   in.EventID,
		"user":       user,
	}
	if len(page) > 0 {
		data["page"] = page
	}
	if len(properties) > 0 {
		data["properties"] = properties
	}

	payload := map[string]any{
		"event_source":    eventSrc,
		"event_source_id": pixelID,
		"data":            []any{data},
	}
	if testCode != "" {
		payload["test_event_code"] = testCode
	}
	if partnerName != "" {
		payload["partner_name"] = partnerName
	}
	return payload
}

// mapEventName traduz o nome do evento Meta pro nome aceito pela TikTok.
// Retorna "" pra eventos que NÃO devem ir pro TikTok (PageView, custom
// events, eventos internos). Veja docstring do package pra justificativa
// de cada decisão.
func mapEventName(name string) string {
	switch name {

	// === Eventos comerciais (mesmo nome em ambas as plataformas) ====
	case domain.EventViewContent, domain.EventPageView:
		// PageView vai como ViewContent porque TikTok Events API V1.3
		// nao aceita PageView server-side. ViewContent e o equivalente
		// "viu pagina" e a Meta usa o event_id pra dedup com browser-side.
		return "ViewContent"
	case domain.EventAddToCart:
		return "AddToCart"
	case domain.EventAddToWishlist:
		return "AddToWishlist"
	case domain.EventInitiateCheckout:
		return "InitiateCheckout"
	case domain.EventAddPaymentInfo:
		return "AddPaymentInfo"
	case domain.EventCompleteRegistration:
		return "CompleteRegistration"
	case domain.EventContact:
		return "Contact"

	// === Eventos com nome diferente em cada plataforma ===============
	case domain.EventPurchase:
		return "CompletePayment"
	case domain.EventLead:
		return "SubmitForm"

	// === NÃO enviar pro TikTok (decisão de produto) ==================
	// Init/UpdateUserData: eventos internos da plataforma.
	// Custom (Scroll/Timer/Video): metricas de engajamento que nao
	// servem pra otimizacao do TikTok algoritmo.
	case domain.EventInit, domain.EventUpdateUserData,
		domain.EventScroll25, domain.EventScroll50, domain.EventScroll75, domain.EventScroll90,
		domain.EventTimer1Min, domain.EventPlayVideo,
		domain.EventViewVideo25, domain.EventViewVideo50, domain.EventViewVideo75, domain.EventViewVideo90:
		return ""
	}

	slog.Debug("tiktok: evento sem mapeamento conhecido", "event", name)
	return ""
}

func setHashed(m map[string]any, key, val string) {
	if h := util.SHA256Hex(val); h != "" {
		m[key] = h
	}
}

// setHashedFirst grava o hash apenas se não for vazio. TikTok espera
// scalar string nesses campos (vs Meta que espera array).
func setHashedFirst(m map[string]any, key, val string) {
	if h := util.SHA256Hex(val); h != "" {
		m[key] = h
	}
}

func setIfNotEmpty(m map[string]any, key, val string) {
	v := strings.TrimSpace(val)
	if v != "" {
		m[key] = v
	}
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func firstNonEmpty(opts ...string) string {
	for _, o := range opts {
		if strings.TrimSpace(o) != "" {
			return o
		}
	}
	return ""
}
