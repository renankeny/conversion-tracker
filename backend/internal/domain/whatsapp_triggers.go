package domain

import "strings"

const maxWhatsAppEventTriggers = 50

// WhatsAppEventTrigger define um gatilho de evento acionado por termos enviados pelo atendente.
type WhatsAppEventTrigger struct {
	EventName string `json:"event_name"`
	Trigger   string `json:"trigger"`
}

func NormalizeWhatsAppEventTriggers(items []WhatsAppEventTrigger) []WhatsAppEventTrigger {
	if len(items) == 0 {
		return []WhatsAppEventTrigger{}
	}
	out := make([]WhatsAppEventTrigger, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, it := range items {
		ev := strings.TrimSpace(it.EventName)
		tr := strings.TrimSpace(it.Trigger)
		if ev == "" || tr == "" || IsInternalEvent(ev) || strings.EqualFold(ev, "none") {
			continue
		}
		if len(ev) > 80 {
			ev = ev[:80]
		}
		key := strings.ToLower(ev) + "\x00" + strings.ToLower(tr)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, WhatsAppEventTrigger{
			EventName: ev,
			Trigger:   tr,
		})
		if len(out) >= maxWhatsAppEventTriggers {
			break
		}
	}
	return out
}

func LegacyTriggersToEventTriggers(purchaseTrigger, leadTrigger, customName, customTrigger string) []WhatsAppEventTrigger {
	var raw []WhatsAppEventTrigger
	if p := strings.TrimSpace(purchaseTrigger); p != "" {
		raw = append(raw, WhatsAppEventTrigger{EventName: EventPurchase, Trigger: p})
	}
	if l := strings.TrimSpace(leadTrigger); l != "" {
		raw = append(raw, WhatsAppEventTrigger{EventName: EventQualifiedLead, Trigger: l})
	}
	if cn, ct := strings.TrimSpace(customName), strings.TrimSpace(customTrigger); cn != "" && ct != "" {
		raw = append(raw, WhatsAppEventTrigger{EventName: cn, Trigger: ct})
	}
	return NormalizeWhatsAppEventTriggers(raw)
}

func (i *WhatsAppInstance) EffectiveEventTriggers() []WhatsAppEventTrigger {
	if i == nil {
		return []WhatsAppEventTrigger{}
	}
	if norm := NormalizeWhatsAppEventTriggers(i.EventTriggers); len(norm) > 0 {
		return norm
	}
	return LegacyTriggersToEventTriggers(i.PurchaseTrigger, i.LeadTrigger, i.CustomEventName, i.CustomEventTrigger)
}

// SyncLegacyTriggerColumns mantém as colunas legadas espelhadas a partir de EventTriggers.
func SyncLegacyTriggerColumns(inst *WhatsAppInstance) {
	if inst == nil {
		return
	}
	inst.EventTriggers = NormalizeWhatsAppEventTriggers(inst.EventTriggers)
	var purchaseParts, leadParts []string
	customName, customTrigger := "", ""

	for _, t := range inst.EventTriggers {
		switch {
		case strings.EqualFold(t.EventName, EventPurchase):
			purchaseParts = append(purchaseParts, t.Trigger)
		case strings.EqualFold(t.EventName, EventQualifiedLead) || strings.EqualFold(t.EventName, EventLead) || strings.EqualFold(t.EventName, EventLeadSubmitted):
			leadParts = append(leadParts, t.Trigger)
		case customName == "":
			customName = t.EventName
			customTrigger = t.Trigger
		case strings.EqualFold(customName, t.EventName):
			customTrigger = customTrigger + ", " + t.Trigger
		}
	}
	inst.PurchaseTrigger = strings.Join(purchaseParts, ", ")
	inst.LeadTrigger = strings.Join(leadParts, ", ")
	inst.CustomEventName = customName
	inst.CustomEventTrigger = customTrigger
}
