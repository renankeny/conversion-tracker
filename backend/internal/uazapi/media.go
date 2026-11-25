package uazapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var ErrMediaTooLarge = fmt.Errorf("media exceeds maximum allowed size")

type downloadResponse struct {
	FileURL     string `json:"fileURL"`
	FileURLAlt  string `json:"fileUrl"`
	Mimetype    string `json:"mimetype"`
	MimeTypeAlt string `json:"mimeType"`
	Base64Data  string `json:"base64Data"`
	Base64Alt   string `json:"base64"`
}

// DownloadMedia solicita à API Premium / Uazapi a descriptografia de mídia via POST /message/download,
// priorizando o download por fileURL (recomendado pela documentação oficial) com fallback para base64Data.
func (c *Client) DownloadMedia(ctx context.Context, messageID, rawMessageID string, maxBytes int64) ([]byte, string, error) {
	if !c.Configured() {
		return nil, "", fmt.Errorf("uazapi client not configured")
	}
	candidates := uniqueNonEmpty(messageID, rawMessageID)
	if len(candidates) == 0 {
		return nil, "", fmt.Errorf("missing uazapi message id")
	}

	var lastErr error
	for _, cand := range candidates {
		resp, err := c.requestDownload(ctx, cand, false, maxBytes)
		if err != nil {
			lastErr = err
			continue
		}
		respMime := firstStr(resp.Mimetype, resp.MimeTypeAlt)
		fileURL := firstStr(resp.FileURL, resp.FileURLAlt)
		if fileURL != "" {
			data, fetchedMime, ferr := c.FetchExternalMedia(ctx, fileURL, maxBytes)
			if errors.Is(ferr, ErrMediaTooLarge) {
				return nil, pickSpecificMime(respMime, fetchedMime), ErrMediaTooLarge
			}
			if ferr == nil && len(data) > 0 {
				return data, pickSpecificMime(respMime, fetchedMime), nil
			}
			lastErr = ferr
		}
		if b64 := firstStr(resp.Base64Data, resp.Base64Alt); b64 != "" {
			return decodeMediaBase64(b64, respMime, maxBytes)
		}

		// Fallback em base64 caso a instância esteja atrás de rede interna ou não retorne fileURL pública.
		b64Resp, berr := c.requestDownload(ctx, cand, true, maxBytes)
		if berr != nil {
			if lastErr == nil {
				lastErr = berr
			}
			continue
		}
		if b64 := firstStr(b64Resp.Base64Data, b64Resp.Base64Alt); b64 != "" {
			mime := pickSpecificMime(firstStr(b64Resp.Mimetype, b64Resp.MimeTypeAlt), respMime)
			return decodeMediaBase64(b64, mime, maxBytes)
		}
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("uazapi /message/download returned empty media")
}

func (c *Client) requestDownload(ctx context.Context, id string, returnBase64 bool, maxBytes int64) (*downloadResponse, error) {
	payload, err := json.Marshal(map[string]any{
		"id":            id,
		"return_base64": returnBase64,
		"return_link":   !returnBase64,
		"generate_mp3":  true,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/message/download", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", c.token)

	res, err := c.mediaHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uazapi download request: %w", err)
	}
	defer res.Body.Close()

	maxJSON := int64(2 << 20)
	if returnBase64 && maxBytes > 0 {
		maxJSON = maxBytes*4/3 + (2 << 20)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxJSON))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("uazapi download status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var out downloadResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode uazapi download response: %w", err)
	}
	return &out, nil
}

func (c *Client) FetchExternalMedia(ctx context.Context, mediaURL string, maxBytes int64) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(mediaURL), nil)
	if err != nil {
		return nil, "", err
	}
	httpCli := c.http
	if c != nil && c.mediaHTTP != nil {
		httpCli = c.mediaHTTP
	}
	res, err := httpCli.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, "", fmt.Errorf("external media status %d", res.StatusCode)
	}
	ct := strings.TrimSpace(res.Header.Get("Content-Type"))
	if maxBytes > 0 && res.ContentLength > maxBytes {
		return nil, ct, ErrMediaTooLarge
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
		return nil, ct, ErrMediaTooLarge
	}
	return data, ct, nil
}

func decodeMediaBase64(rawB64, fallbackMime string, maxBytes int64) ([]byte, string, error) {
	mimeType := fallbackMime
	payloadB64 := strings.TrimSpace(rawB64)
	if strings.HasPrefix(payloadB64, "data:") {
		if commaIdx := strings.IndexByte(payloadB64, ','); commaIdx > 5 {
			header := payloadB64[5:commaIdx]
			if semiIdx := strings.IndexByte(header, ';'); semiIdx > 0 {
				header = header[:semiIdx]
			}
			if strings.TrimSpace(header) != "" {
				mimeType = strings.TrimSpace(header)
			}
			payloadB64 = payloadB64[commaIdx+1:]
		}
	}
	if maxBytes > 0 && int64(len(payloadB64))*3/4 > maxBytes {
		return nil, mimeType, ErrMediaTooLarge
	}
	decoded, err := base64.StdEncoding.DecodeString(payloadB64)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payloadB64)
		if err != nil {
			return nil, "", fmt.Errorf("decode uazapi media base64: %w", err)
		}
	}
	if maxBytes > 0 && int64(len(decoded)) > maxBytes {
		return nil, mimeType, ErrMediaTooLarge
	}
	return decoded, mimeType, nil
}

func uniqueNonEmpty(vals ...string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		s := strings.TrimSpace(v)
		if s == "" {
			continue
		}
		dup := false
		for _, ex := range out {
			if ex == s {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, s)
		}
	}
	return out
}

func firstStr(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func pickSpecificMime(primary, secondary string) string {
	p := strings.TrimSpace(primary)
	s := strings.TrimSpace(secondary)
	if p != "" && !strings.EqualFold(p, "application/octet-stream") {
		return p
	}
	if s != "" && !strings.EqualFold(s, "application/octet-stream") {
		return s
	}
	return firstStr(p, s)
}
