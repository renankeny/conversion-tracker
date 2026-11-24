package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestSyncEvolutionLiveStatusSurvives2sReconnectGap(t *testing.T) {
	prevWait := evolutionReconnectRetryWait
	evolutionReconnectRetryWait = 5 * time.Millisecond
	defer func() { evolutionReconnectRetryWait = prevWait }()

	var statusCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/instance/status":
			n := atomic.AddInt32(&statusCalls, 1)
			if n == 1 {
				// Simula os 2s do ReconnectClient onde clientPointer foi removido.
				_, _ = w.Write([]byte(`{"data":{"connected":false,"loggedIn":false}}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"connected":true,"loggedIn":true,"myJid":"5511988887777@s.whatsapp.net","name":"Atendimento"}}`))
		case "/instance/info/evo-uuid-1":
			_, _ = w.Write([]byte(`{"data":{"id":"evo-uuid-1","jid":"5511988887777@s.whatsapp.net","connected":false,"profileName":"Atendimento"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	inst := &domain.WhatsAppInstance{
		ID:               10,
		EvolutionUUID:    "evo-uuid-1",
		InstanceToken:    "tok_1",
		ServerURL:        srv.URL,
		ServerAPIKey:     "adm_key",
		ConnectionStatus: "OPEN",
		OwnerJID:         "5511988887777@s.whatsapp.net",
		ProfileName:      "Atendimento",
		ProfilePicURL:    "https://cdn.example/avatar.jpg",
	}

	if err := syncEvolutionLiveStatus(context.Background(), nil, inst); err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}
	if inst.ConnectionStatus != "OPEN" {
		t.Fatalf("expected status OPEN after reconnect retry, got %s", inst.ConnectionStatus)
	}
	if inst.OwnerJID == "" || inst.ProfilePicURL == "" {
		t.Fatalf("expected session fields preserved, got owner=%q pic=%q", inst.OwnerJID, inst.ProfilePicURL)
	}
}

func TestSyncEvolutionLiveStatusPreservesPairedSessionWhenOffline(t *testing.T) {
	prevWait := evolutionReconnectRetryWait
	evolutionReconnectRetryWait = 5 * time.Millisecond
	defer func() { evolutionReconnectRetryWait = prevWait }()

	pairedJID := "5511988887777@s.whatsapp.net"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/instance/status":
			_, _ = w.Write([]byte(`{"data":{"connected":false,"loggedIn":false}}`))
		case "/instance/info/evo-uuid-2":
			_, _ = w.Write([]byte(`{"data":{"id":"evo-uuid-2","jid":"` + pairedJID + `","connected":false}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	inst := &domain.WhatsAppInstance{
		ID:               11,
		EvolutionUUID:    "evo-uuid-2",
		InstanceToken:    "tok_2",
		ServerURL:        srv.URL,
		ServerAPIKey:     "adm_key",
		ConnectionStatus: "OPEN",
		OwnerJID:         pairedJID,
		ProfileName:      "Suporte",
		ProfilePicURL:    "https://cdn.example/suporte.jpg",
	}

	if err := syncEvolutionLiveStatus(context.Background(), nil, inst); err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}
	if inst.ConnectionStatus != "DISCONNECTED" {
		t.Fatalf("expected paired offline instance -> DISCONNECTED, got %s", inst.ConnectionStatus)
	}
	if inst.OwnerJID != pairedJID || inst.ProfileName != "Suporte" || inst.ProfilePicURL == "" {
		t.Fatalf("expected paired metadata preserved, got %+v", inst)
	}

	// Quando deslogado de fato (JID vazio na Evolution Go), zera a sessão como CLOSED.
	pairedJID = ""
	if err := syncEvolutionLiveStatus(context.Background(), nil, inst); err != nil {
		t.Fatalf("unexpected sync error on logged out: %v", err)
	}
	if inst.ConnectionStatus != "CLOSED" || inst.OwnerJID != "" {
		t.Fatalf("expected unpaired instance -> CLOSED with cleared owner, got status=%s owner=%q", inst.ConnectionStatus, inst.OwnerJID)
	}
}
