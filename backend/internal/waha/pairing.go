package waha

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HealthStatus struct {
	Version       string `json:"version"`
	Engine        string `json:"engine"`
	Tier          string `json:"tier"`
	InstanceCount int    `json:"instance_count"`
}

// HealthCheck valida conectividade e X-Api-Key consultando /api/server/version e /api/sessions?all=true.
func (c *Client) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	var ver struct {
		Version string `json:"version"`
		Engine  string `json:"engine"`
		Tier    string `json:"tier"`
	}
	if err := c.doJSON(ctx, c.httpClient, http.MethodGet, "/api/server/version", nil, &ver); err != nil {
		return nil, err
	}
	var sessions []json.RawMessage
	if err := c.doJSON(ctx, c.httpClient, http.MethodGet, "/api/sessions?all=true", nil, &sessions); err != nil {
		return nil, fmt.Errorf("falha ao listar sessões WAHA: %w", err)
	}
	return &HealthStatus{
		Version:       strings.TrimSpace(ver.Version),
		Engine:        strings.TrimSpace(ver.Engine),
		Tier:          strings.TrimSpace(ver.Tier),
		InstanceCount: len(sessions),
	}, nil
}

type ProfileInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (c *Client) GetProfile(ctx context.Context, sessionName string) (*ProfileInfo, error) {
	path := "/api/" + url.PathEscape(strings.TrimSpace(sessionName)) + "/profile"
	var prof ProfileInfo
	if err := c.doJSON(ctx, c.httpClient, http.MethodGet, path, nil, &prof); err != nil {
		return nil, err
	}
	prof.ID = strings.TrimSpace(prof.ID)
	prof.Name = strings.TrimSpace(prof.Name)
	prof.Picture = strings.TrimSpace(prof.Picture)
	return &prof, nil
}

// GetQRCode obtém a imagem PNG do QR Code (em data URL) via GET /api/{session}/auth/qr.
func (c *Client) GetQRCode(ctx context.Context, sessionName string) (string, error) {
	path := "/api/" + url.PathEscape(strings.TrimSpace(sessionName)) + "/auth/qr"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return "", fmt.Errorf("create waha qr request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("waha get qr: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read waha qr: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", &APIError{StatusCode: res.StatusCode, Body: strings.TrimSpace(string(body))}
	}

	ct := strings.ToLower(strings.TrimSpace(res.Header.Get("Content-Type")))
	if strings.HasPrefix(ct, "image/") {
		mime := strings.Split(ct, ";")[0]
		return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(body), nil
	}

	var qrResp struct {
		Mimetype string `json:"mimetype"`
		Data     string `json:"data"`
	}
	if err := json.Unmarshal(body, &qrResp); err != nil {
		return "", fmt.Errorf("decode waha qr json: %w", err)
	}
	data := strings.TrimSpace(qrResp.Data)
	if data == "" {
		return "", fmt.Errorf("waha retornou QR Code vazio")
	}
	if strings.HasPrefix(data, "data:image") {
		return data, nil
	}
	mime := strings.TrimSpace(qrResp.Mimetype)
	if mime == "" {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + data, nil
}

// RequestPairingCode solicita código de pareamento por telefone via POST /api/{session}/auth/request-code.
func (c *Client) RequestPairingCode(ctx context.Context, sessionName, phone string) (string, error) {
	cleanPhone := digitsOnly(phone)
	if len(cleanPhone) < 10 {
		return "", fmt.Errorf("telefone inválido para pareamento")
	}
	path := "/api/" + url.PathEscape(strings.TrimSpace(sessionName)) + "/auth/request-code"
	var resp struct {
		Code string `json:"code"`
	}
	if err := c.doJSON(ctx, c.httpClient, http.MethodPost, path, map[string]string{
		"phoneNumber": cleanPhone,
	}, &resp); err != nil {
		return "", err
	}
	code := strings.TrimSpace(resp.Code)
	if code == "" {
		return "", fmt.Errorf("waha não retornou código de pareamento")
	}
	return code, nil
}

// ResolveLID traduz um identificador @lid para o número de telefone via GET /api/{session}/lids/{lid}.
func (c *Client) ResolveLID(ctx context.Context, sessionName, lid string) string {
	clean := strings.TrimSpace(lid)
	if clean == "" || !c.Configured() {
		return ""
	}
	if !strings.Contains(clean, "@") {
		clean += "@lid"
	}
	path := "/api/" + url.PathEscape(strings.TrimSpace(sessionName)) + "/lids/" + url.PathEscape(clean)
	var resp struct {
		LID string `json:"lid"`
		PN  string `json:"pn"`
	}
	if err := c.doJSON(ctx, c.httpClient, http.MethodGet, path, nil, &resp); err != nil {
		return ""
	}
	return strings.TrimSpace(resp.PN)
}

// GetContactProfilePicture busca a foto de perfil de um contato/lead via GET /api/contacts/profile-picture.
func (c *Client) GetContactProfilePicture(ctx context.Context, sessionName, contactID string) string {
	if !c.Configured() || strings.TrimSpace(contactID) == "" {
		return ""
	}
	q := url.Values{}
	q.Set("contactId", strings.TrimSpace(contactID))
	q.Set("session", strings.TrimSpace(sessionName))
	var resp struct {
		ProfilePictureURL string `json:"profilePictureURL"`
	}
	if err := c.doJSON(ctx, c.httpClient, http.MethodGet, "/api/contacts/profile-picture?"+q.Encode(), nil, &resp); err != nil {
		return ""
	}
	return strings.TrimSpace(resp.ProfilePictureURL)
}

// SendText envia mensagem de texto simples via POST /api/sendText (usado nos alertas de Regras de Campanhas).
func (c *Client) SendText(ctx context.Context, sessionName, phone, text string) error {
	cleanPhone := digitsOnly(phone)
	if cleanPhone == "" {
		return fmt.Errorf("número de WhatsApp inválido")
	}
	cleanText := strings.TrimSpace(text)
	if cleanText == "" {
		return fmt.Errorf("mensagem vazia")
	}
	req := map[string]string{
		"session": strings.TrimSpace(sessionName),
		"chatId":  cleanPhone + "@c.us",
		"text":    cleanText,
	}
	return c.doJSON(ctx, c.httpClient, http.MethodPost, "/api/sendText", req, nil)
}

func (c *Client) LogoutSession(ctx context.Context, sessionName string) error {
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(sessionName)) + "/logout"
	err := c.doJSON(ctx, c.httpClient, http.MethodPost, path, map[string]any{}, nil)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
}

func (c *Client) DeleteSession(ctx context.Context, sessionName string) error {
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(sessionName))
	err := c.doJSON(ctx, c.httpClient, http.MethodDelete, path, nil, nil)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
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
