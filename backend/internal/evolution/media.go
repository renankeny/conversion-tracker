package evolution

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DownloadMedia solicita à Evolution Go a descriptografia sob demanda de uma mídia do WhatsApp
// via POST /message/downloadmedia (autenticado com apikey = instance.token).
func (c *Client) DownloadMedia(ctx context.Context, instanceToken string, rawMessageJSON json.RawMessage, maxBytes int64) ([]byte, string, error) {
	body := map[string]json.RawMessage{
		"message": rawMessageJSON,
	}
	var resp struct {
		Data struct {
			Base64 string `json:"base64"`
		} `json:"data"`
	}
	if err := c.doWithKey(ctx, c.mediaClient, instanceToken, http.MethodPost, "/message/downloadmedia", body, &resp); err != nil {
		return nil, "", err
	}
	rawB64 := strings.TrimSpace(resp.Data.Base64)
	if rawB64 == "" {
		return nil, "", fmt.Errorf("evolution retornou base64 vazio")
	}

	mimeType := ""
	payloadB64 := rawB64
	if strings.HasPrefix(rawB64, "data:") {
		if commaIdx := strings.IndexByte(rawB64, ','); commaIdx > 5 {
			header := rawB64[5:commaIdx]
			if semiIdx := strings.IndexByte(header, ';'); semiIdx > 0 {
				mimeType = strings.TrimSpace(header[:semiIdx])
			} else {
				mimeType = strings.TrimSpace(header)
			}
			payloadB64 = rawB64[commaIdx+1:]
		}
	}

	if maxBytes > 0 && int64(len(payloadB64))*3/4 > maxBytes {
		return nil, mimeType, ErrMediaTooLarge
	}

	decoded, err := base64.StdEncoding.DecodeString(payloadB64)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payloadB64)
		if err != nil {
			return nil, "", fmt.Errorf("decode media base64: %w", err)
		}
	}
	if maxBytes > 0 && int64(len(decoded)) > maxBytes {
		return nil, mimeType, ErrMediaTooLarge
	}
	return decoded, mimeType, nil
}

var ErrMediaTooLarge = fmt.Errorf("media exceeds maximum allowed size")

// FetchExternalMedia baixa a mídia caso o servidor Evolution Go esteja com MINIO_ENABLED=true
// e tenha enviado data.Message.mediaUrl no webhook.
func (c *Client) FetchExternalMedia(ctx context.Context, mediaURL string, maxBytes int64) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, "", err
	}
	res, err := c.mediaClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, "", fmt.Errorf("external media status %d", res.StatusCode)
	}
	if maxBytes > 0 && res.ContentLength > maxBytes {
		return nil, res.Header.Get("Content-Type"), ErrMediaTooLarge
	}

	limit := maxBytes + 1
	if maxBytes <= 0 {
		limit = 20 * 1024 * 1024
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, limit))
	if err != nil {
		return nil, "", err
	}
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		return nil, res.Header.Get("Content-Type"), ErrMediaTooLarge
	}
	return data, res.Header.Get("Content-Type"), nil
}

