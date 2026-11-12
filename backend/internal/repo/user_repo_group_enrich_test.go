package repo

import (
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestApplyConsolidatedGroupMembers(t *testing.T) {
	// members ordenados por updated_at DESC (mais recente primeiro)
	newestDesktop := &domain.TrackedUser{
		ID:               19,
		ExternalID:       "3A80OIZZ4Z",
		GCLID:            "CjwKCAiA9vG6BhBzEiwA3xXxY7mKp2qR",
		FBP:              "fb.1.1794430000.111222",
		TTP:              "01JABCDEFGHJKMNPQRSTVWXYZ1",
		GA4ClientID:      "123456.789012",
		LastUTMSource:    "google",
		LastUTMMedium:    "cpc",
		LastUTMCampaign:  "campanha_teste_2026",
		FirstUTMSource:   "chatgpt.com",
		FirstUTMCampaign: "",
	}
	olderMobile := &domain.TrackedUser{
		ID:               316,
		ExternalID:       "BR1E9BYD65",
		Phone:            "5518997136750",
		FirstName:        "Renan",
		FBC:              "fb.1.1794400000.IwZXh0bgNhZW0",
		FirstUTMSource:   "facebook",
		FirstUTMMedium:   "cpc",
		FirstUTMCampaign: "campanha_inicial",
	}

	target := *olderMobile // card do CRM originalmente ligado ao ID 316
	applyConsolidatedGroupMembers(&target, []*domain.TrackedUser{newestDesktop, olderMobile})

	if target.GCLID != newestDesktop.GCLID {
		t.Fatalf("expected consolidated GCLID %q, got %q", newestDesktop.GCLID, target.GCLID)
	}
	if target.FBC != olderMobile.FBC {
		t.Fatalf("expected preserved FBC %q, got %q", olderMobile.FBC, target.FBC)
	}
	if target.FBP != newestDesktop.FBP {
		t.Fatalf("expected consolidated FBP %q, got %q", newestDesktop.FBP, target.FBP)
	}
	if target.TTP != newestDesktop.TTP {
		t.Fatalf("expected consolidated TTP %q, got %q", newestDesktop.TTP, target.TTP)
	}
	if target.GA4ClientID != newestDesktop.GA4ClientID {
		t.Fatalf("expected consolidated GA4ClientID %q, got %q", newestDesktop.GA4ClientID, target.GA4ClientID)
	}
	if target.LastUTMSource != "google" || target.LastUTMCampaign != "campanha_teste_2026" {
		t.Fatalf("expected latest google UTM, got source=%q campaign=%q", target.LastUTMSource, target.LastUTMCampaign)
	}
}

