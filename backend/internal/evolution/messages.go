package evolution

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type sendTextRequest struct {
	Number string `json:"number"`
	Text   string `json:"text"`
}

// SendText envia uma mensagem de texto simples via Evolution Go (POST /send/text)
// utilizando o token da instância conectada.
func (c *Client) SendText(ctx context.Context, instanceToken, number, text string) error {
	cleanNumber := digitsOnly(number)
	if cleanNumber == "" {
		return fmt.Errorf("número de WhatsApp inválido")
	}
	cleanText := strings.TrimSpace(text)
	if cleanText == "" {
		return fmt.Errorf("mensagem vazia")
	}
	req := sendTextRequest{
		Number: cleanNumber,
		Text:   cleanText,
	}
	return c.doWithKey(ctx, c.httpClient, instanceToken, http.MethodPost, "/send/text", req, nil)
}

func digitsOnly(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
