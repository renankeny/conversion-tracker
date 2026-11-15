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
	"github.com/renankeny/conversion-tracker/backend/internal/util"
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
	if input.GA4ClientID == "" {
		return SendResult{Err: fmt.Errorf("ga4_client_id is required")}
	}

	ga4EventName := mapEventName(input.EventName)
	if ga4EventName == "" {
		return SendResult{Err: ErrEventNotMapped}
	}

	params := map[string]any{
		"engagement_time_msec": 100,
	}

	if input.GA4SessionID != "" {
		if sid, err := strconv.ParseInt(input.GA4SessionID, 10, 64); err == nil {
			params["session_id"] = sid
		} else {
			params["session_id"] = input.GA4SessionID
		}
	}

	if input.Value != nil {
		params["value"] = *input.Value
	}
	if input.Currency != "" {
		params["currency"] = input.Currency
	} else if input.Value != nil {
		params["currency"] = "BRL"
	}

	if input.OrderID != "" {
		params["transaction_id"] = input.OrderID
	} else if ga4EventName == "purchase" && input.EventID != "" {
		params["transaction_id"] = input.EventID
	}

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
		return SendResult{
			RequestBody: requestBody,
			Err:         fmt.Errorf("ga4 build request failed: %s", util.RedactSecrets(err.Error())),
		}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SendResult{
			RequestBody: requestBody,
			Err:         fmt.Errorf("ga4 request failed: %s", util.RedactSecrets(err.Error())),
		}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

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
