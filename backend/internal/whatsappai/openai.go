package whatsappai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func firstStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func normalizeImageMime(mimePtr *string, fallback string) string {
	raw := ""
	if mimePtr != nil {
		raw = strings.ToLower(strings.TrimSpace(*mimePtr))
	}
	if raw == "" {
		raw = strings.ToLower(strings.TrimSpace(fallback))
	}
	switch {
	case strings.Contains(raw, "png"):
		return "image/png"
	case strings.Contains(raw, "webp"):
		return "image/webp"
	case strings.Contains(raw, "gif"):
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

func normalizeModelName(model string) string {
	switch strings.TrimSpace(model) {
	case "gpt-4o", "gpt-4o-mini":
		return strings.TrimSpace(model)
	default:
		return "gpt-4o-mini"
	}
}

func callOpenAIAnalysis(
	ctx context.Context,
	apiKey string,
	model string,
	systemPrompt string,
	userParts []map[string]any,
) (*AnalysisResult, error) {
	reqPayload := map[string]any{
		"model":           normalizeModelName(model),
		"temperature":     0.1,
		"response_format": responseJSONSchema(),
		"messages": []map[string]any{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userParts,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal openai analysis payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call openai chat completions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rawErr, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("openai returned %d: %s", resp.StatusCode, string(rawErr))
	}

	var parsedResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsedResp); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsedResp.Choices) == 0 || strings.TrimSpace(parsedResp.Choices[0].Message.Content) == "" {
		return nil, fmt.Errorf("empty openai response")
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(parsedResp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("unmarshal structured analysis json: %w", err)
	}
	return &result, nil
}

