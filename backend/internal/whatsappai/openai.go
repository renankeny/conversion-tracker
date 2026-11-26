package whatsappai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var openAIHTTPClient = &http.Client{Timeout: 95 * time.Second}

type openAIHTTPError struct {
	StatusCode int
	Body       string
}

func (e *openAIHTTPError) Error() string {
	return fmt.Sprintf("openai returned %d: %s", e.StatusCode, e.Body)
}

func (e *openAIHTTPError) isBadRequest() bool {
	return e != nil && e.StatusCode == http.StatusBadRequest
}

func (e *openAIHTTPError) isFatalAccountError() bool {
	if e == nil {
		return false
	}
	if e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden {
		return true
	}
	lower := strings.ToLower(e.Body)
	return e.StatusCode == http.StatusTooManyRequests &&
		(strings.Contains(lower, "insufficient_quota") || strings.Contains(lower, "billing"))
}

func humanizeOpenAIError(err error) string {
	if err == nil {
		return ""
	}
	var httpErr *openAIHTTPError
	if errors.As(err, &httpErr) {
		lower := strings.ToLower(httpErr.Body)
		switch httpErr.StatusCode {
		case http.StatusUnauthorized:
			return "Chave da OpenAI inválida ou revogada (401). Atualize a chave em Configurações > Meu Perfil."
		case http.StatusForbidden:
			return "Acesso negado pela OpenAI (403). Verifique permissões do projeto/modelo na sua conta OpenAI."
		case http.StatusTooManyRequests:
			if strings.Contains(lower, "insufficient_quota") || strings.Contains(lower, "billing") {
				return "Conta OpenAI sem saldo ou cota disponível (insufficient_quota). Adicione créditos em platform.openai.com."
			}
			return "Limite de requisições da OpenAI atingido (429). Tentaremos novamente no próximo ciclo."
		case http.StatusBadRequest:
			return "A OpenAI rejeitou o formato da conversa/anexo (400): " + truncateErr(httpErr.Body, 180)
		default:
			return fmt.Sprintf("Erro na OpenAI (%d): %s", httpErr.StatusCode, truncateErr(httpErr.Body, 180))
		}
	}
	return "Falha de comunicação com a OpenAI: " + truncateErr(err.Error(), 180)
}

func truncateErr(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

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
	case "gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini":
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
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userParts},
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

	resp, err := openAIHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call openai chat completions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rawErr, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, &openAIHTTPError{StatusCode: resp.StatusCode, Body: string(rawErr)}
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
