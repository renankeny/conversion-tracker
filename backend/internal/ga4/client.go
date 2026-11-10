package ga4

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

var ErrEventNotMapped = fmt.Errorf("event not mapped")

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type Payload struct {
	ClientID        string  `json:"client_id"`
	UserID          string  `json:"user_id,omitempty"`
	TimestampMicros int64   `json:"timestamp_micros,omitempty"`
	Events          []Event `json:"events"`
}

type Event struct {
	Name   string         `json:"name"`
	Params map[string]any `json:"params,omitempty"`
}

type SendResult struct {
	StatusCode  int
	Body        string
	RequestBody string
	Err         error
}

// SendEvents envia o evento para o Measurement Protocol do GA4.
func (c *Client) SendEvents(ctx context.Context, apiSecret, measurementID string, input *domain.SendEventInput) SendResult {
	// GA4 exige client_id no payload
	if input.GA4ClientID == "" {
		return SendResult{Err: fmt.Errorf("ga4_client_id is required")}
	}

	// Mapeamento de Eventos (TrackFind -> GA4)
	ga4EventName := mapEventName(input.EventName)
	if ga4EventName == "" {
		// Evento não mapeado, aborta silenciosamente
		return SendResult{Err: ErrEventNotMapped}
	}

	params := map[string]any{
		"engagement_time_msec": 100,
	}

	// Adiciona session_id se disponível (como inteiro quando numérico)
	if input.GA4SessionID != "" {
		if sid, err := strconv.ParseInt(input.GA4SessionID, 10, 64); err == nil {
			params["session_id"] = sid
		} else {
			params["session_id"] = input.GA4SessionID
		}
	}

	// Adiciona IP e UserAgent (Embora o MP do GA4 recomende enviá-los via headers ou IPs no payload, o GA4 v4 removeu
	// suporte nativo de IP override. Tudo é feito no backend deles baseado nos hits originais.)

	// Mapeia Valor monetário
	if input.Value != nil {
		params["value"] = *input.Value
	}
	if input.Currency != "" {
		params["currency"] = input.Currency
	} else if input.Value != nil {
		params["currency"] = "BRL" // default fallback
	}

	// Transação
	if input.OrderID != "" {
		params["transaction_id"] = input.OrderID
	} else if ga4EventName == "purchase" && input.EventID != "" {
		params["transaction_id"] = input.EventID
	}

	// Itens (Simplificado - se precisar do array de itens do e-commerce)
	if input.ContentID != "" || input.ContentName != "" {
		item := map[string]any{}
		if input.ContentID != "" {
			item["item_id"] = input.ContentID
		}
		if input.ContentName != "" {
			item["item_name"] = input.ContentName
		}
		params["items"] = []map[string]any{item}
	}

	payload := Payload{
		ClientID: input.GA4ClientID,
		UserID:   input.ExternalID,
		Events: []Event{
			{
				Name:   ga4EventName,
				Params: params,
			},
		},
	}
	if input.EventTime > 0 && time.Since(time.Unix(input.EventTime, 0)) < 72*time.Hour {
		payload.TimestampMicros = input.EventTime * 1_000_000
	}

	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return SendResult{Err: err}
	}
	requestBody := string(bodyJSON)

	url := fmt.Sprintf("https://www.google-analytics.com/mp/collect?measurement_id=%s&api_secret=%s", measurementID, apiSecret)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return SendResult{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SendResult{Err: err}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// O GA4 MP geralmente retorna 204 No Content se bem sucedido
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return SendResult{
			StatusCode:  resp.StatusCode,
			Body:        string(respBody),
			RequestBody: requestBody,
		}
	}

	return SendResult{
		StatusCode:  resp.StatusCode,
		Body:        string(respBody),
		RequestBody: requestBody,
		Err:         fmt.Errorf("ga4 returned status %d", resp.StatusCode),
	}
}

// mapEventName traduz o nome de evento do TrackFind pro GA4.
//
// Prioriza o nome "recomendado" pelo Google quando existe (ganha os
// relatórios prontos do GA4 — funil de e-commerce, etc). Pros eventos sem
// recomendado oficial, manda como evento customizado (nome próprio em
// snake_case) — o Measurement Protocol aceita qualquer nome, só não tem
// relatório especial pronto. Isso é melhor que descartar silenciosamente:
// o dado chega no GA4 de qualquer forma, só sem tratamento especial.
func mapEventName(eventName string) string {
	switch eventName {
	// --- Recomendados oficiais do GA4 ---
	case domain.EventPurchase:
		return "purchase"
	case domain.EventLead:
		return "generate_lead"
	case domain.EventPageView:
		return "page_view"
	case domain.EventViewContent:
		return "view_item"
	case domain.EventAddToCart:
		return "add_to_cart"
	case domain.EventInitiateCheckout:
		return "begin_checkout"
	case domain.EventAddShippingInfo:
		return "add_shipping_info"
	case domain.EventAddPaymentInfo:
		return "add_payment_info"
	case domain.EventAddToWishlist:
		return "add_to_wishlist"
	case domain.EventCompleteRegistration:
		return "sign_up"
	case "Search":
		return "search"
	case "SubmitApplication":
		return "submit_application"

	// --- Sem recomendado oficial — manda como customizado (mesma info,
	// só sem os relatórios prontos que o GA4 dá pros recomendados) ---
	case domain.EventContact:
		return "contact"
	case domain.EventLeadSubmitted:
		return "lead_submitted"
	case "Subscribe":
		return "subscribe"
	case "FindLocation":
		return "find_location"
	case "Schedule":
		return "schedule"
	case "StartTrial":
		return "start_trial"
	case "Donate":
		return "donate"
	case "CustomizeProduct":
		return "customize_product"
	case domain.EventScroll25:
		return "scroll_25"
	case domain.EventScroll50:
		return "scroll_50"
	case domain.EventScroll75:
		return "scroll_75"
	case domain.EventScroll90:
		return "scroll_90"
	case domain.EventTimer1Min:
		return "timer_1min"
	case domain.EventPlayVideo:
		return "play_video"
	case domain.EventViewVideo25:
		return "view_video_25"
	case domain.EventViewVideo50:
		return "view_video_50"
	case domain.EventViewVideo75:
		return "view_video_75"
	case domain.EventViewVideo90:
		return "view_video_90"
	}
	return ""
}
