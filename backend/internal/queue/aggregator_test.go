package queue

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestPurchaseAggregatorSessionAndDeduplication(t *testing.T) {
	mainInput := domain.SendEventInput{
		PixelName:   "main-pixel",
		EventName:   domain.EventPurchase,
		EventID:     "evt-main-1",
		OrderID:     "ORD-100",
		EventTime:   1700000000,
		Email:       " Buyer@Example.com ",
		Phone:       "(11) 98888-7777",
		ContentID:   "prod-main",
		ContentName: "Curso Principal",
		ItemType:    "main",
	}
	if got := NormalizeCustomerKey(mainInput); got != "em:buyer@example.com" {
		t.Fatalf("NormalizeCustomerKey = %q, want em:buyer@example.com", got)
	}

	key1 := SessionKey("main-pixel", mainInput)
	key2 := SessionKey("main-pixel", domain.SendEventInput{Email: "buyer@example.com"})
	if key1 != key2 || !strings.HasPrefix(key1, "agg_purchase:main-pixel:") {
		t.Fatalf("SessionKey mismatch: %q vs %q", key1, key2)
	}

	itemMain := buildAggregatedItem(mainInput, 197.00)
	session := newAggregatedSession("main-pixel", mainInput, 197.00, itemMain)
	if session.Count != 1 || session.TotalValue != 197.00 || len(session.Items) != 1 {
		t.Fatalf("unexpected initial session: %+v", session)
	}

	// Adiciona Order Bump do mesmo pedido + enriquecimento de FBP/GCLID
	bumpInput := domain.SendEventInput{
		OrderID:     "ORD-100",
		ContentID:   "prod-bump",
		ContentName: "Planilha Bônus",
		ItemType:    "order_bump",
		FBP:         "fb.1.1700000000.999",
		GCLID:       "EAIaIQobChMI",
	}
	itemBump := buildAggregatedItem(bumpInput, 47.00)
	appendSubsequentSessionItem(&session, bumpInput, 47.00, itemBump)

	// Adiciona Upsell em novo OrderID
	upsellInput := domain.SendEventInput{
		OrderID:     "ORD-101",
		ContentID:   "prod-upsell",
		ContentName: "Mentoria VIP",
		ItemType:    "main", // segundo evento sem tipo explícito vira upsell
	}
	itemUpsell := buildAggregatedItem(upsellInput, 497.00)
	appendSubsequentSessionItem(&session, upsellInput, 497.00, itemUpsell)

	if session.Count != 3 {
		t.Fatalf("session.Count = %d, want 3", session.Count)
	}
	if session.TotalValue != 741.00 {
		t.Fatalf("session.TotalValue = %.2f, want 741.00", session.TotalValue)
	}
	if len(session.OrderIDs) != 2 {
		t.Fatalf("expected deduplicated OrderIDs [ORD-100, ORD-101], got %v", session.OrderIDs)
	}
	if session.Items[2].ItemType != "upsell" {
		t.Fatalf("expected subsequent 'main' item to normalize to 'upsell', got %q", session.Items[2].ItemType)
	}
	if session.FBP != "fb.1.1700000000.999" || session.GCLID != "EAIaIQobChMI" {
		t.Fatalf("expected PII/click IDs to merge into session: fbp=%q gclid=%q", session.FBP, session.GCLID)
	}
}

func TestEvalAggregatePurchaseLuaGracefulErrorOnClosedRedis(t *testing.T) {
	if aggregatePurchaseScript == nil || aggregatePurchaseScript.Hash() == "" {
		t.Fatalf("expected non-empty SHA1 hash for aggregatePurchaseScript")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 50 * time.Millisecond,
		MaxRetries:  0,
	})
	defer rdb.Close()

	in := domain.SendEventInput{EventID: "evt-1", OrderID: "ORD-1", EventName: domain.EventPurchase}
	item := buildAggregatedItem(in, 97.0)
	_, _, _, err := evalAggregatePurchaseLua(context.Background(), rdb, "agg_purchase:test:1", "test", in, 97.0, item, 5*time.Minute)
	if err == nil {
		t.Fatalf("expected connection error when Redis is unreachable so caller falls back to direct enqueue")
	}
}
