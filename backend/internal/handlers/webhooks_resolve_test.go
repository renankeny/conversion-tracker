package handlers

import "testing"

func TestWebhookEventID(t *testing.T) {
	cases := []struct{ order, item, product, want string }{
		{"ORD1", "", "P1", "ORD1"},
		{"ORD1", "main", "P1", "ORD1"},
		{"ORD1", "order_bump", "P2", "ORD1:order_bump:P2"},
		{"ORD1", "upsell", "P3", "ORD1:upsell:P3"},
	}
	for _, c := range cases {
		if got := webhookEventID(c.order, c.item, c.product); got != c.want {
			t.Fatalf("webhookEventID(%q,%q,%q)=%q want %q", c.order, c.item, c.product, got, c.want)
		}
	}
	if got := webhookEventID("", "", ""); len(got) != 10 {
		t.Fatalf("expected generated 10-char id, got %q", got)
	}
}

