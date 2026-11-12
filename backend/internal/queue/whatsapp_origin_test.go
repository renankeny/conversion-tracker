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

