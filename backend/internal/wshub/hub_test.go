package wshub

import (
	"encoding/json"
	"testing"
)

func TestBroadcastTenantIsolation(t *testing.T) {
	hub := New(nil)

	ownerAll := &Client{UserID: 1, AccountID: 0, IsOwner: true, Send: make(chan []byte, 4)}
	ownerScoped := &Client{UserID: 1, AccountID: 2, IsOwner: true, Send: make(chan []byte, 4)}
	tenant2 := &Client{UserID: 10, AccountID: 2, IsOwner: false, Send: make(chan []byte, 4)}
	tenant3 := &Client{UserID: 11, AccountID: 3, IsOwner: false, Send: make(chan []byte, 4)}
	nonOwnerUnscoped := &Client{UserID: 12, AccountID: 0, IsOwner: false, Send: make(chan []byte, 4)}

	for _, c := range []*Client{ownerAll, ownerScoped, tenant2, tenant3, nonOwnerUnscoped} {
		hub.clients[c] = struct{}{}
	}

	// Evento sem account_id deve ser ignorado.
	hub.broadcast(LiveEvent{AccountID: 0, EventID: "ev_zero", EventName: "Purchase"})
	for _, c := range []*Client{ownerAll, ownerScoped, tenant2, tenant3, nonOwnerUnscoped} {
		if len(c.Send) != 0 {
			t.Fatalf("expected 0 messages when AccountID=0, got %d", len(c.Send))
		}
	}

	// Evento da conta 2 deve chegar apenas em ownerAll, ownerScoped(2) e tenant2.
	hub.broadcast(LiveEvent{AccountID: 2, EventID: "ev_2", EventName: "Purchase", PixelName: "px2"})
	if len(ownerAll.Send) != 1 || len(ownerScoped.Send) != 1 || len(tenant2.Send) != 1 {
		t.Fatalf("expected ownerAll, ownerScoped and tenant2 to receive event")
	}
	if len(tenant3.Send) != 0 {
		t.Fatalf("tenant3 must not receive account 2 event")
	}
	if len(nonOwnerUnscoped.Send) != 0 {
		t.Fatalf("non-owner with AccountID<=0 must never receive wildcard events")
	}

	// Custom event sem account_id nunca vaza globalmente.
	hub.broadcastCustom(customRedisEvent{AccountID: 0, Type: "events_updated", Data: json.RawMessage(`{"action":"deleted"}`)})
	if len(tenant3.Send) != 0 || len(nonOwnerUnscoped.Send) != 0 {
		t.Fatalf("custom event with AccountID=0 must be dropped")
	}

	// Custom event da conta 3 chega em ownerAll e tenant3, mas não em ownerScoped(2) nem tenant2.
	hub.broadcastCustom(customRedisEvent{AccountID: 3, Type: "events_updated", Data: json.RawMessage(`{"action":"deleted"}`)})
	if len(tenant3.Send) != 1 || len(ownerAll.Send) != 2 {
		t.Fatalf("expected tenant3 and ownerAll to receive custom event for account 3")
	}
	if len(ownerScoped.Send) != 1 || len(tenant2.Send) != 1 || len(nonOwnerUnscoped.Send) != 0 {
		t.Fatalf("custom event for account 3 leaked to another client")
	}
}
