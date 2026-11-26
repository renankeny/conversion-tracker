package ga4

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// SendRefund envia o evento padrão oficial "refund" ao GA4 Measurement Protocol
// vinculado por transaction_id (OrderID) para abatimento nativo de receita no GA4.
func (c *Client) SendRefund(
	ctx context.Context,
	apiSecret, measurementID, clientID, externalID, orderID, currency, productName string,
	value float64,
	refundedAt time.Time,
) SendResult {
	if strings.TrimSpace(apiSecret) == "" || strings.TrimSpace(measurementID) == "" {
		return SendResult{Err: fmt.Errorf("ga4 credentials ausentes")}
	}
	cleanOrder := strings.TrimSpace(orderID)
	if cleanOrder == "" {
		return SendResult{Err: fmt.Errorf("transaction_id ausente para refund GA4")}
	}
	cid := strings.TrimSpace(clientID)
	if cid == "" {
		cid = strings.TrimSpace(externalID)
	}
	if cid == "" {
		cid = "refund." + cleanOrder
	}
	curr := strings.ToUpper(strings.TrimSpace(currency))
	if len(curr) != 3 {
		curr = "BRL"
	}

	params := map[string]any{
		"transaction_id":       cleanOrder,
		"currency":             curr,
		"engagement_time_msec": 100,
	}
	if value > 0 {
		params["value"] = value
	}
	if name := strings.TrimSpace(productName); name != "" {
		params["items"] = []map[string]any{{"item_name": name, "quantity": 1}}
	}

	payload := Payload{
		ClientID: cid,
		UserID:   strings.TrimSpace(externalID),
		Events:   []Event{{Name: "refund", Params: params}},
	}
	if !refundedAt.IsZero() && time.Since(refundedAt) < 72*time.Hour {
		payload.TimestampMicros = refundedAt.Unix() * 1_000_000
	}
	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return SendResult{Err: err}
	}
	u := fmt.Sprintf("https://www.google-analytics.com/mp/collect?measurement_id=%s&api_secret=%s", measurementID, apiSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(bodyJSON))
	if err != nil {
		return SendResult{RequestBody: string(bodyJSON), Err: fmt.Errorf("ga4 refund build: %s", util.RedactSecrets(err.Error()))}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SendResult{RequestBody: string(bodyJSON), Err: fmt.Errorf("ga4 refund request: %s", util.RedactSecrets(err.Error()))}
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return SendResult{StatusCode: resp.StatusCode, Body: string(respBody), RequestBody: string(bodyJSON)}
	}
	return SendResult{
		StatusCode:  resp.StatusCode,
		Body:        string(respBody),
		RequestBody: string(bodyJSON),
		Err:         fmt.Errorf("ga4 refund returned status %d", resp.StatusCode),
	}
}
