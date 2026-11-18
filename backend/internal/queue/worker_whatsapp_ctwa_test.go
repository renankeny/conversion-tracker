package queue

import (
	"context"
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

func TestEffectiveStageTrigger_DefaultFunnelAndNone(t *testing.T) {
	stage0 := &domain.CRMStage{Name: "Novo Lead", OrderIndex: 0, TriggerEvent: "", TriggerEventCTWA: domain.EventLeadSubmitted}
	if got := domain.EffectiveStageTrigger(stage0, false); got != "" {
		t.Fatalf("Stage 0 Web trigger must be empty, got %q", got)
	}
	if got := domain.EffectiveStageTrigger(stage0, true); got != domain.EventLeadSubmitted {
		t.Fatalf("Stage 0 CTWA trigger must be LeadSubmitted, got %q", got)
	}

	stageDisabled := &domain.CRMStage{Name: "Novo Lead", OrderIndex: 0, TriggerEvent: domain.EventLead, TriggerEventCTWA: domain.CRMTriggerNone}
	if got := domain.EffectiveStageTrigger(stageDisabled, true); got != "" {
		t.Fatalf("Stage with trigger_event_ctwa='none' must return empty for CTWA, got %q", got)
	}
	if got := domain.EffectiveStageTrigger(stageDisabled, false); got != domain.EventLead {
		t.Fatalf("Stage with trigger_event_ctwa='none' must still fire Web trigger, got %q", got)
	}

	stageWon := &domain.CRMStage{Name: "Venda Fechada", OrderIndex: 4, TriggerEvent: domain.EventPurchase, TriggerEventCTWA: domain.EventPurchase}
	if domain.EffectiveStageTrigger(stageWon, false) != domain.EventPurchase || domain.EffectiveStageTrigger(stageWon, true) != domain.EventPurchase {
		t.Fatalf("Venda Fechada must fire Purchase for both Web and CTWA")
	}
}

func TestEvaluateAttendantTriggers_IgnoresInboundAndEscapesRegex(t *testing.T) {
	called := false
	enq := &Enqueuer{
		sendHook: func(_ context.Context, _ domain.SendEventInput) error {
			called = true
			return nil
		},
	}
	inst := &domain.WhatsAppInstance{PurchaseTrigger: "pix confirmado, R$ 197,00 (bônus+vip)"}
	u := &domain.TrackedUser{ID: 1, ExternalID: "wpp:5511999998888"}

	if err := evaluateAttendantTriggers(context.Background(), Deps{Enqueuer: enq}, inst, u, 10, "5511999998888", "m1", "pix confirmado!", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("inbound client message (isFromMe=false) must NEVER trigger attendant phrase triggers")
	}

	term, ok := matchPhraseTriggerWithTerm(inst.PurchaseTrigger, "Seu pagamento R$ 197,00 (BÔNUS+VIP) está liberado")
	if !ok || term != "R$ 197,00 (bônus+vip)" {
		t.Fatalf("expected regex metacharacters to match safely, got ok=%v term=%q", ok, term)
	}
}
