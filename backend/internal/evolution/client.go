package evolution

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

var DefaultSubscribedEvents = []string{"MESSAGE", "SEND_MESSAGE", "CONNECTION", "QRCODE"}

type Client struct {
	baseURL     string
	adminAPIKey string
	httpClient  *http.Client
	mediaClient *http.Client
}

func NewClient(baseURL, adminAPIKey string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		adminAPIKey: strings.TrimSpace(adminAPIKey),
		httpClient:  &http.Client{Timeout: 12 * time.Second},
		mediaClient: &http.Client{Timeout: 40 * time.Second},
	}
}

func (c *Client) doWithKey(ctx context.Context, cli *http.Client, apiKey, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("apikey", apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("evolution http %s %s: %w", method, path, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("evolution api error (status %d): %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

type HealthStatus struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	Database      string `json:"database"`
	InstanceCount int    `json:"instance_count"`
}

// HealthCheck valida conectividade (/health) e autenticidade da GLOBAL_API_KEY (/instance/all).
func (c *Client) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	var health HealthStatus
	if err := c.doWithKey(ctx, c.httpClient, "", http.MethodGet, "/health", nil, &health); err != nil {
		return nil, err
	}
	var all struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := c.doWithKey(ctx, c.httpClient, c.adminAPIKey, http.MethodGet, "/instance/all", nil, &all); err != nil {
		return nil, fmt.Errorf("falha na validação da API Key global: %w", err)
	}
	health.InstanceCount = len(all.Data)
	return &health, nil
}

type AdvancedSettings struct {
	AlwaysOnline  bool   `json:"alwaysOnline"`
	RejectCall    bool   `json:"rejectCall"`
	MsgRejectCall string `json:"msgRejectCall"`
	ReadMessages  bool   `json:"readMessages"`
	IgnoreGroups  bool   `json:"ignoreGroups"`
	IgnoreStatus  bool   `json:"ignoreStatus"`
}

type CreateInstanceRequest struct {
	Name             string           `json:"name"`
	Token            string           `json:"token"`
	AdvancedSettings AdvancedSettings `json:"advancedSettings"`
}

type CreateInstanceResponse struct {
	Data struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Token         string `json:"token"`
		Qrcode        string `json:"qrcode"`
		Connected     bool   `json:"connected"`
		JID           string `json:"jid"`
		ProfileName   string `json:"profileName"`
		ProfilePicURL string `json:"profilePicUrl"`
	} `json:"data"`
}

func (c *Client) CreateInstance(ctx context.Context, name, instanceToken string) (*CreateInstanceResponse, error) {
	req := CreateInstanceRequest{
		Name:  name,
		Token: instanceToken,
		AdvancedSettings: AdvancedSettings{
			AlwaysOnline: false,
			RejectCall:   false,
			ReadMessages: false,
			IgnoreGroups: true,
			IgnoreStatus: true,
		},
	}
	var resp CreateInstanceResponse
	if err := c.doWithKey(ctx, c.httpClient, c.adminAPIKey, http.MethodPost, "/instance/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type ConnectInstanceRequest struct {
	WebhookURL  string   `json:"webhookUrl"`
	Clear       bool     `json:"clearWebhook"`
	Subscribe   []string `json:"subscribe"`
	Immediate   bool     `json:"immediate"`
}

type ConnectInstanceResponse struct {
	Data struct {
		JID         string `json:"jid"`
		WebhookURL  string `json:"webhookUrl"`
		EventString string `json:"eventString"`
	} `json:"data"`
}

// ConnectInstance persiste webhookUrl + eventos e inicia (ou faz hot-reload sem derrubar sessão) o cliente whatsmeow.
func (c *Client) ConnectInstance(ctx context.Context, instanceToken, webhookURL string) (*ConnectInstanceResponse, error) {
	req := ConnectInstanceRequest{
		WebhookURL: webhookURL,
		Clear:      false,
		Subscribe:  DefaultSubscribedEvents,
		Immediate:  false,
	}
	var resp ConnectInstanceResponse
	if err := c.doWithKey(ctx, c.httpClient, instanceToken, http.MethodPost, "/instance/connect", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
