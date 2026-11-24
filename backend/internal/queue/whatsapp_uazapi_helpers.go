package queue

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// StripHeavyUazapiThumbnails remove JPEGThumbnail e thumbnail base64 pesados antes de gravar no Redis.
func StripHeavyUazapiThumbnails(raw json.RawMessage) json.RawMessage {
	raw = UnwrapUazapiEnvelope(raw)
	s := string(raw)
	if !strings.Contains(s, "JPEGThumbnail") && !strings.Contains(s, `"thumbnail"`) {
		return raw
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil {
		return raw
	}
	msgRaw, ok := root["message"]
	if !ok {
		return raw
	}
	var msgMap map[string]json.RawMessage
	if json.Unmarshal(msgRaw, &msgMap) != nil {
		return raw
	}
	contentRaw, ok := msgMap["content"]
	if !ok {
		return raw
	}
	var contentMap map[string]json.RawMessage
	if json.Unmarshal(contentRaw, &contentMap) != nil {
		return raw
	}
	delete(contentMap, "JPEGThumbnail")
	if ctxRaw, hasCtx := contentMap["contextInfo"]; hasCtx {
		var ctxMap map[string]json.RawMessage
		if json.Unmarshal(ctxRaw, &ctxMap) == nil {
			if adRaw, hasAd := ctxMap["externalAdReply"]; hasAd {
				var adMap map[string]json.RawMessage
				if json.Unmarshal(adRaw, &adMap) == nil {
					delete(adMap, "thumbnail")
					if b, err := json.Marshal(adMap); err == nil {
						ctxMap["externalAdReply"] = b
					}
				}
			}
			if b, err := json.Marshal(ctxMap); err == nil {
				contentMap["contextInfo"] = b
			}
		}
	}
	if b, err := json.Marshal(contentMap); err == nil {
		msgMap["content"] = b
	}
	if b, err := json.Marshal(msgMap); err == nil {
		root["message"] = b
	}
	if b, err := json.Marshal(root); err == nil {
		return b
	}
	return raw
}

func stripOwnerPrefix(id string) string {
	id = strings.TrimSpace(id)
	if idx := strings.IndexByte(id, ':'); idx >= 0 && idx+1 < len(id) {
		return strings.TrimSpace(id[idx+1:])
	}
	return id
}

func cleanUazapiName(s string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(s), `"'`))
}

func normalizeUazapiPhoneJID(cand string) string {
	cand = strings.TrimSpace(cand)
	if cand == "" {
		return ""
	}
	bare := NormalizeBareJID(cand)
	if strings.HasSuffix(bare, "@c.us") {
		bare = strings.TrimSuffix(bare, "@c.us") + "@s.whatsapp.net"
	}
	if strings.HasSuffix(bare, "@s.whatsapp.net") {
		digits, _ := splitNormalizedJID(bare)
		if len(digits) >= 8 {
			return digits + "@s.whatsapp.net"
		}
		return ""
	}
	if !strings.Contains(bare, "@") {
		digits := rxDigitsOnly.FindString(bare)
		if len(digits) >= 8 {
			return digits + "@s.whatsapp.net"
		}
	}
	return ""
}

func resolveUazapiPhoneJID(fromMe bool, ownerBare, senderPN, msgChatID, chatWaChatID, chatPhone string) string {
	var cands []string
	if !fromMe {
		cands = []string{senderPN, msgChatID, chatWaChatID, chatPhone}
	} else {
		cands = []string{msgChatID, chatWaChatID, chatPhone}
	}
	for _, raw := range cands {
		jid := normalizeUazapiPhoneJID(raw)
		if jid != "" && (ownerBare == "" || jid != ownerBare) {
			return jid
		}
	}
	return ""
}

func resolveUazapiLID(fromMe bool, msgChatLID, chatWaChatLID, senderLID, sender string) string {
	var cands []string
	if !fromMe {
		cands = []string{msgChatLID, chatWaChatLID, senderLID, sender}
	} else {
		cands = []string{msgChatLID, chatWaChatLID}
	}
	for _, raw := range cands {
		bare := NormalizeBareJID(raw)
		if strings.HasSuffix(bare, "@lid") {
			return bare
		}
	}
	return ""
}

func extractUazapiContentAndContext(rawContent json.RawMessage, out *ParsedWhatsAppMessage) {
	trimmed := bytes.TrimSpace(rawContent)
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

	var obj struct {
		Text        string          `json:"text"`
		Caption     string          `json:"caption"`
		Title       string          `json:"title"`
		Description string          `json:"description"`
		URL         string          `json:"URL"`
		UrlLower    string          `json:"url"`
		Mimetype    string          `json:"mimetype"`
		ContextInfo *evoContextInfo `json:"contextInfo"`
	}
	if json.Unmarshal(trimmed, &obj) == nil {
		if out.Content == "" {
			out.Content = strings.TrimSpace(util.FirstNonEmpty(obj.Text, obj.Caption))
		}
		if out.ExternalMediaURL == "" {
			out.ExternalMediaURL = strings.TrimSpace(util.FirstNonEmpty(obj.URL, obj.UrlLower))
		}
		if out.MimeType == "" && obj.Mimetype != "" {
			out.MimeType = strings.TrimSpace(obj.Mimetype)
		}
		applyContextInfo(obj.ContextInfo, out)
	}
}

func isStandardUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		c := s[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex {
			return false
		}
	}
	return true
}

// isValidUazapiCtwaClid valida se lead_field10 contém um ctwa_clid genuíno da Meta,
// rejeitando textos arbitrários de CRM, e-mails, telefones, CPFs ou UUIDs.
func isValidUazapiCtwaClid(raw string) bool {
	s := strings.TrimSpace(raw)
	if len(s) < 20 || len(s) > 512 || isStandardUUID(s) {
		return false
	}
	hasLetter := false
	hasDigit := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			hasLetter = true
		case c >= '0' && c <= '9':
			hasDigit = true
		case c == '+' || c == '/' || c == '=' || c == '.' || c == '_' || c == '-':
		default:
			return false
		}
	}
	return hasLetter && hasDigit
}

func normalizeAdSourceApp(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "facebook", "fb", "fb_ads", "meta", "meta_ads", "ctwa", "ctwa_ad":
		return "facebook"
	case "instagram", "ig", "ig_ads":
		return "instagram"
	default:
		return ""
	}
}

// isValidUazapiLeadAdSourceID valida se lead_field11 representa um Ad ID numérico real da Meta,
// evitando que CPFs, CNPJs ou telefones salvos em campos customizados virem CTWA falso.
func isValidUazapiLeadAdSourceID(rawID, validClid, rawApp string) bool {
	s := strings.TrimSpace(rawID)
	if len(s) < 12 || len(s) > 25 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	if (len(s) == 12 || len(s) == 13) && strings.HasPrefix(s, "55") {
		return false
	}
	return validClid != "" || normalizeAdSourceApp(rawApp) != "" || len(s) >= 15
}
