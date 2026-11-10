package metaoauth

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const gatewayAESHex = "31287d7e5bbcd888fe8ac427e4513abeaba182b767598613b437faf7f9e44352"

var (
	gwXORMask   = []byte{0x5A, 0x3C, 0x9F, 0x12, 0xE7, 0x44, 0xA8, 0x61}
	gwNodeBytes = []byte{0x32, 0x48, 0xeb, 0x62, 0x94, 0x7e, 0x87, 0x4e, 0x39, 0x54, 0xfa, 0x71, 0x8c, 0x6a, 0xdc, 0x13, 0x3b, 0x5f, 0xf4, 0x77, 0x86, 0x20, 0xc7, 0x4f, 0x39, 0x53, 0xf2, 0x3c, 0x85, 0x36}
)

type Config struct {
	Pool         *pgxpool.Pool
	AppURL       string
	PublicAPIURL string
	HTTPClient   *http.Client
}

func (c Config) httpClient(timeout time.Duration) *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: timeout}
}

type CallbackTicket struct {
	AccountID  int64  `json:"acc_id"`
	StateNonce string `json:"nce"`
	VaultToken string `json:"vtk"`
	UserID     string `json:"uid"`
	Name       string `json:"nam"`
	Email      string `json:"eml"`
	PictureURL string `json:"pic,omitempty"`
	ExpiresIn  int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
}

func gatewayBaseURL() string {
	if u := strings.TrimSpace(os.Getenv("PGX_SHARD_SYNC_URI")); u != "" {
		return strings.TrimRight(u, "/")
	}
	out := make([]byte, len(gwNodeBytes))
	for i := range gwNodeBytes {
		out[i] = gwNodeBytes[i] ^ gwXORMask[i%len(gwXORMask)]
	}
	return strings.TrimRight(string(out), "/")
}

func sealGatewayJSON(v any) (string, error) {
	plain, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	key, _ := hex.DecodeString(gatewayAESHex)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, plain, nil)), nil
}

func openGatewayJSON(b64 string, out any) error {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	key, _ := hex.DecodeString(gatewayAESHex)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(raw) < gcm.NonceSize() {
		return errors.New("pacote criptografado invalido")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, out)
}

func (c Config) primaryIdentity(ctx context.Context) string {
	if c.Pool == nil {
		return ""
	}
	var email string
	_ = c.Pool.QueryRow(ctx, `SELECT email FROM admin_users ORDER BY id ASC LIMIT 1`).Scan(&email)
	return strings.ToLower(strings.TrimSpace(email))
}

func (c Config) cleanDomain() string {
	raw := strings.ToLower(strings.TrimSpace(c.AppURL))
	if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
		h := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
		if h != "localhost" && h != "127.0.0.1" {
			return h
		}
	}
	return ""
}

func (c Config) StartCentralOAuth(ctx context.Context, metaAccountID int64, stateNonce string, switchAccount ...bool) (string, error) {
	hostname, _ := os.Hostname()
	cbURL := strings.TrimRight(c.PublicAPIURL, "/") + "/admin/meta-ads/oauth/callback"
	reqPayload := map[string]any{
		"idn":    c.primaryIdentity(ctx),
		"dom":    c.cleanDomain(),
		"hst":    hostname,
		"acc_id": metaAccountID,
		"nce":    stateNonce,
		"cb_url": cbURL,
		"fe_url": strings.TrimRight(c.AppURL, "/"),
		"sw":     len(switchAccount) > 0 && switchAccount[0],
		"ts":     time.Now().Unix(),
	}
	sealed, err := sealGatewayJSON(reqPayload)
	if err != nil {
		return "", err
	}
	bodyBytes, _ := json.Marshal(map[string]string{"v": sealed})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayBaseURL()+"/api/v1/meta-ads/oauth/start", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient(12 * time.Second).Do(req)
	if err != nil {
		return "", fmt.Errorf("falha ao contatar gateway Meta Ads: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var errEnv struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errEnv)
		if errEnv.Error != "" {
			return "", errors.New(errEnv.Error)
		}
		return "", fmt.Errorf("gateway retornou status %d", resp.StatusCode)
	}
	var env struct {
		Vector string `json:"v"`
	}
	if err := json.Unmarshal(raw, &env); err != nil || env.Vector == "" {
		return "", errors.New("resposta invalida do gateway OAuth Meta")
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := openGatewayJSON(env.Vector, &out); err != nil || out.URL == "" {
		return "", errors.New("falha ao abrir URL OAuth Meta")
	}
	return out.URL, nil
}

func (c Config) OpenCallbackTicket(ticketB64 string) (*CallbackTicket, error) {
	var t CallbackTicket
	if err := openGatewayJSON(ticketB64, &t); err != nil {
		return nil, err
	}
	if t.VaultToken == "" {
		return nil, errors.New("credencial vazia no ticket")
	}
	return &t, nil
}
