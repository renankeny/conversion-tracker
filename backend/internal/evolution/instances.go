package evolution

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type QrData struct {
	Qrcode string `json:"qrcode"`
	Code   string `json:"code"`
}

func (c *Client) GetQrCode(ctx context.Context, instanceToken string) (*QrData, error) {
	var raw struct {
		Data struct {
			QrcodeLower string `json:"qrcode"`
			QrcodeUpper string `json:"Qrcode"`
			CodeLower   string `json:"code"`
			CodeUpper   string `json:"Code"`
		} `json:"data"`
	}
	if err := c.doWithKey(ctx, c.httpClient, instanceToken, http.MethodGet, "/instance/qr", nil, &raw); err != nil {
		return nil, err
	}
	qr := raw.Data.QrcodeLower
	if qr == "" {
		qr = raw.Data.QrcodeUpper
	}
	code := raw.Data.CodeLower
	if code == "" {
		code = raw.Data.CodeUpper
	}
	return &QrData{Qrcode: qr, Code: code}, nil
}

func (c *Client) PairPhone(ctx context.Context, instanceToken, phone string) (string, error) {
	cleanPhone := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	if len(cleanPhone) < 10 {
		return "", fmt.Errorf("telefone inválido para pareamento")
	}
	var resp struct {
		Data struct {
			PairingCodeUpper string `json:"PairingCode"`
			PairingCodeLower string `json:"pairingCode"`
			Code             string `json:"code"`
		} `json:"data"`
	}
	body := map[string]string{"phone": cleanPhone}
	if err := c.doWithKey(ctx, c.httpClient, instanceToken, http.MethodPost, "/instance/pair", body, &resp); err != nil {
		return "", err
	}
	code := resp.Data.PairingCodeUpper
	if code == "" {
		code = resp.Data.PairingCodeLower
	}
	if code == "" {
		code = resp.Data.Code
	}
	return code, nil
}

type StatusData struct {
	Connected bool   `json:"connected"`
	LoggedIn  bool   `json:"loggedIn"`
	MyJID     string `json:"myJid"`
	Name      string `json:"name"`
}

func parseRawJID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return strings.TrimSpace(str)
	}
	var obj struct {
		User   string `json:"User"`
		Server string `json:"Server"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.User != "" {
		srv := obj.Server
		if srv == "" {
			srv = "s.whatsapp.net"
		}
		return obj.User + "@" + srv
	}
	return ""
}

func (c *Client) GetStatus(ctx context.Context, instanceToken string) (*StatusData, error) {
	var resp struct {
		Data struct {
			ConnectedLower bool            `json:"connected"`
			ConnectedUpper bool            `json:"Connected"`
			LoggedInLower  bool            `json:"loggedIn"`
			LoggedInUpper  bool            `json:"LoggedIn"`
			MyJID          json.RawMessage `json:"myJid"`
			Name           string          `json:"name"`
		} `json:"data"`
	}
	if err := c.doWithKey(ctx, c.httpClient, instanceToken, http.MethodGet, "/instance/status", nil, &resp); err != nil {
		return nil, err
	}
	return &StatusData{
		Connected: resp.Data.ConnectedLower || resp.Data.ConnectedUpper,
		LoggedIn:  resp.Data.LoggedInLower || resp.Data.LoggedInUpper,
		MyJID:     parseRawJID(resp.Data.MyJID),
		Name:      strings.TrimSpace(resp.Data.Name),
	}, nil
}

type InstanceInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	JID           string `json:"jid"`
	Qrcode        string `json:"qrcode"`
	Connected     bool   `json:"connected"`
	ProfileName   string `json:"profileName"`
	ProfilePicURL string `json:"profilePicUrl"`
}

func (c *Client) GetInfo(ctx context.Context, idOrName string) (*InstanceInfo, error) {
	var resp struct {
		Data InstanceInfo `json:"data"`
	}
	path := "/instance/info/" + url.PathEscape(idOrName)
	if err := c.doWithKey(ctx, c.httpClient, c.adminAPIKey, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetUserAvatar busca a URL da foto de perfil via POST /user/avatar na Evolution Go.
func (c *Client) GetUserAvatar(ctx context.Context, instanceToken, phoneOrJID string) string {
	num := strings.TrimSpace(strings.Split(phoneOrJID, "@")[0])
	num = strings.Split(num, ":")[0]
	if num == "" {
		return ""
	}
	var resp struct {
		Data struct {
			URLUpper string `json:"URL"`
			URLLower string `json:"url"`
		} `json:"data"`
	}
	body := map[string]any{"number": num, "preview": true}
	if err := c.doWithKey(ctx, c.httpClient, instanceToken, http.MethodPost, "/user/avatar", body, &resp); err != nil {
		return ""
	}
	if resp.Data.URLUpper != "" {
		return strings.TrimSpace(resp.Data.URLUpper)
	}
	return strings.TrimSpace(resp.Data.URLLower)
}

func (c *Client) ResolveInstanceUUID(ctx context.Context, instanceName string) string {
	var listResp struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := c.doWithKey(ctx, c.httpClient, c.adminAPIKey, http.MethodGet, "/instance/all", nil, &listResp); err != nil {
		return ""
	}
	for _, inst := range listResp.Data {
		if inst.Name == instanceName {
			return inst.ID
		}
	}
	return ""
}

func (c *Client) DisconnectInstance(ctx context.Context, instanceToken string) error {
	return c.doWithKey(ctx, c.httpClient, instanceToken, http.MethodPost, "/instance/disconnect", map[string]any{}, nil)
}

func (c *Client) LogoutInstance(ctx context.Context, instanceToken string) error {
	return c.doWithKey(ctx, c.httpClient, instanceToken, http.MethodDelete, "/instance/logout", nil, nil)
}

func (c *Client) DeleteInstance(ctx context.Context, evolutionUUID, instanceName string) error {
	uuid := strings.TrimSpace(evolutionUUID)
	if uuid == "" && instanceName != "" {
		uuid = c.ResolveInstanceUUID(ctx, instanceName)
	}
	if uuid == "" {
		return fmt.Errorf("evolution_uuid não encontrado para instância %q", instanceName)
	}
	return c.doWithKey(ctx, c.httpClient, c.adminAPIKey, http.MethodDelete, "/instance/delete/"+url.PathEscape(uuid), nil, nil)
}
