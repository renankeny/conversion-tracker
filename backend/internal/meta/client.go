// Package meta encapsula chamadas à Meta Conversions API (Graph API /events).
package meta

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

const baseURL = "https://graph.facebook.com"

type Client struct {
	apiVersion   string
	partnerAgent string
	http         *http.Client
}

func New(apiVersion, partnerAgent string) *Client {
	return &Client{
		apiVersion:   apiVersion,
		partnerAgent: partnerAgent,
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

type SendResult struct {
	PixelID     string
	StatusCode  int
	Body        string
	RequestBody string
	Err         error
}

func (c *Client) Send(ctx context.Context, pixel *domain.Pixel, in domain.SendEventInput) []SendResult {
	return c.SendFiltered(ctx, pixel, in, nil)
}

func (c *Client) SendFiltered(ctx context.Context, pixel *domain.Pixel, in domain.SendEventInput, skip func(string) bool) []SendResult {
	results := make([]SendResult, 0, len(pixel.PixelIDs))
	for i, pixelID := range pixel.PixelIDs {
		if skip != nil && skip(pixelID) {
			continue
		}
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

	if isWebEvent(in.EventName) && in.SourceURL == "" && in.ActionSource != "business_messaging" {
		slog.Warn("missing event_source_url for web event", "event", in.EventName, "pixel_id", pixelID, "event_id", in.EventID)
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

func (c *Client) PartnerAgent() string {
	return c.partnerAgent
}

func BuildCAPIPayload(in domain.SendEventInput, testCode, partnerAgent string) ([]byte, error) {
	return json.Marshal(buildPayload(in, testCode, partnerAgent))
}

func buildPayload(in domain.SendEventInput, testCode, partnerAgent string) map[string]any {
	userData := map[string]any{}
	setHashed(userData, "em", util.NormalizeEmail(in.Email))
	setHashed(userData, "ph", util.NormalizePhone(in.Phone))
	setHashed(userData, "fn", util.NormalizeName(in.FirstName))
	setHashed(userData, "ln", util.NormalizeName(in.LastName))
	setHashed(userData, "ct", util.NormalizeCity(in.City))
	setHashed(userData, "st", util.NormalizeStateISO(in.State))
	setHashed(userData, "zp", util.NormalizeZip(in.PostalCode))
	setHashed(userData, "country", util.NormalizeCountry(in.Country))
	setHashed(userData, "external_id", in.ExternalID)

	setIfNotEmpty(userData, "client_ip_address", in.ClientIP)
	setIfNotEmpty(userData, "client_user_agent", in.ClientUserAgent)
	setIfNotEmpty(userData, "fbp", in.FBP)
	setIfNotEmpty(userData, "fbc", in.FBC)
	if in.ActionSource != "business_messaging" {
		setIfNotEmpty(userData, "subscription_id", in.IDWpp)
	}
	setIfNotEmpty(userData, "ctwa_clid", in.CtwaClid)
	setIfNotEmpty(userData, "page_id", in.PageID)

	customData := map[string]any{}
	if in.ContentID != "" {
		rawIDs := strings.Split(in.ContentID, ",")
		ids := make([]string, 0, len(rawIDs))
		for _, id := range rawIDs {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				ids = append(ids, trimmed)
			}
		}
		if len(ids) > 0 {
			customData["content_ids"] = ids
		}
		customData["content_type"] = orDefault(in.ContentType, "product")
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
	if in.NumItems != nil && *in.NumItems > 0 {
		customData["num_items"] = *in.NumItems
	} else if in.AggregatedCount > 1 {
		customData["num_items"] = in.AggregatedCount
	}

	actionSrc := validActionSource(in.ActionSource)
	if in.CtwaClid != "" && actionSrc != "business_messaging" {
		actionSrc = "business_messaging"
	}
	eventName := in.EventName
	if (actionSrc == "business_messaging" || in.CtwaClid != "") && eventName == domain.EventLead {
		eventName = domain.EventLeadSubmitted
	} else if actionSrc == "website" && eventName == domain.EventLeadSubmitted && in.CtwaClid == "" {
		eventName = domain.EventLead
	}

	event := map[string]any{
		"event_name":    eventName,
		"event_time":    in.EventTime,
		"event_id":      in.EventID,
		"action_source": actionSrc,
		"user_data":     userData,
	}
	if in.MessagingChannel != "" {
		event["messaging_channel"] = in.MessagingChannel
	}
	if in.SourceURL != "" {
		event["event_source_url"] = in.SourceURL
	}
	if len(customData) > 0 {
		event["custom_data"] = customData
	}

	payload := map[string]any{"data": []any{event}}
	if testCode != "" {
		payload["test_event_code"] = testCode
	}
	if partnerAgent != "" {
		payload["partner_agent"] = partnerAgent
	}
	return payload
}

