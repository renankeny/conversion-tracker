package domain

import "testing"

func TestWhatsAppEventTriggersNormalizationAndLegacySync(t *testing.T) {
	legacy := LegacyTriggersToEventTriggers("pix recebido, pago", "proposta enviada", "Schedule", "reunião marcada")
	if len(legacy) != 3 {
		t.Fatalf("expected 3 migrated triggers, got %d: %+v", len(legacy), legacy)
	}
	if legacy[0].EventName != EventPurchase || legacy[1].EventName != EventQualifiedLead || legacy[2].EventName != "Schedule" {
		t.Fatalf("unexpected migrated order/events: %+v", legacy)
	}

	inst := &WhatsAppInstance{
		EventTriggers: []WhatsAppEventTrigger{
			{EventName: " Purchase ", Trigger: " pix confirmado "},
			{EventName: "Purchase", Trigger: "pagamento aprovado"},
			{EventName: "Schedule", Trigger: "agenda confirmada"},
			{EventName: "InitiateCheckout", Trigger: "segue o link"},
			{EventName: "PropostaVip", Trigger: "proposta vip enviada"},
			{EventName: "Init", Trigger: "ignorar interno"},
			{EventName: "Schedule", Trigger: "   "},
		},
	}
	SyncLegacyTriggerColumns(inst)

	if len(inst.EventTriggers) != 5 {
		t.Fatalf("expected 5 valid normalized triggers, got %d: %+v", len(inst.EventTriggers), inst.EventTriggers)
	}
	if inst.PurchaseTrigger != "pix confirmado, pagamento aprovado" {
		t.Fatalf("expected joined purchase_trigger, got %q", inst.PurchaseTrigger)
	}
	if inst.LeadTrigger != "" {
		t.Fatalf("expected empty lead_trigger, got %q", inst.LeadTrigger)
	}
	if inst.CustomEventName != "Schedule" || inst.CustomEventTrigger != "agenda confirmada" {
		t.Fatalf("expected first custom/extra event mirrored, got %q=%q", inst.CustomEventName, inst.CustomEventTrigger)
	}

	// Ao remover todos os gatilhos, as colunas legadas devem ser limpas.
	inst.EventTriggers = []WhatsAppEventTrigger{}
	SyncLegacyTriggerColumns(inst)
	if inst.PurchaseTrigger != "" || inst.CustomEventName != "" || inst.CustomEventTrigger != "" {
		t.Fatalf("expected legacy columns cleared when triggers removed, got %+v", inst)
	}
}
