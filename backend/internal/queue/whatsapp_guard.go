package queue

import (
	"regexp"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// IsIgnorableSpecialJID bloqueia grupos, listas de transmissão, Status, canais (newsletter),
// bots internos (@bot / Meta AI) e o número oficial de sistema do WhatsApp.
func IsIgnorableSpecialJID(raw string) bool {
	bare := NormalizeBareJID(raw)
	if bare == "" {
		return false
	}
	if strings.HasSuffix(bare, "@g.us") ||
		strings.HasSuffix(bare, "@broadcast") ||
		strings.HasSuffix(bare, "@newsletter") ||
		strings.HasSuffix(bare, "@bot") {
		return true
	}
	return bare == "0@s.whatsapp.net" || bare == "13135550002@s.whatsapp.net"
}

func isIgnorableUazapiMessageType(msgType, normType string) bool {
	for _, raw := range []string{msgType, normType} {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "reactionmessage", "reaction", "protocolmessage",
			"keepinchatmessage", "pininchatmessage", "pollupdatemessage",
			"groupstatusmessage", "senderkeydistributionmessage",
			"messagehistorybundle", "call", "calllogmesssage":
			return true
		}
	}
	return false
}

// sanitizeParsedMessageDirection impede que mensagens enviadas pelo atendente (IsFromMe)
// carreguem metadados residuais de anúncio citados/respondidos como se fossem clique inbound.
func sanitizeParsedMessageDirection(out *ParsedWhatsAppMessage) {
	if out == nil || !out.IsFromMe {
		return
	}
	out.HasMsgAdReferral = false
	out.CtwaClid = ""
	out.SourceID = ""
}

func applyContextInfo(ctxInfo *evoContextInfo, out *ParsedWhatsAppMessage) {
	if ctxInfo == nil || out == nil {
		return
	}
	// Mensagens encaminhadas por amigos/colegas nunca representam clique próprio em anúncio CTWA.
	if ctxInfo.IsForwarded || ctxInfo.ForwardingScore > 0 {
		return
	}
	clid := cleanValidCtwaClid(ctxInfo.CtwaClidDirect)
	if clid != "" {
		out.HasMsgAdReferral = true
		if out.CtwaClid == "" {
			out.CtwaClid = clid
		}
	}
	explicitAdCtx := clid != "" || isExplicitMetaAdConversion(ctxInfo.ConversionSrc, ctxInfo.ConversionFB)
	applyReferral(&ctxInfo.ExternalAdReply, explicitAdCtx, ctxInfo.ConversionApp, out)
}

// applyReferral aceita externalAdReply/referral APENAS quando há prova inequívoca de anúncio
// Meta Click-to-WhatsApp, bloqueando Reels/posts orgânicos do Instagram, vídeos do YouTube/TikTok,
// prévias de links ricos, itens de catálogo e links wa.me comuns (click_to_chat_link).
func applyReferral(ref *evoAdReferral, explicitAdCtx bool, fallbackApp string, out *ParsedWhatsAppMessage) {
	if ref == nil || out == nil {
		return
	}
	clid := cleanValidCtwaClid(util.FirstNonEmpty(ref.CtwaClid, ref.CtwaClidSnake))
	sid := cleanValidMetaAdSourceID(util.FirstNonEmpty(ref.SourceID, ref.SourceIdAlt, ref.SourceIDSnake))
	srcType := strings.ToLower(strings.TrimSpace(util.FirstNonEmpty(ref.SourceType, ref.SourceTypeSnake)))

	isVerifiedAd := false
	if clid != "" {
		isVerifiedAd = true
	} else if sid != "" && (srcType == "ad" || (srcType == "" && explicitAdCtx)) {
		isVerifiedAd = true
	}
	if !isVerifiedAd {
		return
	}

	out.HasMsgAdReferral = true
	if clid != "" && out.CtwaClid == "" {
		out.CtwaClid = clid
	}
	if sid != "" && out.SourceID == "" {
		out.SourceID = sid
	}
	if surl := strings.TrimSpace(util.FirstNonEmpty(ref.SourceURL, ref.SourceUrlAlt)); surl != "" && out.SourceURL == "" {
		out.SourceURL = surl
	}
	if app := normalizeAdSourceApp(util.FirstNonEmpty(ref.SourceApp, ref.SourceAppAlt, fallbackApp)); app != "" && out.SourceApp == "" {
		out.SourceApp = app
	}
	if thumb := strings.TrimSpace(util.FirstNonEmpty(ref.ThumbnailURL, ref.OriginalImageURL, ref.MediaURL)); thumb != "" && out.AdThumbnailURL == "" {
		out.AdThumbnailURL = thumb
	}
}

func isExplicitMetaAdConversion(convSrc, convFB string) bool {
	for _, raw := range []string{convSrc, convFB} {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "ctwa_ad", "fb_ads":
			return true
		}
	}
	return false
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
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func cleanValidCtwaClid(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) < 8 || len(s) > 512 || isStandardUUID(s) {
		return ""
	}
	hasLetter, hasDigit := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			hasLetter = true
		case c >= '0' && c <= '9':
			hasDigit = true
		case c == '+' || c == '/' || c == '=' || c == '.' || c == '_' || c == '-':
		default:
			return ""
		}
	}
	if !hasLetter || !hasDigit {
		return ""
	}
	return s
}

func cleanValidMetaAdSourceID(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) < 12 || len(s) > 22 {
		return ""
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return ""
		}
	}
	if (len(s) == 12 || len(s) == 13) && strings.HasPrefix(s, "55") {
		return ""
	}
	return s
}

// rxProtocol exige delimitador explícito (: ou #) após a palavra-chave para jamais capturar
// frases naturais em português como "Segue n.f referente ao serviço" ou "protocolo enviado".
var rxProtocol = regexp.MustCompile(`(?i)(?:^|[\s*_(\[])(?:protocolo(?:\s+de\s+atendimento)?|c[oó]d(?:igo)?\s+de\s+atendimento|ref(?:er[eê]ncia)?)\*?\s*[:#][\s*]*([A-Za-z0-9_-]{8,36})\*?(?:$|[\s*_.,;!?)\]])`)

func ExtractProtocolCode(text string) string {
	matches := rxProtocol.FindAllStringSubmatch(text, 4)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		cand := strings.TrimSpace(m[1])
		if isValidTrackFindProtocolToken(cand) {
			return cand
		}
	}
	return ""
}

func isValidTrackFindProtocolToken(tok string) bool {
	if len(tok) < 8 || len(tok) > 36 || isStandardUUID(tok) {
		return false
	}
	hasUpper, hasLower, hasDigit, allUpperAlpha := false, false, false, true
	for i := 0; i < len(tok); i++ {
		c := tok[i]
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
			allUpperAlpha = false
		case c >= '0' && c <= '9':
			hasDigit = true
			allUpperAlpha = false
		default:
			allUpperAlpha = false
		}
	}
	// Rejeita protocolos puramente numéricos (bancos, operadoras, prefeituras, notas fiscais).
	if !hasUpper && !hasLower {
		return false
	}
	// Sem dígito, só aceita o formato legado exato de 10 letras maiúsculas [A-Z]{10} do GenerateID10;
	// palavras comuns em português ("pagamento", "orcamento", "confirmado") são 100% bloqueadas.
	if !hasDigit {
		return len(tok) == 10 && allUpperAlpha
	}
	return true
}
