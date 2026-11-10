package queue

import (
	"regexp"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
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
		// Se o único JID encontrado for o próprio ownerJID, descarta para não criar chat fantasma.
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

func isPaidMetaUTM(src, med string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	m := strings.ToLower(strings.TrimSpace(med))
	isMeta := s == "facebook" || s == "fb" || s == "instagram" || s == "ig" || s == "meta" || strings.Contains(s, "facebook") || strings.Contains(s, "instagram")
	if !isMeta {
		return false
	}
	return m == "cpc" || m == "ppc" || m == "paid" || m == "ads" || m == "cpm" || m == "paid_social" || strings.Contains(m, "ad") || strings.Contains(m, "pago") || strings.Contains(m, "trafego")
}

func isBioLinkSession(u *domain.TrackedUser) bool {
	if u == nil {
		return false
	}
	med := strings.ToLower(u.FirstUTMMedium + " " + u.LastUTMMedium + " " + u.FirstUTMSource + " " + u.LastUTMSource)
	if strings.Contains(med, "bio") || strings.Contains(med, "linktree") || strings.Contains(med, "perfil") {
		return true
	}
	urls := strings.ToLower(u.FirstURL + " " + u.URL + " " + u.Referrer)
	for _, token := range []string{"linktr.ee", "beacons.ai", "bio.site", "lnk.bio", "l.instagram.com", "instagram.com", "/bio"} {
		if strings.Contains(urls, token) {
			return true
		}
	}
	return false
}

func applyReferral(ref *evoAdReferral, fallbackApp string, out *ParsedWhatsAppMessage) {
	if ref == nil {
		return
	}
	if ref.CtwaClid != "" && out.CtwaClid == "" {
		out.CtwaClid = strings.TrimSpace(ref.CtwaClid)
	}
	sid := util.FirstNonEmpty(ref.SourceID, ref.SourceIdAlt)
	if sid != "" && out.SourceID == "" {
		out.SourceID = strings.TrimSpace(sid)
	}
	if ref.SourceURL != "" && out.SourceURL == "" {
		out.SourceURL = strings.TrimSpace(ref.SourceURL)
	}
	app := util.FirstNonEmpty(ref.SourceApp, fallbackApp)
	if app != "" && out.SourceApp == "" {
		out.SourceApp = strings.ToLower(strings.TrimSpace(app))
	}
	thumb := util.FirstNonEmpty(ref.ThumbnailURL, ref.MediaURL)
	if thumb != "" && out.AdThumbnailURL == "" {
		out.AdThumbnailURL = strings.TrimSpace(thumb)
	}
}

var rxProtocol = regexp.MustCompile(`(?i)protocolo:?\s*([A-Za-z0-9_-]{6,36})`)

func ExtractProtocolCode(text string) string {
	m := rxProtocol.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func ClassifyPortalOrigin(ctwaClid, sourceID string, u *domain.TrackedUser) string {
	if ctwaClid != "" || sourceID != "" {
		return domain.WhatsAppOriginMetaCTWA
	}
	if u == nil {
		return domain.WhatsAppOriginWebsiteOrganic
	}
	if u.FBC != "" || isPaidMetaUTM(u.FirstUTMSource, u.FirstUTMMedium) || isPaidMetaUTM(u.LastUTMSource, u.LastUTMMedium) {
		return domain.WhatsAppOriginMetaLP
	}
	if u.GCLID != "" || u.GBraid != "" || u.WBraid != "" {
		return domain.WhatsAppOriginGoogleAds
	}
	if u.TTClid != "" {
		return domain.WhatsAppOriginTikTokAds
	}
	if isBioLinkSession(u) {
		return domain.WhatsAppOriginBioLinkOrganic
	}
	return domain.WhatsAppOriginWebsiteOrganic
}
