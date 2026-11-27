package waha

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var ErrMediaTooLarge = errors.New("media exceeds maximum allowed size")

// DownloadMedia baixa mídias da WAHA API com suporte a:
// 1. Reescrita automática de URLs localhost/127.0.0.1 para o host público configurado da WAHA;
// 2. Autenticação X-Api-Key apenas quando o destino é o próprio servidor WAHA (preserva S3 pré-assinado);
// 3. Fallback sob demanda via GET /api/{session}/chats/{chatId}/messages/{messageId}?downloadMedia=true.
func (c *Client) DownloadMedia(
	ctx context.Context,
	sessionName, mediaURL, chatID, messageID string,
	maxBytes int64,
) ([]byte, string, error) {
	if !c.Configured() && strings.TrimSpace(mediaURL) == "" {
		return nil, "", fmt.Errorf("waha client not configured")
	}

	var lastErr error
	if cleaned := strings.TrimSpace(mediaURL); cleaned != "" {
		data, mime, err := c.fetchURL(ctx, cleaned, maxBytes)
		if errors.Is(err, ErrMediaTooLarge) {
			return nil, mime, ErrMediaTooLarge
		}
		if err == nil && len(data) > 0 {
			return data, mime, nil
		}
		lastErr = err
	}

	if c.Configured() && strings.TrimSpace(sessionName) != "" && strings.TrimSpace(chatID) != "" && strings.TrimSpace(messageID) != "" {
		fetchedURL, fetchedMime, ferr := c.fetchMessageMediaURL(ctx, sessionName, chatID, messageID)
		if ferr != nil {
			if lastErr != nil {
				return nil, "", fmt.Errorf("waha media download: %w (fallback: %v)", lastErr, ferr)
			}
			return nil, "", ferr
		}
		if fetchedURL != "" {
			data, mime, err := c.fetchURL(ctx, fetchedURL, maxBytes)
			if errors.Is(err, ErrMediaTooLarge) {
				return nil, pickSpecificMime(fetchedMime, mime), ErrMediaTooLarge
			}
			if err == nil && len(data) > 0 {
				return data, pickSpecificMime(fetchedMime, mime), nil
			}
			return nil, "", err
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("waha media url unavailable")
}

func (c *Client) fetchMessageMediaURL(ctx context.Context, sessionName, chatID, messageID string) (string, string, error) {
	path := fmt.Sprintf(
		"/api/%s/chats/%s/messages/%s?downloadMedia=true",
		url.PathEscape(strings.TrimSpace(sessionName)),
		url.PathEscape(strings.TrimSpace(chatID)),
		url.PathEscape(strings.TrimSpace(messageID)),
	)
	var resp struct {
		Media *struct {
			URL      string `json:"url"`
			Mimetype string `json:"mimetype"`
		} `json:"media"`
		MediaURL string `json:"mediaUrl"`
	}
	if err := c.doJSON(ctx, c.mediaClient, http.MethodGet, path, nil, &resp); err != nil {
		return "", "", err
	}
	if resp.Media != nil && strings.TrimSpace(resp.Media.URL) != "" {
		return strings.TrimSpace(resp.Media.URL), strings.TrimSpace(resp.Media.Mimetype), nil
	}
	return strings.TrimSpace(resp.MediaURL), "", nil
}

func (c *Client) fetchURL(ctx context.Context, rawURL string, maxBytes int64) ([]byte, string, error) {
	targetURL, sendAPIKey := c.resolveMediaRequestURL(rawURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, "", err
	}
	if sendAPIKey && c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}

	res, err := c.mediaClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("waha fetch media: %w", err)
	}
	defer res.Body.Close()

	ct := strings.TrimSpace(res.Header.Get("Content-Type"))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, ct, fmt.Errorf("waha media status %d", res.StatusCode)
	}
	if maxBytes > 0 && res.ContentLength > maxBytes {
		return nil, ct, ErrMediaTooLarge
	}
	limit := maxBytes + 1
	if maxBytes <= 0 {
		limit = 20 * 1024 * 1024
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, limit))
	if err != nil {
		return nil, ct, err
	}
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		return nil, ct, ErrMediaTooLarge
	}
	return data, ct, nil
}

// resolveMediaRequestURL reescreve URLs locais da WAHA (ex: http://localhost:3000/api/files/...)
// para o baseURL configurado e indica se X-Api-Key deve ser enviado (apenas para o mesmo host:porta da WAHA).
func (c *Client) resolveMediaRequestURL(rawURL string) (string, bool) {
	trimmed := strings.TrimSpace(rawURL)
	if strings.HasPrefix(trimmed, "/") && c.Configured() {
		return c.baseURL + trimmed, true
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return trimmed, false
	}
	baseParsed, berr := url.Parse(c.baseURL)
	if berr != nil || baseParsed.Host == "" {
		return trimmed, false
	}
	host := strings.ToLower(parsed.Hostname())
	isLocalhost := host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1"
	if isLocalhost && strings.HasPrefix(parsed.Path, "/api/files/") {
		parsed.Scheme = baseParsed.Scheme
		parsed.Host = baseParsed.Host
		return parsed.String(), true
	}
	sameHost := strings.EqualFold(parsed.Host, baseParsed.Host)
	return trimmed, sameHost
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
	if p != "" {
		return p
	}
	return s
}
