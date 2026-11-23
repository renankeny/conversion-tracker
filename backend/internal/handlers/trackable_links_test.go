package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestIsLinkPreviewCrawler(t *testing.T) {
	bots := []struct {
		name string
		ua   string
		hdr  string
	}{
		{"facebookexternalhit", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)", ""},
		{"whatsapp_preview", "WhatsApp/2.23.20.0", ""},
		{"telegram_preview", "TelegramBot (like TwitterBot)", ""},
		{"prefetch_header", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)", "prefetch"},
		{"empty_ua", "", ""},
	}
	for _, tc := range bots {
		req := httptest.NewRequest("GET", "/w/bio-insta", nil)
		req.Header.Set("User-Agent", tc.ua)
		if tc.hdr != "" {
			req.Header.Set("Sec-Purpose", tc.hdr)
		}
		if !IsLinkPreviewCrawler(req) {
			t.Errorf("%s should be detected as preview crawler", tc.name)
		}
	}

	humans := []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Instagram 326.0.0.18.90",
		"Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Mobile Safari/537.36 [FB_IAB/FB4A;FBAV/458.0.0.35.109;]",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/124.0.0.0 Safari/537.36",
	}
	for _, ua := range humans {
		req := httptest.NewRequest("GET", "/w/bio-insta", nil)
		req.Header.Set("User-Agent", ua)
		if IsLinkPreviewCrawler(req) {
			t.Errorf("human browser UA should NOT be marked as preview crawler: %s", ua)
		}
	}
}

func TestSlugifyAndExtractOwnerJID(t *testing.T) {
	if got := slugifyTrackableLink("Bio do Instagram! (Oferta #1)"); got != "bio-do-instagram-oferta-1" {
		t.Fatalf("unexpected slugify result: %q", got)
	}
	if got := extractPhoneFromOwnerJID("5511998887777:14@s.whatsapp.net"); got != "5511998887777" {
		t.Fatalf("unexpected phone from owner_jid: %q", got)
	}
	code := domain.GenerateTrackableLinkSlug(14)
	if len(code) != 10 || !strings.HasPrefix(code, "14") {
		t.Fatalf("expected 10-digit code starting with account_id 14, got %q", code)
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			t.Fatalf("expected numeric-only 10-digit code, got %q", code)
		}
	}
}

func TestTrackableLinkAllowedChannelsAndPublicURL(t *testing.T) {
	l := &domain.TrackableLink{
		Slug:              "bio-insta",
		TrackingHost:      "trk.meudominio.com.br",
		SendEventsEnabled: true,
		SendMeta:          true,
		SendTikTok:        false,
		SendGoogleAds:     true,
		SendGA4:           false,
	}
	chans := l.AllowedChannels()
	if len(chans) != 2 || chans[0] != "meta" || chans[1] != "google_ads" {
		t.Fatalf("unexpected allowed channels: %#v", chans)
	}
	if u := l.BuildPublicURL("https://api.sistema.com/api/v1"); u != "https://trk.meudominio.com.br/w/bio-insta" {
		t.Fatalf("unexpected CNAME public URL: %s", u)
	}
	l.TrackingHost = ""
	if u := l.BuildPublicURL("https://s.trackeado.com.br"); u != "https://s.trackeado.com.br/w/bio-insta" {
		t.Fatalf("unexpected app domain public URL: %s", u)
	}
	req := httptest.NewRequest("GET", "http://trackeado-api:8080/w/bio-insta?utm_source=ig", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "s.trackeado.com.br")
	if full := buildRequestFullURL(req, ""); full != "https://s.trackeado.com.br/w/bio-insta?utm_source=ig" {
		t.Fatalf("unexpected forwarded full URL: %s", full)
	}
	l.SendEventsEnabled = false
	if len(l.AllowedChannels()) != 0 {
		t.Fatalf("expected empty allowed channels when send_events_enabled=false")
	}
}

func TestRedirectBridgeHTMLContainsProtocolRef(t *testing.T) {
	rec := httptest.NewRecorder()
	writeRedirectBridgeHTML(rec, bridgeRenderParams{
		Title:              "Bio Instagram",
		Phone:              "5511999998888",
		PrefillMessage:     "Olá, vim pelo perfil!",
		ProtocolCode:       "AB12CD34EF",
		MetaBrowserPixelID: "1234567890",
		ExternalID:         "USR1234567",
	})
	body := rec.Body.String()
	if !strings.Contains(body, "Protocolo%3A+AB12CD34EF") {
		t.Fatalf("expected bridge HTML to embed Protocolo: AB12CD34EF like Landing Pages, got: %s", body)
	}
	if !strings.Contains(body, "1234567890") {
		t.Fatalf("expected bridge HTML to initialize Meta Pixel ID")
	}
}
