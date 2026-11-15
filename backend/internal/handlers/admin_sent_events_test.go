package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func TestChiURLParamEncoding(t *testing.T) {
	r := chi.NewRouter()
	var captured string
	r.Get("/test/{event_id}", func(w http.ResponseWriter, r *http.Request) {
		raw := chi.URLParam(r, "event_id")
		if unescaped, err := url.PathUnescape(raw); err == nil && unescaped != "" {
			captured = unescaped
		} else {
			captured = raw
		}
		w.WriteHeader(http.StatusOK)
	})

	rawURL := "/test/ctwa%3AAfgXdCAaFOpZsTzywANCp7ri0t99XU"
	req := httptest.NewRequest("GET", rawURL, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expected := "ctwa:AfgXdCAaFOpZsTzywANCp7ri0t99XU"
	if captured != expected {
		t.Fatalf("expected unescaped %q, got %q", expected, captured)
	}
}

func TestParseListFilterChannelSearchAndToMinute(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/sent-events/grouped?channel=ga4&search=joao&to=2026-11-15T14:30:00Z",
		nil,
	)
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{
		UserID:    5,
		Role:      "admin",
		TokenType: auth.TokenTypeHuman,
		AccountID: 7,
	}))

	f, err := parseListFilter(req)
	if err != nil {
		t.Fatalf("parseListFilter failed: %v", err)
	}
	if f.AccountID != 7 || f.Channel != "ga4" || f.Search != "joao" {
		t.Fatalf("unexpected filter fields: %+v", f)
	}
	wantTo := time.Date(2026, 11, 15, 14, 31, 0, 0, time.UTC)
	if !f.To.Equal(wantTo) {
		t.Fatalf("got To=%s, want %s", f.To.Format(time.RFC3339), wantTo.Format(time.RFC3339))
	}

	reqExact := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sent-events?to=2026-11-15T14:30:45Z", nil)
	fExact, err := parseListFilter(reqExact)
	if err != nil {
		t.Fatalf("parseListFilter exact failed: %v", err)
	}
	wantExact := time.Date(2026, 11, 15, 14, 30, 45, 0, time.UTC)
	if !fExact.To.Equal(wantExact) {
		t.Fatalf("got exact To=%s, want %s", fExact.To.Format(time.RFC3339), wantExact.Format(time.RFC3339))
	}
}

func TestResolveEventRefs(t *testing.T) {
	refs := resolveEventRefs(
		[]repo.EventRef{{EventID: " ev1 ", PixelName: " px1 "}},
		[]string{"ignored"},
	)
	if len(refs) != 1 || refs[0].EventID != "ev1" || refs[0].PixelName != "px1" {
		t.Fatalf("unexpected items refs: %+v", refs)
	}

	legacy := resolveEventRefs(nil, []string{" ev_a ", "", "ev_b"})
	if len(legacy) != 2 || legacy[0].EventID != "ev_a" || legacy[0].PixelName != "" || legacy[1].EventID != "ev_b" {
		t.Fatalf("unexpected legacy refs: %+v", legacy)
	}
}

func TestWSEventsRejectsQueryTokenAndRevokedSession(t *testing.T) {
	jwtSvc := auth.NewJWT("ws-test-secret-key-with-32-chars!", 24)
	tickets := cache.NewWSTicketStore(nil)
	sessions := cache.NewSessionStore(nil)
	wsH := NewWSEvents(wshub.New(nil), jwtSvc)
	wsH.SetSessionDeps(tickets, sessions)

	tok, err := jwtSvc.Issue(42, "u@test.com", "admin", 3, nil, nil)
	if err != nil {
		t.Fatalf("issue jwt: %v", err)
	}

	// Quando tickets != nil, ?token= na query string deve ser rejeitado (401).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/events?token="+tok, nil)
	rec := httptest.NewRecorder()
	wsH.Handle(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when passing ?token= with ticket store enabled, got %d", rec.Code)
	}

	// Ticket emitido a partir de sessão revogada deve ser rejeitado no handshake (401).
	ticketID, err := tickets.Issue(context.Background(), 42, 3, "admin", false, "jti-revoked", time.Now().Unix())
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	sessions.RevokeToken(context.Background(), "jti-revoked", 5*time.Minute)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/ws/events?ticket="+ticketID, nil)
	rec2 := httptest.NewRecorder()
	wsH.Handle(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked session ticket, got %d", rec2.Code)
	}
}
