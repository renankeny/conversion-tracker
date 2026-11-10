package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func TestBuildConfirmedPurchaseEventID(t *testing.T) {
	orig := "evt_pix_12345"
	derived := BuildConfirmedPurchaseEventID(orig)
	if derived == orig {
		t.Fatalf("derived Purchase event_id must differ from original AddPaymentInfo event_id")
	}
	if derived != "pur_evt_pix_12345" {
		t.Fatalf("got %q, want pur_evt_pix_12345", derived)
	}
	if BuildConfirmedPurchaseEventID(derived) != derived {
		t.Fatalf("BuildConfirmedPurchaseEventID must be idempotent")
	}
}

func TestWebhookConstantTimeAuthAndDeferredContext(t *testing.T) {
	secret := "whsec_live_987654321"
	body := []byte(`{"event":"purchase_approved","order_id":"ORD-1"}`)

	// 1. Token simples exato vs divergente
	if !matchWebhookAttemptAgainstSecret(secret, webhookAuthAttempt{candidates: []string{"wrong", secret}}) {
		t.Fatalf("expected exact candidate to match secret")
	}
	if matchWebhookAttemptAgainstSecret(secret, webhookAuthAttempt{candidates: []string{"whsec_live_987654320"}}) {
		t.Fatalf("expected wrong token candidate to fail")
	}
	if matchWebhookAttemptAgainstSecret(secret, webhookAuthAttempt{}) {
		t.Fatalf("expected empty attempt to fail when secret is configured")
	}

	// 2. HMAC-SHA256 hex e base64
	hexSig := util.GenerateHMACSHA256(body, secret)
	if !matchWebhookAttemptAgainstSecret(secret, webhookAuthAttempt{rawBody: body, hmacSHA256: hexSig}) {
		t.Fatalf("expected valid HMAC-SHA256 hex to pass")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	b64Sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !matchWebhookAttemptAgainstSecret(secret, webhookAuthAttempt{rawBody: body, hmacSHA256: b64Sig}) {
		t.Fatalf("expected valid HMAC-SHA256 base64 to pass")
	}

	// 3. HMAC-SHA1 hex
	sha1Sig := util.GenerateHMACSHA1(body, secret)
	if !matchWebhookAttemptAgainstSecret(secret, webhookAuthAttempt{rawBody: body, hmacSHA1: sha1Sig}) {
		t.Fatalf("expected valid HMAC-SHA1 hex to pass")
	}

	// 4. Sem ?acc= na URL -> ValidatePlatformToken deve registrar tentativa no contexto da request
	wh := &Webhooks{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/cakto?token=tok_abc", nil)
	req.Header.Set("X-Webhook-Token", "hdr_xyz")
	if !wh.ValidatePlatformToken(req.Context(), req, 0, "cakto", "payload_secret") {
		t.Fatalf("deferred ValidatePlatformToken without accountID should defer rather than drop")
	}
	att, ok := req.Context().Value(webhookAuthAttemptKey{}).(webhookAuthAttempt)
	if !ok || att.platform != "cakto" {
		t.Fatalf("expected deferred webhookAuthAttempt stored in request context")
	}
	if !matchWebhookAttemptAgainstSecret("hdr_xyz", att) || !matchWebhookAttemptAgainstSecret("tok_abc", att) || !matchWebhookAttemptAgainstSecret("payload_secret", att) {
		t.Fatalf("expected payload, header, and query candidates to be captured in deferred attempt")
	}
}

func TestSafeLogoOrFaviconURLAndCanAccessAccount(t *testing.T) {
	if SafeLogoOrFaviconURL("javascript:alert(1)") {
		t.Fatalf("javascript: URL must be rejected")
	}
	if SafeLogoOrFaviconURL("data:image/svg+xml;base64,PHN2Zz48c2NyaXB0PmFsZXJ0KDEpPC9zY3JpcHQ+PC9zdmc+") {
		t.Fatalf("inline SVG data URI must be rejected to prevent stored XSS")
	}
	if !SafeLogoOrFaviconURL("https://cdn.example.com/logo.png") {
		t.Fatalf("https PNG URL should be allowed")
	}
	if !SafeLogoOrFaviconURL("data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUA") {
		t.Fatalf("data:image/png;base64 URI should be allowed")
	}

	apiClaims := &auth.Claims{Role: "member", TokenType: auth.TokenTypeAPIKey, AccountID: 10}
	if !CanAccessAccount(context.Background(), nil, apiClaims, 10) {
		t.Fatalf("api key should access its own account 10")
	}
	if CanAccessAccount(context.Background(), nil, apiClaims, 11) {
		t.Fatalf("api key must not access account 11")
	}
}

