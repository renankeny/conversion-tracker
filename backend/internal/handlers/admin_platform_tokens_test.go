package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func withChiPlatformParam(r *http.Request, platform string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("platform", platform)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestPlatformTokensRBACAndValidation(t *testing.T) {
	h := NewPlatformTokens(nil)

	// 1. Viewer / employee sem permissão deve receber 403 Forbidden
	viewerClaims := &auth.Claims{UserID: 5, Role: "employee", AccountID: 1, Permissions: map[string]any{"edit_pixels": false}}
	reqViewer := httptest.NewRequest(http.MethodPut, "/admin/integrations/platforms/tokens/hotmart", bytes.NewBufferString(`{"token_secret":"abc"}`))
	reqViewer = withChiPlatformParam(reqViewer.WithContext(auth.WithClaims(reqViewer.Context(), viewerClaims)), "hotmart")
	recViewer := httptest.NewRecorder()
	h.SetToken(recViewer, reqViewer)
	if recViewer.Code != http.StatusForbidden {
		t.Fatalf("viewer SetToken: got %d, want 403", recViewer.Code)
	}

	// 2. Plataforma desconhecida deve receber 400 Bad Request
	adminClaims := &auth.Claims{UserID: 1, Role: "admin", AccountID: 1}
	reqBadPlat := httptest.NewRequest(http.MethodPut, "/admin/integrations/platforms/tokens/desconhecida", bytes.NewBufferString(`{"token_secret":"abc"}`))
	reqBadPlat = withChiPlatformParam(reqBadPlat.WithContext(auth.WithClaims(reqBadPlat.Context(), adminClaims)), "desconhecida")
	recBadPlat := httptest.NewRecorder()
	h.SetToken(recBadPlat, reqBadPlat)
	if recBadPlat.Code != http.StatusBadRequest {
		t.Fatalf("unknown platform SetToken: got %d, want 400", recBadPlat.Code)
	}

	// 3. Owner em modo global (AccountID=0) sem account_id informado recebe 400 em vez de FK 500
	ownerClaims := &auth.Claims{UserID: 1, Role: "superadmin", AccountID: 0}
	reqNoAcc := httptest.NewRequest(http.MethodPut, "/admin/integrations/platforms/tokens/hotmart", bytes.NewBufferString(`{"token_secret":"abc"}`))
	reqNoAcc = withChiPlatformParam(reqNoAcc.WithContext(auth.WithClaims(reqNoAcc.Context(), ownerClaims)), "hotmart")
	recNoAcc := httptest.NewRecorder()
	h.SetToken(recNoAcc, reqNoAcc)
	if recNoAcc.Code != http.StatusBadRequest {
		t.Fatalf("owner without account_id SetToken: got %d, want 400", recNoAcc.Code)
	}
}

func TestWebhookPixelQueryDoesNotCollideWithAuthToken(t *testing.T) {
	reqAuth := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/hotmart?acc=1&token=hottok_secreto", nil)
	if got := webhookPixelQuery(reqAuth); got != "" {
		t.Fatalf("webhookPixelQuery collided with auth ?token=: got %q, want empty", got)
	}

	reqPix := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/hotmart?acc=1&pixel=produto-vip", nil)
	if got := webhookPixelQuery(reqPix); got != "produto-vip" {
		t.Fatalf("webhookPixelQuery ?pixel=: got %q, want produto-vip", got)
	}

	reqPixTok := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/hotmart?acc=1&pixel_token=px_abc123", nil)
	if got := webhookPixelQuery(reqPixTok); got != "px_abc123" {
		t.Fatalf("webhookPixelQuery ?pixel_token=: got %q, want px_abc123", got)
	}
}

func TestStrictHMACAndNameCasing(t *testing.T) {
	body := []byte(`{"event":"order.paid"}`)
	secret := "yampi_hmac_secret_123"

	// Assinatura HMAC inválida sem payloadTokens não deve cair em query/header candidates
	invalidAttempt := webhookAuthAttempt{
		platform:   "yampi",
		rawBody:    body,
		hmacSHA256: "invalid_sig",
		candidates: []string{secret},
	}
	if matchWebhookAttemptAgainstSecret(secret, invalidAttempt) {
		t.Fatalf("expected invalid HMAC without payload fallback to be rejected")
	}

	// Quando há payloadToken explícito (ex: Eduzz OriginSecret com X-Signature diferente), permite validar pelo payloadToken
	eduzzAttempt := webhookAuthAttempt{
		platform:      "eduzz",
		rawBody:       body,
		hmacSHA256:    "different_webhook_sig",
		payloadTokens: []string{"eduzz_origin_secret"},
		candidates:    []string{"eduzz_origin_secret"},
	}
	if !matchWebhookAttemptAgainstSecret("eduzz_origin_secret", eduzzAttempt) {
		t.Fatalf("expected payload token fallback (Eduzz OriginSecret) to pass")
	}

	// Kiwify HMAC-SHA1 válido passa, inválido rejeita
	validSHA1 := util.GenerateHMACSHA1(body, secret)
	if !matchWebhookAttemptAgainstSecret(secret, webhookAuthAttempt{platform: "kiwify", rawBody: body, hmacSHA1: validSHA1}) {
		t.Fatalf("expected valid Kiwify HMAC-SHA1 to pass")
	}
	if matchWebhookAttemptAgainstSecret(secret, webhookAuthAttempt{platform: "kiwify", rawBody: body, hmacSHA1: "bad", candidates: []string{secret}}) {
		t.Fatalf("expected invalid Kiwify HMAC-SHA1 to be rejected")
	}

	// Preservação de maiúsculas/minúsculas em nomes e mascaramento de segredo
	var hp hotmartPayload
	hp.Data.Buyer.Name = "João Silva"
	first, last := hotmartName(hp)
	if first != "João" || last != "Silva" {
		t.Fatalf("hotmartName casing: got (%q, %q), want (João, Silva)", first, last)
	}
	if masked := domain.MaskPlatformSecret("hottok_987654"); masked != "••••••••7654" {
		t.Fatalf("MaskPlatformSecret: got %q, want ••••••••7654", masked)
	}
}
