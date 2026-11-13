package queue

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

var rxDigitsOnly = regexp.MustCompile(`\d+`)

// NormalizeBareJID remove sufixos de device (:14) mantendo apenas user@domain em lowercase.
func NormalizeBareJID(jid string) string {
	jid = strings.ToLower(strings.TrimSpace(jid))
	if jid == "" {
		return ""
	}
	parts := strings.SplitN(jid, "@", 2)
	if len(parts) != 2 {
		return jid
	}
	user := parts[0]
	if colonIdx := strings.IndexByte(user, ':'); colonIdx > 0 {
		user = user[:colonIdx]
	}
	return user + "@" + parts[1]
}

func splitNormalizedJID(jid string) (digits, domainPart string) {
	bare := NormalizeBareJID(jid)
	parts := strings.SplitN(bare, "@", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return rxDigitsOnly.FindString(parts[0]), parts[1]
}

// ResolveContactJID aplica a blindagem contra grupos/status e contra o bug de swap @lid
// em mensagens enviadas pelo celular (IsFromMe == true) na Evolution Go.
func ResolveContactJID(
	msg *ParsedWhatsAppMessage,
	ownerJID string,
	lookupPhoneJIDByLID func(lidJID string) string,
	saveLIDPair func(lidJID, phoneJID string),
) (remoteJID, phone, lid string, skip bool) {
	if msg == nil || msg.IsGroup {
		return "", "", "", true
	}

	chatBare := NormalizeBareJID(msg.ChatJID)
	if chatBare == "" ||
		strings.HasSuffix(chatBare, "@g.us") ||
		strings.HasSuffix(chatBare, "@broadcast") ||
		strings.HasSuffix(chatBare, "@newsletter") ||
		chatBare == "status@broadcast" {
		return "", "", "", true
	}

	ownerBare := NormalizeBareJID(ownerJID)
	senderBare := NormalizeBareJID(msg.SenderJID)
	senderAltBare := NormalizeBareJID(msg.SenderAltJID)
	recipAltBare := NormalizeBareJID(msg.RecipientAltJID)
	deviceDestBare := NormalizeBareJID(msg.DeviceDestJID)

	var phoneJID, lidJID string

	if !msg.IsFromMe {
		for _, cand := range []string{senderBare, senderAltBare, chatBare} {
			if strings.HasSuffix(cand, "@s.whatsapp.net") && cand != ownerBare {
				phoneJID = cand
				break
			}
		}
		for _, cand := range []string{senderAltBare, senderBare, chatBare} {
			if strings.HasSuffix(cand, "@lid") {
				lidJID = cand
				break
			}
		}
	} else {
		// Quando IsFromMe == true, whatsmeow.go pode ter trocado Chat pelo número do próprio
		// atendente (SenderAlt). Priorizamos RecipientAlt e DeviceSentMeta.DestinationJID!
		for _, cand := range []string{recipAltBare, deviceDestBare, chatBare} {
			if strings.HasSuffix(cand, "@s.whatsapp.net") && (ownerBare == "" || cand != ownerBare) {
				phoneJID = cand
				break
			}
		}
		for _, cand := range []string{recipAltBare, deviceDestBare, chatBare} {
			if strings.HasSuffix(cand, "@lid") && (ownerBare == "" || cand != ownerBare) {
				lidJID = cand
				break
			}
		}
		if phoneJID == "" && lidJID == "" {
			return "", "", "", true
		}
	}

	if phoneJID != "" && lidJID != "" && saveLIDPair != nil {
		saveLIDPair(lidJID, phoneJID)
	} else if phoneJID == "" && lidJID != "" && lookupPhoneJIDByLID != nil {
		if resolved := NormalizeBareJID(lookupPhoneJIDByLID(lidJID)); strings.HasSuffix(resolved, "@s.whatsapp.net") {
			phoneJID = resolved
		}
	}

	if phoneJID != "" {
		rawDigits, _ := splitNormalizedJID(phoneJID)
		phone = util.NormalizePhone(rawDigits)
		remoteJID = phoneJID
	} else if lidJID != "" {
		remoteJID = lidJID
	} else {
		return "", "", "", true
	}

	if lidJID != "" {
		lidDigits, _ := splitNormalizedJID(lidJID)
		lid = lidDigits
	}
	return remoteJID, phone, lid, false
}

func extractMediaSubMessage(msgMap map[string]json.RawMessage, out *ParsedWhatsAppMessage) {
	mediaKeys := [][2]string{
		{"audioMessage", "audio"}, {"imageMessage", "image"},
		{"documentMessage", "document"}, {"videoMessage", "video"}, {"stickerMessage", "sticker"},
	}
	for _, mk := range mediaKeys {
		rawSub, ok := msgMap[mk[0]]
		if !ok {
			continue
		}
		var sub struct {
			URL         string          `json:"URL"`
			UrlLower    string          `json:"url"`
			Mimetype    string          `json:"mimetype"`
			Caption     string          `json:"caption"`
			FileLength  uint64          `json:"fileLength"`
			Seconds     uint32          `json:"seconds"`
			FileName    string          `json:"fileName"`
			Title       string          `json:"title"`
			ContextInfo *evoContextInfo `json:"contextInfo"`
		}
		if json.Unmarshal(rawSub, &sub) == nil && (sub.URL != "" || sub.UrlLower != "" || out.ExternalMediaURL != "") {
			out.MessageType = mk[1]
			out.HasMedia = true
			out.MimeType = sub.Mimetype
			out.FileLength = int64(sub.FileLength)
			out.DurationSec = int(sub.Seconds)
			out.Filename = util.FirstNonEmpty(sub.FileName, sub.Title)
			if sub.Caption != "" && out.Content == "" {
				out.Content = sub.Caption
			}
			applyContextInfo(sub.ContextInfo, out)
			protoEnvelope, _ := json.Marshal(map[string]json.RawMessage{mk[0]: rawSub})
			out.MediaProtoJSON = protoEnvelope
			return
		}
	}
}

func applyContextInfo(ctxInfo *evoContextInfo, out *ParsedWhatsAppMessage) {
	if ctxInfo == nil {
		return
	}
	if clid := strings.TrimSpace(ctxInfo.CtwaClidDirect); clid != "" {
		out.HasMsgAdReferral = true
		if out.CtwaClid == "" {
			out.CtwaClid = clid
		}
	}
	applyReferral(&ctxInfo.ExternalAdReply, util.FirstNonEmpty(ctxInfo.ConversionApp, ctxInfo.ConversionSrc, ctxInfo.ConversionFB), out)
}

func applyReferral(ref *evoAdReferral, fallbackApp string, out *ParsedWhatsAppMessage) {
	if ref == nil {
		return
	}
	if clid := strings.TrimSpace(util.FirstNonEmpty(ref.CtwaClid, ref.CtwaClidSnake)); clid != "" {
		out.HasMsgAdReferral = true
		if out.CtwaClid == "" {
			out.CtwaClid = clid
		}
	}
	if sid := strings.TrimSpace(util.FirstNonEmpty(ref.SourceID, ref.SourceIdAlt, ref.SourceIDSnake)); sid != "" {
		out.HasMsgAdReferral = true
		if out.SourceID == "" {
			out.SourceID = sid
		}
	}
	if surl := util.FirstNonEmpty(ref.SourceURL, ref.SourceUrlAlt); surl != "" && out.SourceURL == "" {
		out.SourceURL = strings.TrimSpace(surl)
	}
	if app := util.FirstNonEmpty(ref.SourceApp, ref.SourceAppAlt, fallbackApp); app != "" && out.SourceApp == "" {
		out.SourceApp = strings.ToLower(strings.TrimSpace(app))
	}
	if thumb := util.FirstNonEmpty(ref.ThumbnailURL, ref.OriginalImageURL, ref.MediaURL); thumb != "" && out.AdThumbnailURL == "" {
		out.AdThumbnailURL = strings.TrimSpace(thumb)
	}
}

var rxProtocol = regexp.MustCompile(`(?i)(?:protocolo(?:\s+de\s+atendimento)?|c[oó]d(?:igo)?\s+de\s+atendimento|ref(?:er[eê]ncia)?)\s*[:#\-]?\s*\*?([A-Za-z0-9_-]{6,36})\*?`)

func ExtractProtocolCode(text string) string {
	m := rxProtocol.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}
