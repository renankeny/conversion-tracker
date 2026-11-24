package repo

import (
	"testing"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
)

func TestIsNonOrderPurchaseEventID(t *testing.T) {
	manualIDs := []string{
		"crm_pur:abc1234567",
		"crm:12:4:Purchase",
		"ai_sale:55:900",
		"wpp_manual_xyz9876543",
		"pur_evt_pix_123",
		"web_pur:abc9876543",
		"3EB0ABCDEF:Purchase",
	}
	for _, id := range manualIDs {
		if !IsNonOrderPurchaseEventID(id) {
			t.Fatalf("expected IsNonOrderPurchaseEventID(%q) = true", id)
		}
	}

	realOrderIDs := []string{
		"HP1794829102",
		"ORD-2026-001",
		"ORD-2026-001:order_bump:PROD2",
		"kiwify_998877",
	}
	for _, id := range realOrderIDs {
		if IsNonOrderPurchaseEventID(id) {
			t.Fatalf("expected IsNonOrderPurchaseEventID(%q) = false for checkout order", id)
		}
	}
}

func TestMatchCrossSourcePurchase(t *testing.T) {
	now := time.Now()

	// 1. Página /obrigado sem order_id (random 10-char event_id) seguida do Webhook com order_id real -> deduplica!
	if !cache.MatchCrossSourcePurchase("A9B8C7D6E5", "", 197.00, now.Add(-2*time.Minute), "HP-ORDER-999", false, 197.00, now) {
		t.Fatal("expected browser /obrigado Purchase without order_id to deduplicate against gateway webhook")
	}

	// 2. CRM Kanban R$ 197 à vista seguido de Webhook parcelado 12x R$ 238,90 -> deduplica!
	if !cache.MatchCrossSourcePurchase("crm_pur:xyz1234567", "", 197.00, now.Add(-30*time.Minute), "KW-ORDER-555", false, 238.90, now) {
		t.Fatal("expected CRM manual sale to deduplicate against 12x installment gateway webhook")
	}

	// 3. Boleto/PIX pago 36h depois de baixa manual no WhatsApp -> deduplica dentro da janela de 72h!
	if !cache.MatchCrossSourcePurchase("ai_sale:10:99", "", 500.00, now.Add(-36*time.Hour), "CK-ORDER-777", false, 450.00, now) {
		t.Fatal("expected PIX/boleto webhook 36h later with 10% discount to deduplicate against AI sale")
	}

	// 4. Dois pedidos reais e distintos de gateway (upsell ou recompra no mesmo dia) -> NUNCA bloquear!
	if cache.MatchCrossSourcePurchase("ORD-MAIN-100", "ORD-MAIN-100", 197.00, now.Add(-3*time.Minute), "ORD-UPSELL-101", false, 197.00, now) {
		t.Fatal("distinct real checkout order_ids must NEVER be blocked")
	}

	// 5. Produto de ticket completamente diferente (R$ 97 vs R$ 497) -> não bloquear!
	if cache.MatchCrossSourcePurchase("crm_pur:lowticket", "", 97.00, now.Add(-1*time.Hour), "ORD-MENTORIA-1", false, 497.00, now) {
		t.Fatal("high-ticket purchase far above 35% tolerance must not be blocked by low-ticket manual entry")
	}

	// 6. Recompra legítima 5 dias (120h) depois -> não bloquear!
	if cache.MatchCrossSourcePurchase("crm_pur:old123", "", 197.00, now.Add(-120*time.Hour), "ORD-RENEW-2", false, 197.00, now) {
		t.Fatal("repeat purchase after >72h must not be deduplicated")
	}
}
