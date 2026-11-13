package meta

import (
	"encoding/json"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type ctwaParsed struct {
	Data []struct {
		EventName        string         `json:"event_name"`
		ActionSource     string         `json:"action_source"`
		MessagingChannel string         `json:"messaging_channel"`
		SourceURL        string         `json:"event_source_url"`
		UserData         map[string]any `json:"user_data"`
	} `json:"data"`
}

func buildParsed(t *testing.T, in domain.SendEventInput) ctwaParsed {
	t.Helper()
	raw, err := BuildCAPIPayload(in, "", "")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var p ctwaParsed
	if err := json.Unmarshal(raw, &p); err != nil || len(p.Data) != 1 {
		t.Fatalf("unexpected payload: %s", raw)
	}
	return p
}

func TestBuildPayload_CtwaWithoutPageIDIsNotBusinessMessaging(t *testing.T) {
	p := buildParsed(t, domain.SendEventInput{
		EventName: domain.EventPurchase, EventID: "e1", EventTime: 1700000000,
		ActionSource: "business_messaging", CtwaClid: "clid", IDWpp: "SUB123",
	})
	d := p.Data[0]
	if d.ActionSource != "system_generated" {
		t.Fatalf("expected system_generated, got %q", d.ActionSource)
	}
	if _, ok := d.UserData["ctwa_clid"]; ok {
		t.Fatal("ctwa_clid must not be sent without page_id")
	}
	if d.MessagingChannel != "" {
		t.Fatal("messaging_channel must not be sent")
	}
}

func TestBuildPayload_WebsiteSaleWithCtwaKeepsWebsite(t *testing.T) {
	p := buildParsed(t, domain.SendEventInput{
		EventName: domain.EventPurchase, EventID: "e2", EventTime: 1700000000,
		ActionSource: "website", CtwaClid: "clid", PageID: "123", SourceURL: "https://lp.example.com",
	})
	d := p.Data[0]
	if d.ActionSource != "website" || d.SourceURL == "" {
		t.Fatalf("expected website with url, got %q / %q", d.ActionSource, d.SourceURL)
	}
	if _, ok := d.UserData["ctwa_clid"]; ok {
		t.Fatal("ctwa_clid must not be sent on website events")
	}
}

func TestBuildPayload_ValidCtwaDefaultsWhatsAppChannel(t *testing.T) {
	p := buildParsed(t, domain.SendEventInput{
		EventName: domain.EventPurchase, EventID: "e3", EventTime: 1700000000,
		ActionSource: "business_messaging", CtwaClid: "clid", PageID: "123",
		SourceURL: "https://ignored.example.com", IDWpp: "SUB123",
	})
	d := p.Data[0]
	if d.ActionSource != "business_messaging" || d.MessagingChannel != "whatsapp" {
		t.Fatalf("expected business_messaging/whatsapp, got %q/%q", d.ActionSource, d.MessagingChannel)
	}
	if d.SourceURL != "" {
		t.Fatal("event_source_url must not be sent on business_messaging")
	}
	if _, ok := d.UserData["subscription_id"]; ok {
		t.Fatal("subscription_id must not be sent on business_messaging")
	}
}

