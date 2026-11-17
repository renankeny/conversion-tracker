package util

import "testing"

func TestIsPlausibleMetaToken(t *testing.T) {
	valid := "EAABsbCS1iHgBO0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	if !IsPlausibleMetaToken(valid) {
		t.Fatalf("expected valid Meta token to pass")
	}
	if !IsPlausibleMetaToken(` "` + valid + `" `) {
		t.Fatalf("expected quoted valid Meta token to pass after CleanTokenInput")
	}

	invalids := []string{
		"",
		"SenhaDoPainel123!",
		"1057871509598219",
		"TEST12345",
		"EAAcurto",
		"eaa0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", // lowercase hex ciphertext
		"EAABsbCS1iHgBO0123456789 token com espaco 01234567890123456789",
	}
	for _, bad := range invalids {
		if IsPlausibleMetaToken(bad) {
			t.Fatalf("expected %q to be rejected as invalid Meta token", bad)
		}
	}
}

func TestIsMetaAuthFailure(t *testing.T) {
	if !IsMetaAuthFailure(401, `{"error":{"message":"Invalid OAuth access token - Cannot parse access token"}}`) {
		t.Fatal("expected 401 to be detected as auth failure")
	}
	if !IsMetaAuthFailure(400, `{"error":{"message":"Error validating access token: Session has expired","type":"OAuthException","code":190}}`) {
		t.Fatal("expected 400 code 190 to be detected as auth failure")
	}
	if IsMetaAuthFailure(400, `{"error":{"message":"Invalid parameter","code":100}}`) {
		t.Fatal("did not expect parameter error 100 to be treated as auth failure")
	}
}
