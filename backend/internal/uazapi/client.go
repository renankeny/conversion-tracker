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

// Client é o cliente HTTP para servidores compatíveis com uazapiGO (API Premium e Uazapi).
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type InstanceStatus struct {
	Connected     bool
	LoggedIn      bool
	Status        string
	OwnerJID      string
	ProfileName   string
	ProfilePicURL string
	Qrcode        string
}

func New(baseURL, token string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &Client{
		baseURL: baseURL,
		token:   strings.TrimSpace(token),
		http: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.baseURL != "" && c.token != ""
}

// ConfigureWebhook configura o webhook da instância no modo simples da API Premium / Uazapi,
// ativando eventos de mensagens e conexão e bloqueando mensagens enviadas por API e grupos.
func (c *Client) ConfigureWebhook(ctx context.Context, webhookURL, _ string) error {
	if !c.Configured() {
		return fmt.Errorf("uazapi client not configured")
	}
	payload := map[string]any{
		"enabled": true,
		"url":     webhookURL,
		"events": []string{
			"messages",
			"connection",
		},
		"excludeMessages": []string{
			"wasSentByApi",
			"isGroupYes",
		},
		"addUrlEvents":        false,
		"addUrlTypesMessages": false,
	}

	_, err := c.doJSON(ctx, http.MethodPost, "/webhook", payload)
	return err
}

func (c *Client) GetStatus(ctx context.Context) (*InstanceStatus, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("uazapi client not configured")
	}
	body, err := c.doJSON(ctx, http.MethodGet, "/instance/status", nil)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Instance struct {
			Name          string `json:"name"`
			Status        string `json:"status"`
			Owner         string `json:"owner"`
			ProfileName   string `json:"profileName"`
			ProfilePicURL string `json:"profilePicUrl"`
			Qrcode        string `json:"qrcode"`
		} `json:"instance"`
		Status struct {
			Connected bool            `json:"connected"`
			LoggedIn  bool            `json:"loggedIn"`
			JID       json.RawMessage `json:"jid"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode uazapi status: %w", err)
	}

	instState := strings.ToLower(strings.TrimSpace(raw.Instance.Status))
	loggedIn := raw.Status.LoggedIn || instState == "connected"
	connected := (raw.Status.Connected && loggedIn) || instState == "connected"

	ownerJID := parseUazapiJID(raw.Status.JID, raw.Instance.Owner)
	profName := strings.TrimSpace(raw.Instance.ProfileName)
	if profName == "" {
		profName = strings.TrimSpace(raw.Instance.Name)
	}

	return &InstanceStatus{
		Connected:     connected,
		LoggedIn:      loggedIn,
		Status:        instState,
		OwnerJID:      ownerJID,
		ProfileName:   profName,
		ProfilePicURL: strings.TrimSpace(raw.Instance.ProfilePicURL),
		Qrcode:        strings.TrimSpace(raw.Instance.Qrcode),
	}, nil
}

func (c *Client) Connect(ctx context.Context) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("uazapi client not configured")
	}
	body, err := c.doJSON(ctx, http.MethodPost, "/instance/connect", map[string]any{})
	if err != nil {
		return "", err
	}
	var raw struct {
		Qrcode   string `json:"qrcode"`
		Instance struct {
			Qrcode string `json:"qrcode"`
		} `json:"instance"`
	}
	_ = json.Unmarshal(body, &raw)
	if raw.Instance.Qrcode != "" {
		return raw.Instance.Qrcode, nil
	}
	return raw.Qrcode, nil
}

func (c *Client) Disconnect(ctx context.Context) error {
	if !c.Configured() {
		return nil
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/instance/disconnect", map[string]any{})
	return err
}

func (c *Client) EditLead(ctx context.Context, chatID string, fields map[string]any) error {
	if !c.Configured() {
		return fmt.Errorf("uazapi client not configured")
	}
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["id"] = chatID
	_, err := c.doJSON(ctx, http.MethodPost, "/chat/editLead", fields)
	return err
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any) ([]byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

func parseUazapiJID(raw json.RawMessage, fallbackOwner string) string {
	if len(raw) > 0 && string(raw) != "null" {
		var s string
		if json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s) != "" {
			s = strings.TrimSpace(s)
			if !strings.Contains(s, "@") {
				return s + "@s.whatsapp.net"
			}
			return s
		}
		var obj struct {
			User   string `json:"user"`
			Server string `json:"server"`
		}
		if json.Unmarshal(raw, &obj) == nil && strings.TrimSpace(obj.User) != "" {
			srv := strings.TrimSpace(obj.Server)
			if srv == "" {
				srv = "s.whatsapp.net"
			}
			return strings.TrimSpace(obj.User) + "@" + srv
		}
	}
	owner := strings.TrimSpace(fallbackOwner)
	if owner == "" {
		return ""
	}
	if !strings.Contains(owner, "@") {
		return owner + "@s.whatsapp.net"
	}
	return owner
}
