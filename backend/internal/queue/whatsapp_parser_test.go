package queue

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestStripHeavyBase64FromEventData(t *testing.T) {
	raw := json.RawMessage(`{"Info":{"ID":"MSG1","Chat":"5511999999999@s.whatsapp.net"},"Message":{"audioMessage":{"URL":"https://mmg.whatsapp.net/x","mimetype":"audio/ogg","seconds":12},"base64":"AAAA_HEAVY_PAYLOAD_ZZZZ"}}`)
	cleaned := StripHeavyBase64FromEventData(raw)
	if strings.Contains(string(cleaned), "AAAA_HEAVY_PAYLOAD_ZZZZ") {
		t.Fatalf("expected base64 to be stripped, got: %s", string(cleaned))
	}
	parsed, err := ParseEvolutionMessageEvent(cleaned)
	if err != nil || parsed == nil {
		t.Fatalf("expected valid parsed message, err=%v", err)
	}
	if parsed.MessageType != "audio" || !parsed.HasMedia || parsed.DurationSec != 12 {
		t.Fatalf("unexpected media parse: %+v", parsed)
	}
}

func TestResolveContactJID_ProtectsAgainstFromMeLIDSwap(t *testing.T) {
	lidStore := map[string]string{}
	lookup := func(lid string) string { return lidStore[lid] }
	save := func(lid, phone string) { lidStore[lid] = phone }

	// 1. Lead envia 1ª mensagem com @lid + SenderAlt contendo telefone real
	inMsg := &ParsedWhatsAppMessage{
		MessageID:    "IN_1",
		ChatJID:      "5511988887777@s.whatsapp.net",
		SenderJID:    "5511988887777:2@s.whatsapp.net",
		SenderAltJID: "998877665544@lid",
		IsFromMe:     false,
		Content:      "Olá! Protocolo: ABC1234567",
	}
	remote, phone, lid, skip := ResolveContactJID(inMsg, "5511911112222:1@s.whatsapp.net", lookup, save)
	if skip || remote != "5511988887777@s.whatsapp.net" || phone != "5511988887777" || lid != "998877665544" {
		t.Fatalf("unexpected incoming resolution: remote=%q phone=%q lid=%q skip=%v", remote, phone, lid, skip)
	}

	// 2. Vendedor responde pelo celular em conversa @lid onde whatsmeow trocou Chat pelo número do próprio vendedor!
	outMsg := &ParsedWhatsAppMessage{
		MessageID:       "OUT_1",
		ChatJID:         "5511911112222@s.whatsapp.net", // bug do whatsmeow.go quando IsFromMe=true
		SenderJID:       "5511911112222@s.whatsapp.net",
		RecipientAltJID: "998877665544@lid",
		IsFromMe:        true,
		Content:         "Oi! Tudo bem?",
	}
	remote2, phone2, lid2, skip2 := ResolveContactJID(outMsg, "5511911112222@s.whatsapp.net", lookup, save)
	if skip2 || remote2 != "5511988887777@s.whatsapp.net" || phone2 != "5511988887777" || lid2 != "998877665544" {
		t.Fatalf("expected outbound @lid reply to resolve to lead via RecipientAlt+lidStore, got remote=%q phone=%q lid=%q skip=%v", remote2, phone2, lid2, skip2)
	}

	// 3. Grupo @g.us e status@broadcast sempre ignorados
	grp := &ParsedWhatsAppMessage{MessageID: "G1", ChatJID: "12036302@g.us", Content: "oi"}
	if _, _, _, skipGrp := ResolveContactJID(grp, "", lookup, save); !skipGrp {
		t.Fatalf("expected group message to be skipped")
	}
}

func TestClassifyPortalOrigin_AllPortals(t *testing.T) {
	if got := ClassifyPortalOrigin("ARAk123", "", nil); got != domain.WhatsAppOriginMetaCTWA {
		t.Fatalf("expected meta_ads_ctwa, got %q", got)
	}
	if got := ClassifyPortalOrigin("", "", &domain.TrackedUser{FBC: "fb.1.123.abc"}); got != domain.WhatsAppOriginMetaLP {
		t.Fatalf("expected meta_ads_lp, got %q", got)
	}
	if got := ClassifyPortalOrigin("", "", &domain.TrackedUser{WBraid: "wb_999"}); got != domain.WhatsAppOriginGoogleAds {
		t.Fatalf("expected google_ads, got %q", got)
	}
	if got := ClassifyPortalOrigin("", "", &domain.TrackedUser{TTClid: "tt_111"}); got != domain.WhatsAppOriginTikTokAds {
		t.Fatalf("expected tiktok_ads, got %q", got)
	}
	if got := ClassifyPortalOrigin("", "", &domain.TrackedUser{Referrer: "https://l.instagram.com/", FirstURL: "https://meusite.com/bio"}); got != domain.WhatsAppOriginBioLinkOrganic {
		t.Fatalf("expected bio_link_organic, got %q", got)
	}
	if got := ClassifyPortalOrigin("", "", &domain.TrackedUser{FirstURL: "https://meusite.com/artigo"}); got != domain.WhatsAppOriginWebsiteOrganic {
		t.Fatalf("expected website_organic, got %q", got)
	}
}

