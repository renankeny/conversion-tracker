package queue

import (
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestShouldDispatchCTWALead_ImmediateMode(t *testing.T) {
	inst := &domain.WhatsAppInstance{CtwaMinInboundMessages: 1}
	if inst.EffectiveCtwaMinMessages() != 1 {
		t.Fatalf("expected effective min 1, got %d", inst.EffectiveCtwaMinMessages())
	}
	if shouldDispatchCTWALead(1, 1, false) {
		t.Fatal("outbound message must never trigger CTWA LeadSubmitted")
	}
	if !shouldDispatchCTWALead(1, 1, true) {
		t.Fatal("1st inbound message must trigger CTWA LeadSubmitted when configured for 1")
	}
}

func TestShouldDispatchCTWALead_ThreeMessagesMode(t *testing.T) {
	inst := &domain.WhatsAppInstance{CtwaMinInboundMessages: 3}
	if inst.EffectiveCtwaMinMessages() != 3 {
		t.Fatalf("expected effective min 3, got %d", inst.EffectiveCtwaMinMessages())
	}
	for count := 0; count < 3; count++ {
		if shouldDispatchCTWALead(3, count, true) {
			t.Fatalf("inbound count %d must NOT trigger CTWA LeadSubmitted when min=3", count)
		}
	}
	if !shouldDispatchCTWALead(3, 3, true) {
		t.Fatal("3rd inbound message must trigger CTWA LeadSubmitted when min=3")
	}
	if !shouldDispatchCTWALead(3, 4, true) {
		t.Fatal("subsequent inbound messages pass threshold check (deduplicated by ExistsSent)")
	}
}

func TestNormalizeCtwaMinMessages_FallbackToDefault(t *testing.T) {
	for _, invalid := range []int{-1, 0, 2, 5, 99} {
		if got := domain.NormalizeCtwaMinMessages(invalid); got != 1 {
			t.Fatalf("NormalizeCtwaMinMessages(%d) = %d, want 1", invalid, got)
		}
	}
	if got := domain.NormalizeCtwaMinMessages(3); got != 3 {
		t.Fatalf("NormalizeCtwaMinMessages(3) = %d, want 3", got)
	}
}

