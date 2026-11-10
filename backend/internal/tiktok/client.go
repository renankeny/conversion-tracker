// Package tiktok encapsula chamadas à TikTok Events API V1.3.
package tiktok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

const (
	baseURL  = "https://business-api.tiktok.com/open_api/v1.3/event/track/"
	eventSrc = "web"
)

type Client struct {
	partnerName string
	http        *http.Client
}

func New(partnerName string) *Client {
	return &Client{
		partnerName: partnerName,
		http:        &http.Client{Timeout: 10 * time.Second},
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
	tiktokEvent := mapEventName(in.EventName)
	if tiktokEvent == "" {
		return nil
	}
	results := make([]SendResult, 0, len(pixel.TikTokPixelIDs))
	for i, pixelID := range pixel.TikTokPixelIDs {
		if skip != nil && skip(pixelID) {
			continue
		}
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
		res.StatusCode = http.StatusBadRequest
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

	if resp.StatusCode >= 300 {
		res.Err = fmt.Errorf("tiktok returned %d: %s", resp.StatusCode, res.Body)
		return res
	}
	if mappedStatus, logicalErr := extractTikTokError(respBody); logicalErr != "" {
		res.StatusCode = mappedStatus
		res.Err = fmt.Errorf("tiktok logical error: %s", logicalErr)
	}
	return res
}

func extractTikTokError(body []byte) (int, string) {
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.Code == 0 {
		return http.StatusOK, ""
	}
	status := mapTikTokCodeToHTTP(out.Code)
	return status, fmt.Sprintf("code=%d message=%s", out.Code, out.Message)
}

func mapTikTokCodeToHTTP(code int) int {
	switch {
	case code == 40016 || (code >= 42900 && code < 43000):
		return http.StatusTooManyRequests
	case code >= 40100 && code < 40200:
		return http.StatusUnauthorized
	case code >= 40300 && code < 40400:
		return http.StatusForbidden
	case code >= 40000 && code < 41000:
		return http.StatusBadRequest
	default:
		return http.StatusBadGateway
	}
}
