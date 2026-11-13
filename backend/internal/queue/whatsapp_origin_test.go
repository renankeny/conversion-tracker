package queue

import (
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestClassifyPortalOrigin(t *testing.T) {
	t.Run("google ads with gclid even if older fbc exists", func(t *testing.T) {
		u := &domain.TrackedUser{
			FBC:           "fb.1.1700000000.old_fbclid",
			GCLID:         "CjwKCAiA9vG6BhBzEiwA3xXxY7mKp2qR",
			LastUTMSource: "google",
			LastUTMMedium: "cpc",
		}
		if got := ClassifyPortalOrigin("", "", u); got != domain.WhatsAppOriginGoogleAds {
			t.Fatalf("expected %q, got %q", domain.WhatsAppOriginGoogleAds, got)
		}
	})

	t.Run("google ads paid utm fallback when browser strips gclid", func(t *testing.T) {
		u := &domain.TrackedUser{
			FirstUTMSource:   "google",
			FirstUTMMedium:   "cpc",
			FirstUTMCampaign: "campanha_teste_2026",
			LastUTMSource:    "google",
			LastUTMMedium:    "cpc",
		}
		if got := ClassifyPortalOrigin("", "", u); got != domain.WhatsAppOriginGoogleAds {
			t.Fatalf("expected %q, got %q", domain.WhatsAppOriginGoogleAds, got)
		}
	})

	t.Run("tiktok ads paid utm fallback", func(t *testing.T) {
		u := &domain.TrackedUser{
			LastUTMSource: "tiktok",
			LastUTMMedium: "cpc",
		}
		if got := ClassifyPortalOrigin("", "", u); got != domain.WhatsAppOriginTikTokAds {
			t.Fatalf("expected %q, got %q", domain.WhatsAppOriginTikTokAds, got)
		}
	})

	t.Run("direct ctwa overrides everything", func(t *testing.T) {
		u := &domain.TrackedUser{GCLID: "CjwKCAi"}
		if got := ClassifyPortalOrigin("ctwa_123", "", u); got != domain.WhatsAppOriginMetaCTWA {
			t.Fatalf("expected %q, got %q", domain.WhatsAppOriginMetaCTWA, got)
		}
	})
}

func TestResolveMessagePortalFields(t *testing.T) {
	existing := &domain.WhatsAppChat{ID: 42, CtwaClid: "old_ctwa", SourceID: "old_ad"}

	t.Run("normal follow-up message on subsequent day returns empty", func(t *testing.T) {
		msg := &ParsedWhatsAppMessage{Content: "Bom dia, tudo bem?", IsFromMe: false}
		clid, sid, proto := resolveMessagePortalFields(msg, existing, ExtractProtocolCode(msg.Content))
		if clid != "" || sid != "" || proto != "" {
			t.Fatalf("expected empty portal fields, got clid=%q sid=%q proto=%q", clid, sid, proto)
		}
	})

	t.Run("provider echoing same chat-level ctwa without message referral returns empty", func(t *testing.T) {
		msg := &ParsedWhatsAppMessage{Content: "Vou fazer o pix agora", CtwaClid: "old_ctwa", SourceID: "old_ad", HasMsgAdReferral: false}
		clid, sid, proto := resolveMessagePortalFields(msg, existing, "")
		if clid != "" || sid != "" || proto != "" {
			t.Fatalf("expected empty portal fields on echoed chat ctwa, got clid=%q sid=%q proto=%q", clid, sid, proto)
		}
	})

	t.Run("new ctwa referral on message counts even on existing chat", func(t *testing.T) {
		msg := &ParsedWhatsAppMessage{Content: "Vi o outro anúncio", CtwaClid: "new_ctwa", SourceID: "new_ad", HasMsgAdReferral: true}
		clid, sid, proto := resolveMessagePortalFields(msg, existing, "")
		if clid != "new_ctwa" || sid != "new_ad" || proto != "" {
			t.Fatalf("unexpected portal fields: clid=%q sid=%q proto=%q", clid, sid, proto)
		}
	})

	t.Run("same or new protocol code from cookie counts on existing chat", func(t *testing.T) {
		msg := &ParsedWhatsAppMessage{Content: "Olá! Protocolo: ABC123COOKIE", IsFromMe: false}
		clid, sid, proto := resolveMessagePortalFields(msg, existing, ExtractProtocolCode(msg.Content))
		if clid != "" || sid != "" || proto != "ABC123COOKIE" {
			t.Fatalf("expected protocol ABC123COOKIE, got clid=%q sid=%q proto=%q", clid, sid, proto)
		}
	})
}

