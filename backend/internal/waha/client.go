package waha

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	DefaultWebhookEvents = []string{"message.any", "session.status"}
	ErrSessionNotFound   = errors.New("waha session not found")
)

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("waha api error (status %d): %s", e.StatusCode, e.Body)
}

type Client struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	mediaClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:      strings.TrimSpace(apiKey),
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		mediaClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.baseURL != ""
}

func (c *Client) doJSON(ctx context.Context, cli *http.Client, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal waha request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create waha request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("waha http %s %s: %w", method, path, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return &APIError{StatusCode: res.StatusCode, Body: strings.TrimSpace(string(b))}
	}
	if out != nil {
		if err := json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(out); err != nil {
			return fmt.Errorf("decode waha response: %w", err)
		}
	}
	return nil
}

type SessionMe struct {
	ID       string `json:"id"`
	LID      string `json:"lid"`
	JID      string `json:"jid"`
	PushName string `json:"pushName"`
}

type SessionConfig struct {
	Webhooks []WebhookConfig `json:"webhooks,omitempty"`
	Ignore   *IgnoreConfig   `json:"ignore,omitempty"`
}

type WebhookConfig struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type IgnoreConfig struct {
	Status    bool `json:"status"`
	Groups    bool `json:"groups"`
	Channels  bool `json:"channels"`
	Broadcast bool `json:"broadcast"`
}

type SessionInfo struct {
	Name   string         `json:"name"`
	Status string         `json:"status"`
	Me     *SessionMe     `json:"me,omitempty"`
	Config *SessionConfig `json:"config,omitempty"`
}

func defaultSessionConfig(webhookURL string) SessionConfig {
	cfg := SessionConfig{
		Ignore: &IgnoreConfig{
			Status:    true,
			Groups:    true,
			Channels:  true,
			Broadcast: true,
		},
	}
	if strings.TrimSpace(webhookURL) != "" {
		cfg.Webhooks = []WebhookConfig{{
			URL:    strings.TrimSpace(webhookURL),
			Events: DefaultWebhookEvents,
		}}
	}
	return cfg
}

// CreateAndStartSession cria e inicia uma sessão WAHA já com webhook e filtros de grupo/status.
func (c *Client) CreateAndStartSession(ctx context.Context, sessionName, webhookURL string) (*SessionInfo, error) {
	req := map[string]any{
		"name":   strings.TrimSpace(sessionName),
		"start":  true,
		"config": defaultSessionConfig(webhookURL),
	}
	var info SessionInfo
	err := c.doJSON(ctx, c.httpClient, http.MethodPost, "/api/sessions", req, &info)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusConflict || apiErr.StatusCode == http.StatusUnprocessableEntity) {
			return c.EnsureSessionRunning(ctx, sessionName, webhookURL)
		}
		return nil, err
	}
	return &info, nil
}

func (c *Client) GetSession(ctx context.Context, sessionName string) (*SessionInfo, error) {
	var info SessionInfo
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(sessionName))
	if err := c.doJSON(ctx, c.httpClient, http.MethodGet, path, nil, &info); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return &info, nil
}

// EnsureSessionRunning garante que a sessão exista na WAHA com o webhook configurado e em execução.
func (c *Client) EnsureSessionRunning(ctx context.Context, sessionName, webhookURL string) (*SessionInfo, error) {
	name := strings.TrimSpace(sessionName)
	info, err := c.GetSession(ctx, name)
	if errors.Is(err, ErrSessionNotFound) {
		return c.CreateAndStartSession(ctx, name, webhookURL)
	}
	if err != nil {
		return nil, err
	}

	if needsWebhookMerge(info.Config, webhookURL) {
		merged := mergeSessionConfig(info.Config, webhookURL)
		var updated SessionInfo
		path := "/api/sessions/" + url.PathEscape(name)
		if uerr := c.doJSON(ctx, c.httpClient, http.MethodPut, path, map[string]any{"config": merged}, &updated); uerr == nil && updated.Name != "" {
			info = &updated
		}
	}

	status := strings.ToUpper(strings.TrimSpace(info.Status))
	if status == "STOPPED" || status == "FAILED" {
		var started SessionInfo
		path := "/api/sessions/" + url.PathEscape(name) + "/start"
		if serr := c.doJSON(ctx, c.httpClient, http.MethodPost, path, nil, &started); serr != nil {
			return nil, serr
		}
		return &started, nil
	}
	return info, nil
}

func needsWebhookMerge(cfg *SessionConfig, webhookURL string) bool {
	target := strings.TrimSpace(webhookURL)
	if target == "" {
		return false
	}
	if cfg == nil {
		return true
	}
	for _, wh := range cfg.Webhooks {
		if strings.TrimSpace(wh.URL) == target {
			return false
		}
	}
	return true
}

func mergeSessionConfig(cur *SessionConfig, webhookURL string) SessionConfig {
	out := defaultSessionConfig(webhookURL)
	if cur == nil {
		return out
	}
	target := strings.TrimSpace(webhookURL)
	merged := make([]WebhookConfig, 0, len(cur.Webhooks)+1)
	for _, wh := range cur.Webhooks {
		if strings.TrimSpace(wh.URL) != "" && strings.TrimSpace(wh.URL) != target {
			merged = append(merged, wh)
		}
	}
	if target != "" {
		merged = append(merged, WebhookConfig{URL: target, Events: DefaultWebhookEvents})
	}
	out.Webhooks = merged
	return out
}
