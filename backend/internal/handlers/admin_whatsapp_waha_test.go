package handlers

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestSyncWAHALiveStatusAndQRCode(t *testing.T) {
	prevWait := wahaStartingRetryWait
	wahaStartingRetryWait = 5 * time.Millisecond
	defer func() { wahaStartingRetryWait = prevWait }()

	var sessionCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "waha-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/sessions/cli-waha":
			n := atomic.AddInt32(&sessionCalls, 1)
			if n == 1 {
				_, _ = w.Write([]byte(`{"name":"cli-waha","status":"STARTING","config":{"webhooks":[{"url":"https://api.example/webhooks/waha/sec1","events":["message.any","session.status"]}]}}`))
				return
			}
			if n == 2 {
				_, _ = w.Write([]byte(`{"name":"cli-waha","status":"SCAN_QR_CODE"}`))
				return
			}
			_, _ = w.Write([]byte(`{"name":"cli-waha","status":"WORKING","me":{"id":"5511988887777@c.us","pushName":"Comercial WAHA"}}`))
		case "/api/cli-waha/auth/qr":
			b64 := base64.StdEncoding.EncodeToString([]byte("png-qr"))
			_, _ = w.Write([]byte(`{"mimetype":"image/png","data":"` + b64 + `"}`))
		case "/api/cli-waha/profile":
			_, _ = w.Write([]byte(`{"id":"5511988887777@c.us","name":"Comercial WAHA","picture":"https://cdn.example/waha.jpg"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	inst := &domain.WhatsAppInstance{
		ID:            20,
		Provider:      domain.WhatsAppProviderWAHA,
		InstanceName:  "cli-waha",
		WebhookSecret: "sec1",
		ServerURL:     srv.URL,
		ServerAPIKey:  "waha-secret",
	}

	qr, connected, err := fetchWAHAQRCode(context.Background(), nil, inst, "https://api.example")
	if err != nil || connected || !strings.HasPrefix(qr, "data:image/png;base64,") {
		t.Fatalf("expected QR after STARTING transition, got qr=%q connected=%v err=%v", qr, connected, err)
	}

	if err := syncWAHALiveStatus(context.Background(), nil, inst); err != nil {
		t.Fatalf("syncWAHALiveStatus failed: %v", err)
	}
	if inst.ConnectionStatus != "OPEN" || inst.OwnerJID != "5511988887777@s.whatsapp.net" || inst.ProfileName != "Comercial WAHA" || inst.ProfilePicURL != "https://cdn.example/waha.jpg" {
		t.Fatalf("unexpected synced WAHA instance: %+v", inst)
	}
}
