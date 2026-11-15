package util

import (
	"strings"
	"testing"
)

func TestRedactSecrets_GA4APISecret(t *testing.T) {
	raw := `Post "https://www.google-analytics.com/mp/collect?measurement_id=G-ABC12345&api_secret=MySuperSecretKey99": context deadline exceeded`
	got := RedactSecrets(raw)

	if strings.Contains(got, "MySuperSecretKey99") {
		t.Fatalf("expected api_secret to be redacted, got: %s", got)
	}
	want := `Post "https://www.google-analytics.com/mp/collect?measurement_id=G-ABC12345&api_secret=***": context deadline exceeded`
	if got != want {
		t.Fatalf("unexpected redacted output:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestRedactSecrets_QueryParams(t *testing.T) {
	raw := `https://example.com/oauth?client_secret=cs_123&refresh_token=rt_456&token=tk_789&access_token=ya29.secret_tok`
	got := RedactSecrets(raw)

	for _, secret := range []string{"cs_123", "rt_456", "tk_789", "ya29.secret_tok"} {
		if strings.Contains(got, secret) {
			t.Fatalf("expected %q to be redacted, got: %s", secret, got)
		}
	}
	want := `https://example.com/oauth?client_secret=***&refresh_token=***&token=***&access_token=***`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRedactSecrets_MetaEAAToken(t *testing.T) {
	tok := "EAABsbCS1iHgBO1234567890abcdefghijWXYZ"
	raw := `Invalid OAuth access token: ` + tok
	got := RedactSecrets(raw)

	if strings.Contains(got, "1234567890abcdefghij") {
		t.Fatalf("expected Meta token middle to be masked, got: %s", got)
	}
	want := `Invalid OAuth access token: EAAB...WXYZ`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRedactSecrets_BearerToken(t *testing.T) {
	raw := `request failed with header Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload_123-abc`
	got := RedactSecrets(raw)

	if strings.Contains(got, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") {
		t.Fatalf("expected Bearer token to be masked, got: %s", got)
	}
	want := `request failed with header Authorization: Bearer ***`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
