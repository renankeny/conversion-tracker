package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
)

func okHandler(onClaims func(*auth.Claims)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if onClaims != nil {
			c, _ := auth.ClaimsFrom(r.Context())
			onClaims(c)
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireHumanAuthBlocksAPIKey(t *testing.T) {
	h := RequireHumanAuth(okHandler(nil))

	// 1. Sem claims -> 403
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing claims: got %d, want 403", rec.Code)
	}

	// 2. API Key (`trk_*`) -> 403
	apiKeyClaims := &auth.Claims{Role: "member", TokenType: auth.TokenTypeAPIKey, AccountID: 5}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), apiKeyClaims))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("api key claims: got %d, want 403", rec.Code)
	}

	// 3. Humano autenticado -> 200
	humanClaims := &auth.Claims{UserID: 10, Role: "admin", TokenType: auth.TokenTypeHuman, AccountID: 5}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), humanClaims))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("human claims: got %d, want 200", rec.Code)
	}
}

func TestAuthMiddlewareRevokesJTIAndUserSessions(t *testing.T) {
	jwtSvc := auth.NewJWT("middleware-test-secret-key-32-chars", 24)
	sessions := cache.NewSessionStore(nil)
	mw := AuthMiddlewareWithSessions(jwtSvc, nil, sessions)(okHandler(nil))

	tok, err := jwtSvc.Issue(77, "admin@trackfind.test", "admin", 3, nil, nil)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	parsed, err := jwtSvc.Parse(tok)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	// Antes do logout -> 200
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid token: got %d, want 200", rec.Code)
	}

	// Logout explícito (RevokeToken por JTI) -> 401
	sessions.RevokeToken(req.Context(), parsed.ID, 10*time.Minute)
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("revoked JTI: got %d, want 401", rec.Code)
	}

	// Novo token revogado por corte de usuário (troca de senha/exclusão)
	tok2, err := jwtSvc.Issue(77, "admin@trackfind.test", "admin", 3, nil, nil)
	if err != nil {
		t.Fatalf("issue tok2: %v", err)
	}
	sessions.RevokeUserSessions(req.Context(), 77)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
	req2.Header.Set("Authorization", "Bearer "+tok2)
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("user-level session cutoff: got %d, want 401", rec2.Code)
	}
}

func TestTenantMiddlewareEnforcesAccountIsolation(t *testing.T) {
	var resolvedAcc int64
	mw := TenantMiddlewareWithSessions(nil, nil)(okHandler(func(c *auth.Claims) {
		if c != nil {
			resolvedAcc = c.AccountID
		}
	}))

	// 1. Header inválido -> 403
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/pixels", nil)
	req.Header.Set("X-Account-ID", "abc")
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{UserID: 1, Role: "owner", TokenType: auth.TokenTypeHuman, AccountID: 1}))
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("invalid X-Account-ID: got %d, want 403", rec.Code)
	}

	// 2. API Key tentando acessar outra empresa -> 403
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/pixels", nil)
	req.Header.Set("X-Account-ID", "99")
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{Role: "member", TokenType: auth.TokenTypeAPIKey, AccountID: 5}))
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("api key cross-tenant: got %d, want 403", rec.Code)
	}

	// 3. Employee tentando alternar de empresa -> 403
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/pixels", nil)
	req.Header.Set("X-Account-ID", "99")
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{UserID: 8, Role: "employee", TokenType: auth.TokenTypeHuman, AccountID: 5}))
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("employee cross-tenant: got %d, want 403", rec.Code)
	}

	// 4. Owner alternando para empresa 42 -> 200 e AccountID atualizado
	resolvedAcc = 0
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/pixels", nil)
	req.Header.Set("X-Account-ID", "42")
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{UserID: 1, Role: "owner", TokenType: auth.TokenTypeHuman, AccountID: 1}))
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || resolvedAcc != 42 {
		t.Fatalf("owner switch: code=%d resolvedAcc=%d, want 200 and 42", rec.Code, resolvedAcc)
	}
}

