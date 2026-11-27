package queue

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func preloadWAHAMediaFields(msg *wahaMessagePayload, out *ParsedWhatsAppMessage) {
	if msg == nil || out == nil {
		return
	}
	if msg.HasMedia {
		out.HasMedia = true
	}
	var mediaURL string
	if msg.Media != nil {
		mediaURL = strings.TrimSpace(msg.Media.URL)
	}
	if mediaURL == "" {
		mediaURL = strings.TrimSpace(msg.MediaURL)
	}
	if isUsableWAHAMediaURL(mediaURL) {
		out.ExternalMediaURL = mediaURL
		out.HasMedia = true
	}
}

func applyWAHAMediaMetadata(msg *wahaMessagePayload, rawDataType string, out *ParsedWhatsAppMessage) {
	if msg == nil || out == nil {
		return
	}
	var mediaURL, mediaMime, mediaFilename string
	if msg.Media != nil {
		mediaURL = strings.TrimSpace(msg.Media.URL)
		mediaMime = strings.TrimSpace(msg.Media.Mimetype)
		mediaFilename = strings.TrimSpace(msg.Media.Filename)
	}
	if mediaURL == "" {
		mediaURL = strings.TrimSpace(msg.MediaURL)
	}
	if isUsableWAHAMediaURL(mediaURL) {
		out.ExternalMediaURL = mediaURL
	}
	if mediaMime != "" {
		out.MimeType = mediaMime
	}
	if mediaFilename != "" && out.Filename == "" {
		out.Filename = mediaFilename
	}

	if msg.HasMedia || out.ExternalMediaURL != "" || out.HasMedia {
		out.HasMedia = true
		if out.MessageType == "" || out.MessageType == "text" {
			out.MessageType = classifyWAHAMediaType(rawDataType, out.MimeType)
		}
	}
}

func classifyWAHAMediaType(rawType, mimeType string) string {
	t := strings.ToLower(strings.TrimSpace(rawType))
	switch t {
	case "ptt", "audio", "voice":
		return "audio"
	case "image", "photo":
		return "image"
	case "video", "ptv", "gif":
		return "video"
	case "sticker":
		return "sticker"
	case "document", "file", "vcard", "multi_vcard":
		return "document"
	}

	m := strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.HasPrefix(m, "audio/"):
		return "audio"
	case strings.HasPrefix(m, "video/"):
		return "video"
	case m == "image/webp":
		return "sticker"
	case strings.HasPrefix(m, "image/"):
		return "image"
	case strings.HasPrefix(m, "application/") || strings.Contains(m, "vcard") || m == "text/csv":
		return "document"
	default:
		return "document"
	}
}

// isUsableWAHAMediaURL aceita URLs da própria WAHA (incluindo http://localhost:3000/api/files/...)
// e URLs externas descriptografadas (S3/R2), rejeitando a CDN criptografada do WhatsApp (.enc).
func isUsableWAHAMediaURL(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "/api/files/") {
		return true
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(host, "whatsapp.net") || strings.HasSuffix(host, "whatsapp.com") {
		return false
	}
	return !strings.HasSuffix(strings.ToLower(u.Path), ".enc")
}

func isWAHAIgnorableRawData(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}
	var quick struct {
		Type       string `json:"type"`
		IsGroupMsg bool   `json:"isGroupMsg"`
		Info       *struct {
			IsGroup bool `json:"IsGroup"`
		} `json:"Info"`
	}
	if json.Unmarshal(trimmed, &quick) != nil {
		return false
	}
	if quick.IsGroupMsg || (quick.Info != nil && quick.Info.IsGroup) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(quick.Type)) {
	case "e2e_notification", "notification", "notification_template",
		"gp2", "protocol", "revoked", "reaction", "call_log", "ciphertext":
		return true
	}
	return false
}

// StripHeavyWAHAThumbnails remove campos base64 pesados (JPEGThumbnail, thumbnail, body base64)
// dentro de payload._data antes de enfileirar no Redis.
func StripHeavyWAHAThumbnails(raw json.RawMessage) json.RawMessage {
	s := string(raw)
	if !strings.Contains(s, "JPEGThumbnail") && !strings.Contains(s, `"thumbnail"`) && !strings.Contains(s, `"base64"`) {
		return raw
	}
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return raw
	}
	if p, ok := root["payload"].(map[string]any); ok {
		if dataMap, ok := p["_data"].(map[string]any); ok {
			pruneHeavyFields(dataMap, 0)
		}
	}
	cleaned, err := json.Marshal(root)
	if err != nil {
		return raw
	}
	return cleaned
}

func pruneHeavyFields(m map[string]any, depth int) {
	if depth > 6 {
		return
	}
	for k, v := range m {
		switch k {
		case "JPEGThumbnail", "jpegThumbnail", "thumbnail", "base64", "mediaKey":
			if str, ok := v.(string); ok && len(str) > 256 {
				delete(m, k)
				continue
			}
		}
		if sub, ok := v.(map[string]any); ok {
			pruneHeavyFields(sub, depth+1)
		}
	}
}

func resolveWAHAMeOwnerJID(me *wahaMeInfo) string {
	if me == nil {
		return ""
	}
	for _, cand := range []string{me.JID, me.ID} {
		norm := NormalizeWAHAJID(cand)
		if strings.HasSuffix(norm, "@s.whatsapp.net") {
			return norm
		}
	}
	return util.FirstNonEmpty(NormalizeWAHAJID(me.JID), NormalizeWAHAJID(me.ID))
}
