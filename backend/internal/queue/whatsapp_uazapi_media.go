package queue

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func extractUazapiContentAndMedia(
	msgType, mediaType, normType, rawFileURL string,
	rawContent json.RawMessage,
	out *ParsedWhatsAppMessage,
) {
	if isDecryptedPublicMediaURL(rawFileURL) {
		out.ExternalMediaURL = strings.TrimSpace(rawFileURL)
	}

	var isPTT bool
	parseUazapiContentNode(rawContent, out, &isPTT, 0)

	if kind, ok := classifyUazapiMediaType(msgType, mediaType, normType, out.MimeType, isPTT); ok {
		out.MessageType = kind
		out.HasMedia = true
	}
}

func parseUazapiContentNode(raw json.RawMessage, out *ParsedWhatsAppMessage, isPTT *bool, depth int) {
	if depth > 4 {
		return
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return
	}
	if trimmed[0] == '"' {
		var s string
		if json.Unmarshal(trimmed, &s) == nil && strings.TrimSpace(s) != "" && out.Content == "" {
			out.Content = strings.TrimSpace(s)
		}
		return
	}
	if trimmed[0] != '{' {
		return
	}

	var nodeMap map[string]json.RawMessage
	if json.Unmarshal(trimmed, &nodeMap) == nil {
		for _, wrapKey := range []string{
			"documentWithCaptionMessage", "viewOnceMessage", "viewOnceMessageV2",
			"viewOnceMessageV2Extension", "ephemeralMessage", "editedMessage",
			"deviceSentMessage", "message", "audioMessage", "imageMessage",
			"videoMessage", "ptvMessage", "documentMessage", "stickerMessage", "contactMessage",
		} {
			if subRaw, ok := nodeMap[wrapKey]; ok {
				parseUazapiContentNode(subRaw, out, isPTT, depth+1)
			}
		}
	}

	var obj struct {
		Text        string          `json:"text"`
		Caption     string          `json:"caption"`
		Title       string          `json:"title"`
		FileName    string          `json:"fileName"`
		FilenameAlt string          `json:"filename"`
		DisplayName string          `json:"displayName"`
		FileURL     string          `json:"fileURL"`
		FileURLAlt  string          `json:"fileUrl"`
		Mimetype    string          `json:"mimetype"`
		MimeTypeAlt string          `json:"mimeType"`
		FileLength  json.RawMessage `json:"fileLength"`
		Seconds     json.RawMessage `json:"seconds"`
		PTTUpper    bool            `json:"PTT"`
		PTTLower    bool            `json:"ptt"`
		ContextInfo *evoContextInfo `json:"contextInfo"`
	}
	if json.Unmarshal(trimmed, &obj) != nil {
		return
	}

	if obj.PTTUpper || obj.PTTLower {
		*isPTT = true
	}
	if out.Content == "" {
		out.Content = strings.TrimSpace(util.FirstNonEmpty(obj.Text, obj.Caption))
	}
	if candURL := strings.TrimSpace(util.FirstNonEmpty(obj.FileURL, obj.FileURLAlt)); out.ExternalMediaURL == "" && isDecryptedPublicMediaURL(candURL) {
		out.ExternalMediaURL = candURL
	}
	if mime := strings.TrimSpace(util.FirstNonEmpty(obj.Mimetype, obj.MimeTypeAlt)); out.MimeType == "" && mime != "" {
		out.MimeType = mime
	}
	if out.FileLength == 0 {
		out.FileLength = parseFlexibleInt64(obj.FileLength)
	}
	if out.DurationSec == 0 {
		out.DurationSec = int(parseFlexibleInt64(obj.Seconds))
	}
	if fname := strings.TrimSpace(util.FirstNonEmpty(obj.FileName, obj.FilenameAlt, obj.Title, obj.DisplayName)); out.Filename == "" && fname != "" {
		out.Filename = fname
	}
	applyContextInfo(obj.ContextInfo, out)
}

func classifyUazapiMediaType(msgType, mediaType, normType, mimeType string, isPTT bool) (string, bool) {
	mt := strings.ToLower(strings.TrimSpace(mediaType))
	wt := strings.ToLower(strings.TrimSpace(msgType))
	nt := strings.ToLower(strings.TrimSpace(normType))
	mm := strings.ToLower(strings.TrimSpace(mimeType))

	switch mt {
	case "audio", "ptt", "myaudio", "voice":
		return "audio", true
	case "video", "ptv", "gif":
		return "video", true
	case "image", "photo":
		return "image", true
	case "sticker":
		return "sticker", true
	case "document", "file", "vcard", "contact":
		return "document", true
	}
	if isPTT {
		return "audio", true
	}

	switch {
	case strings.Contains(wt, "sticker"):
		return "sticker", true
	case strings.Contains(wt, "audio") || strings.Contains(wt, "ptt") || wt == "voice":
		return "audio", true
	case strings.Contains(wt, "video") || strings.Contains(wt, "ptv") || wt == "gifmessage":
		return "video", true
	case strings.Contains(wt, "image") || wt == "photo":
		return "image", true
	case strings.Contains(wt, "document") || wt == "contactmessage" || wt == "contactsarraymessage" || wt == "vcard":
		return "document", true
	}

	switch {
	case strings.HasPrefix(mm, "audio/"):
		return "audio", true
	case strings.HasPrefix(mm, "video/"):
		return "video", true
	case strings.HasPrefix(mm, "image/"):
		return "image", true
	case strings.HasPrefix(mm, "application/") || strings.Contains(mm, "vcard") || mm == "text/csv":
		return "document", true
	case nt == "media":
		return "document", true
	}
	return "text", false
}

// isDecryptedPublicMediaURL aceita apenas URLs HTTP(S) públicas já descriptografadas,
// ignorando URLs criptografadas da CDN interna do WhatsApp (mmg.whatsapp.net / .enc).
func isDecryptedPublicMediaURL(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(host, "whatsapp.net") || strings.HasSuffix(host, "whatsapp.com") {
		return false
	}
	if strings.HasSuffix(strings.ToLower(u.Path), ".enc") {
		return false
	}
	return true
}

func parseFlexibleInt64(raw json.RawMessage) int64 {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return 0
	}
	var n int64
	if json.Unmarshal(trimmed, &n) == nil && n > 0 {
		return n
	}
	var f float64
	if json.Unmarshal(trimmed, &f) == nil && f > 0 {
		return int64(f)
	}
	var s string
	if json.Unmarshal(trimmed, &s) == nil {
		s = strings.TrimSpace(s)
		if v, err := strconv.ParseInt(s, 10, 64); err == nil && v > 0 {
			return v
		}
		if vf, err := strconv.ParseFloat(s, 64); err == nil && vf > 0 {
			return int64(vf)
		}
	}
	var hl struct {
		Low int64 `json:"low"`
	}
	if json.Unmarshal(trimmed, &hl) == nil && hl.Low > 0 {
		return hl.Low
	}
	return 0
}
