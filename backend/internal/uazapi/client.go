package uazapi

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

// Client é o cliente HTTP para a uazAPI Premium.
// Responsável por persistir dados de volta no CRM da uazAPI (lead_field01..12, lead_email).
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New cria um novo cliente uazAPI.
func New(baseURL, token string) *Client {
	// Remove trailing slash da baseURL se presente
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// EditLead atualiza campos do lead no CRM da uazAPI.
// O campo "id" no payload deve conter o wa_chat_id.
// Outros campos suportados: lead_email, lead_field01..12, etc.
//
// Exemplo de payload:
//
//	{
//	  "id": "5511999999999@c.us",
//	  "lead_field01": "Protocolo#12345",
//	  "lead_email": "cliente@example.com",
//	  "lead_field10": "Cj0KCQ...",
//	  "lead_field11": "1234567890",
//	  "lead_field12": "fb_messenger"
//	}
func (c *Client) EditLead(ctx context.Context, chatID string, fields map[string]any) error {
	// Garante que o campo "id" esteja presente
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["id"] = chatID

	payload, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := c.baseURL + "/chat/editLead"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	// Lê o body da resposta para logar em caso de erro
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("uazapi returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ConfigureWebhook configura o webhook da instância na uazAPI.
// Usa o "modo simples" da API que gerencia automaticamente um único webhook.
//
// Parâmetros:
//   - webhookURL: URL completa do TrackFind que receberá os eventos
//   - secret: Webhook secret para validação (será enviado como header)
//
// O webhook será configurado para receber eventos de "messages" e excluir
// mensagens enviadas pela própria API (evita loops).
func (c *Client) ConfigureWebhook(ctx context.Context, webhookURL, secret string) error {
	payload := map[string]any{
		"enabled": true,
		"url":     webhookURL,
		"events": []string{
			"messages", // Mensagens recebidas
		},
		"excludeMessages": []string{
			"wasSentByApi", // CRÍTICO: evita loops infinitos
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := c.baseURL + "/webhook"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("uazapi returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetWebhooks retorna a configuração atual de webhooks da instância.
func (c *Client) GetWebhooks(ctx context.Context) ([]map[string]any, error) {
	url := c.baseURL + "/webhook"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("uazapi returned status %d: %s", resp.StatusCode, string(body))
	}

	var webhooks []map[string]any
	if err := json.Unmarshal(body, &webhooks); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return webhooks, nil
}
