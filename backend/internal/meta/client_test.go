package meta

import (
	"encoding/json"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestBuildCAPIPayload_BusinessMessagingLeadMapsToLeadSubmitted(t *testing.T) {
	raw, err := BuildCAPIPayload(domain.SendEventInput{
		EventName:        domain.EventLead,
		EventID:          "evt-1",
		EventTime:        1700000000,
		ActionSource:     "business_messaging",
		MessagingChannel: "whatsapp",
		CtwaClid:         "ctwa_abc",
		PageID:           "12345",
		Phone:            "5511999998888",
	}, "", "")
	if err != nil {
		t.Fatalf("BuildCAPIPayload error: %v", err)
	}

	var parsed struct {
		Data []struct {
			EventName    string `json:"event_name"`
			ActionSource string `json:"action_source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Data) != 1 {
		t.Fatalf("unexpected payload: %s", string(raw))
	}
	if parsed.Data[0].EventName != domain.EventLeadSubmitted {
		t.Fatalf("expected %q for business_messaging, got %q", domain.EventLeadSubmitted, parsed.Data[0].EventName)
	}

	webRaw, err := BuildCAPIPayload(domain.SendEventInput{
		EventName:    domain.EventLead,
		EventID:      "evt-2",
		EventTime:    1700000000,
		ActionSource: "website",
		SourceURL:    "https://lp.example.com",
	}, "", "")
	if err != nil {
		t.Fatalf("BuildCAPIPayload web error: %v", err)
	}
	if err := json.Unmarshal(webRaw, &parsed); err != nil || len(parsed.Data) != 1 {
		t.Fatalf("unexpected web payload: %s", string(webRaw))
	}
	if parsed.Data[0].EventName != domain.EventLead {
		t.Fatalf("expected %q for website, got %q", domain.EventLead, parsed.Data[0].EventName)
	}
}

