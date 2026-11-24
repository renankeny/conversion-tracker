package queue

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestEvolutionTransientDisconnectUsesGracePeriod(t *testing.T) {
	ctx := context.Background()
	clearPendingWhatsAppDisconnect(ctx, nil, 42)
	defer clearPendingWhatsAppDisconnect(ctx, nil, 42)

	enqueued := 0
	var capturedDelay time.Duration
	enq := &Enqueuer{
		discCheckHook: func(_ context.Context, in WhatsAppVerifyDisconnectPayload, delay time.Duration) error {
			enqueued++
			capturedDelay = delay
			if in.InstanceID != 42 || in.Event != "Disconnected" {
				t.Fatalf("unexpected payload: %+v", in)
			}
			return nil
		},
	}

	inst := &domain.WhatsAppInstance{
		ID:               42,
		AccountID:        7,
		ConnectionStatus: "OPEN",
		OwnerJID:         "5511999999999@s.whatsapp.net",
		ProfileName:      "Comercial",
		ProfilePicURL:    "https://cdn.example/pic.jpg",
		Enabled:          true,
	}
	deps := Deps{Enqueuer: enq}

	if !HandleWhatsAppLifecycleEvent(ctx, deps, inst, "Disconnected", json.RawMessage(`{}`)) {
		t.Fatal("expected Disconnected event to be handled")
	}
	if inst.ConnectionStatus != "OPEN" {
		t.Fatalf("expected status to stay OPEN during grace period, got %s", inst.ConnectionStatus)
	}
	if enqueued != 1 || capturedDelay != whatsappDisconnectGracePeriod {
		t.Fatalf("expected 1 delayed check (%s), got count=%d delay=%s", whatsappDisconnectGracePeriod, enqueued, capturedDelay)
	}
	if !hasPendingWhatsAppDisconnect(ctx, nil, 42) {
		t.Fatal("expected pending disconnect marker to be set")
	}

	// Segundo Disconnected consecutivo não agenda tarefa duplicada.
	HandleWhatsAppLifecycleEvent(ctx, deps, inst, "Disconnected", json.RawMessage(`{}`))
	if enqueued != 1 {
		t.Fatalf("expected duplicate Disconnected to be deduplicated, got %d", enqueued)
	}

	// Evento Connected limpa o marcador pendente sem alterar o estado OPEN.
	HandleWhatsAppLifecycleEvent(ctx, deps, inst, "Connected", json.RawMessage(`{"jid":"5511999999999@s.whatsapp.net"}`))
	if hasPendingWhatsAppDisconnect(ctx, nil, 42) {
		t.Fatal("expected Connected to clear pending disconnect marker")
	}
	if inst.ConnectionStatus != "OPEN" {
		t.Fatalf("expected status OPEN after Connected, got %s", inst.ConnectionStatus)
	}
}

func TestEvolutionPermanentDisconnectsApplyImmediately(t *testing.T) {
	ctx := context.Background()
	enq := &Enqueuer{
		discCheckHook: func(_ context.Context, _ WhatsAppVerifyDisconnectPayload, _ time.Duration) error {
			t.Fatal("permanent disconnect must not schedule delayed check")
			return nil
		},
	}
	deps := Deps{Enqueuer: enq}

	instLogout := &domain.WhatsAppInstance{ID: 50, AccountID: 1, ConnectionStatus: "OPEN", Enabled: true}
	HandleWhatsAppLifecycleEvent(ctx, deps, instLogout, "LoggedOut", json.RawMessage(`{}`))
	if instLogout.ConnectionStatus != "CLOSED" {
		t.Fatalf("expected LoggedOut -> CLOSED, got %s", instLogout.ConnectionStatus)
	}

	instAuthFail := &domain.WhatsAppInstance{ID: 51, AccountID: 1, ConnectionStatus: "OPEN", Enabled: true}
	HandleWhatsAppLifecycleEvent(ctx, deps, instAuthFail, "ConnectFailure", json.RawMessage(`{"reason":"401"}`))
	if instAuthFail.ConnectionStatus != "CLOSED" {
		t.Fatalf("expected ConnectFailure 401 -> CLOSED, got %s", instAuthFail.ConnectionStatus)
	}

	instBan := &domain.WhatsAppInstance{ID: 52, AccountID: 1, ConnectionStatus: "OPEN", Enabled: true}
	HandleWhatsAppLifecycleEvent(ctx, deps, instBan, "TemporaryBan", json.RawMessage(`{}`))
	if instBan.ConnectionStatus != "DISCONNECTED" {
		t.Fatalf("expected TemporaryBan -> DISCONNECTED, got %s", instBan.ConnectionStatus)
	}
}

func TestVerifyAndConfirmEvolutionDisconnectLiveProbe(t *testing.T) {
	prevWait := evolutionVerifyRetryWait
	evolutionVerifyRetryWait = 5 * time.Millisecond
	defer func() { evolutionVerifyRetryWait = prevWait }()

	ctx := context.Background()
	online := true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/instance/status":
			if online {
				_, _ = w.Write([]byte(`{"data":{"connected":true,"loggedIn":true,"myJid":"5511999999999@s.whatsapp.net","name":"Vendas"}}`))
			} else {
				_, _ = w.Write([]byte(`{"data":{"connected":false,"loggedIn":false}}`))
			}
		case "/instance/info/uuid-1":
			_, _ = w.Write([]byte(`{"data":{"id":"uuid-1","jid":"5511999999999@s.whatsapp.net","connected":false}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	inst := &domain.WhatsAppInstance{
		ID:               99,
		AccountID:        1,
		EvolutionUUID:    "uuid-1",
		InstanceToken:    "tok_test",
		ServerURL:        srv.URL,
		ServerAPIKey:     "adm_test",
		ConnectionStatus: "OPEN",
		OwnerJID:         "5511999999999@s.whatsapp.net",
		Enabled:          true,
	}

	markPendingWhatsAppDisconnect(ctx, nil, inst.ID, time.Now().Unix())
	if err := verifyAndConfirmEvolutionDisconnect(ctx, Deps{}, inst, "Disconnected", true); err != nil {
		t.Fatalf("unexpected error when recovered: %v", err)
	}
	if inst.ConnectionStatus != "OPEN" || hasPendingWhatsAppDisconnect(ctx, nil, inst.ID) {
		t.Fatalf("expected recovered instance to remain OPEN and clear pending key, got status=%s", inst.ConnectionStatus)
	}

	online = false
	markPendingWhatsAppDisconnect(ctx, nil, inst.ID, time.Now().Unix())
	if err := verifyAndConfirmEvolutionDisconnect(ctx, Deps{}, inst, "Disconnected", true); err != nil {
		t.Fatalf("unexpected error when offline: %v", err)
	}
	if inst.ConnectionStatus != "DISCONNECTED" {
		t.Fatalf("expected offline paired instance -> DISCONNECTED, got %s", inst.ConnectionStatus)
	}
}
