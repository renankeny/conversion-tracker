package queue

import (
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestIsChannelAllowed(t *testing.T) {
	if !isChannelAllowed(nil, "meta") || !isChannelAllowed([]string{}, "google_ads") {
		t.Fatal("empty allowed list must allow all channels by default")
	}
	subset := []string{"meta", "ga4"}
	if !isChannelAllowed(subset, "meta") || !isChannelAllowed(subset, "ga4") {
		t.Fatal("expected meta and ga4 to be allowed")
	}
	if isChannelAllowed(subset, "tiktok") || isChannelAllowed(subset, "google_ads") {
		t.Fatal("expected tiktok and google_ads to be blocked when not in AllowedChannels")
	}
}

func TestResolveEffectiveTrackableLinkIDPrecedence(t *testing.T) {
	idClick := int64(10)
	idChat := int64(20)
	idUser := int64(30)

	click := &domain.TrackableLinkClick{LinkID: idClick}
	chat := &domain.WhatsAppChat{TrackableLinkID: &idChat}
	user := &domain.TrackedUser{TrackableLinkID: &idUser}

	if got := resolveEffectiveTrackableLinkID(click, nil, chat, user); got == nil || *got != idClick {
		t.Fatalf("expected click link_id %d to take precedence, got %v", idClick, got)
	}
	if got := resolveEffectiveTrackableLinkID(nil, nil, chat, user); got == nil || *got != idChat {
		t.Fatalf("expected existing chat link_id %d, got %v", idChat, got)
	}
	if got := resolveEffectiveTrackableLinkID(nil, nil, nil, user); got == nil || *got != idUser {
		t.Fatalf("expected user link_id %d, got %v", idUser, got)
	}
}

func TestExtractProtocolCodeMatchesBridgeRef(t *testing.T) {
	msg := "Olá! Vim pelo Instagram e gostaria de mais informações.\nProtocolo: A9F2K7M1Z0"
	if got := ExtractProtocolCode(msg); got != "A9F2K7M1Z0" {
		t.Fatalf("expected ExtractProtocolCode to extract A9F2K7M1Z0, got %q", got)
	}
}
