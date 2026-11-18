package repo

import "testing"

func TestIsNonOrderPurchaseEventID(t *testing.T) {
	manualIDs := []string{
		"crm_pur:abc1234567",
		"crm:12:4:Purchase",
		"ai_sale:55:900",
		"wpp_manual_xyz9876543",
		"pur_evt_pix_123",
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
