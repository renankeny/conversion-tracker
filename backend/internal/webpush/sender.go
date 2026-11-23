package webpush

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Sender struct {
	httpClient *http.Client
}

func NewSender() *Sender {
	return &Sender{
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

type SubscriptionTarget struct {
	Endpoint   string
	P256dh     string
	AuthSecret string
}

// ValidateEndpoint garante que o endpoint de push usa HTTPS (exceto localhost em dev).
func ValidateEndpoint(rawEndpoint string) error {
	u, err := url.Parse(strings.TrimSpace(rawEndpoint))
	if err != nil || u.Host == "" {
		return errors.New("endpoint de Web Push inválido")
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && (strings.HasPrefix(u.Host, "localhost") || strings.HasPrefix(u.Host, "127.0.0.1")) {
		return nil
	}
	return errors.New("endpoint de Web Push exige HTTPS")
}

// Send criptografa e envia a notificação Web Push.
// Retorna gone=true quando o navegador revogou/desinstalou a inscrição (404 / 410).
func (s *Sender) Send(ctx context.Context, kp *KeyPair, sub SubscriptionTarget, payload []byte) (gone bool, err error) {
	if err := ValidateEndpoint(sub.Endpoint); err != nil {
		return false, err
	}
	body, err := EncryptAES128GCM(payload, sub.P256dh, sub.AuthSecret)
	if err != nil {
		return false, err
	}
	authHdr, err := BuildVAPIDAuthHeader(sub.Endpoint, kp)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("TTL", "86400")
	req.Header.Set("Urgency", "high")
	req.Header.Set("Authorization", authHdr)

	cli := http.DefaultClient
	if s != nil && s.httpClient != nil {
		cli = s.httpClient
	}
	resp, err := cli.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return true, fmt.Errorf("push subscription expired (status %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("push gateway status %d", resp.StatusCode)
	}
	return false, nil
}
