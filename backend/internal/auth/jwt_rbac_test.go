package auth

import (
	"testing"
	"time"
)

func TestClaimsIsOwnerAndAPIKey(t *testing.T) {
	var nilClaims *Claims
	if nilClaims.IsOwner() || nilClaims.IsAPIKey() || nilClaims.HasAccountAccess(1) {
		t.Fatalf("nil claims must deny all privileges")
	}

	cases := []struct {
		name      string
		claims    Claims
		wantOwner bool
		wantAPI   bool
	}{
		{
			name:      "empty role must NEVER be owner",
			claims:    Claims{UserID: 1, Role: "", TokenType: TokenTypeHuman},
			wantOwner: false,
			wantAPI:   false,
		},
		{
			name:      "employee human",
			claims:    Claims{UserID: 2, Role: "employee", TokenType: TokenTypeHuman, AccountID: 7},
			wantOwner: false,
			wantAPI:   false,
		},
		{
			name:      "admin human",
			claims:    Claims{UserID: 3, Role: "admin", TokenType: TokenTypeHuman, AccountID: 7},
			wantOwner: false,
			wantAPI:   false,
		},
		{
			name:      "owner human",
			claims:    Claims{UserID: 4, Role: "owner", TokenType: TokenTypeHuman, AccountID: 1},
			wantOwner: true,
			wantAPI:   false,
		},
		{
			name:      "superadmin human",
			claims:    Claims{UserID: 5, Role: "superadmin", TokenType: TokenTypeHuman, AccountID: 1},
			wantOwner: true,
			wantAPI:   false,
		},
		{
			name:      "api key member",
			claims:    Claims{UserID: 0, Role: "member", TokenType: TokenTypeAPIKey, AccountID: 7},
			wantOwner: false,
			wantAPI:   true,
		},
		{
			name:      "api key spoofing owner role is still blocked",
			claims:    Claims{UserID: 0, Role: "owner", TokenType: TokenTypeAPIKey, AccountID: 7},
			wantOwner: false,
			wantAPI:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.claims.IsOwner(); got != tc.wantOwner {
				t.Fatalf("IsOwner() = %v, want %v", got, tc.wantOwner)
			}
			if got := tc.claims.IsAPIKey(); got != tc.wantAPI {
				t.Fatalf("IsAPIKey() = %v, want %v", got, tc.wantAPI)
			}
		})
	}
}

func TestJWTIssueAndParse(t *testing.T) {
	j := NewJWT("unit-test-secret-key-32-bytes-minimum", 24)

	tok, err := j.IssueWithDuration(99, "dev@trackfind.test", "", 12, nil, nil, 1*time.Hour)
	if err != nil {
		t.Fatalf("IssueWithDuration error: %v", err)
	}

	claims, err := j.Parse(tok)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if claims.Role != "employee" {
		t.Fatalf("expected empty role to default to 'employee', got %q", claims.Role)
	}
	if claims.IsOwner() {
		t.Fatalf("default role must not be owner")
	}
	if claims.TokenType != TokenTypeHuman {
		t.Fatalf("expected TokenType %q, got %q", TokenTypeHuman, claims.TokenType)
	}
	if claims.ID == "" {
		t.Fatalf("expected non-empty JTI in RegisteredClaims.ID")
	}
	if !claims.HasAccountAccess(12) || claims.HasAccountAccess(13) {
		t.Fatalf("unexpected HasAccountAccess behavior for account 12")
	}

	other := NewJWT("different-secret-key-must-fail-parse", 24)
	if _, err := other.Parse(tok); err == nil {
		t.Fatalf("expected Parse with wrong secret to fail")
	}
}

