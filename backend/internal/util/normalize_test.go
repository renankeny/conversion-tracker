package util

import (
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"11999998888", "5511999998888"},
		{"011999998888", "5511999998888"},
		{"(11) 99999-8888", "5511999998888"},
		{"011 99999-8888", "5511999998888"},
		{"+55 11 99999-8888", "5511999998888"},
		{"5511999998888", "5511999998888"},
		{"1133334444", "551133334444"},
		{"01133334444", "551133334444"},
		{"2015550123", "2015550123"},
		{"0014155552671", "14155552671"},
		{"12345", "12345"},
	}

	for _, tc := range tests {
		got := NormalizePhone(tc.input)
		if got != tc.want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeStateISO(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SP", "sp"},
		{"sp", "sp"},
		{"São Paulo", "sp"},
		{"sao paulo", "sp"},
		{"Rio de Janeiro", "rj"},
		{"Minas Gerais", "mg"},
		{"Rio Grande do Sul", "rs"},
		{"Rio Grande do Norte", "rn"},
		{"Bahia", "ba"},
		{"Paraná", "pr"},
		{"Santa Catarina", "sc"},
		{"Goiás", "go"},
		{"Distrito Federal", "df"},
		{"Ceará", "ce"},
		{"Amazonas", "am"},
		{"New York", "ny"},
		{"California", "ca"},
		{"Texas", "tx"},
		{"Florida", "fl"},
	}

	for _, tc := range tests {
		got := NormalizeStateISO(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeStateISO(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeCountry(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BR", "br"},
		{"br", "br"},
		{"Brasil", "br"},
		{"Brazil", "br"},
		{"United States", "us"},
		{"Estados Unidos", "us"},
		{"USA", "us"},
		{"Portugal", "pt"},
		{"Espanha", "es"},
		{"Spain", "es"},
		{"Alemanha", "de"},
		{"Germany", "de"},
		{"Argentina", "ar"},
		{"Paraguai", "py"},
		{"Reino Unido", "gb"},
		{"Pais Desconhecido", ""},
	}

	for _, tc := range tests {
		got := NormalizeCountry(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeCountry(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizePhoneWithCountry_DummyAndInternational(t *testing.T) {
	if got := NormalizePhone("(00) 00000-0000"); got != "" {
		t.Fatalf("dummy zeros must return empty, got %q", got)
	}
	if got := NormalizePhone("99999999999"); got != "" {
		t.Fatalf("repeated digits must return empty, got %q", got)
	}
	if got := NormalizePhone("+55 011 99999-8888"); got != "5511999998888" {
		t.Fatalf("expected +55 trunk zero stripped to 5511999998888, got %q", got)
	}
	if got := NormalizePhoneWithCountry("(212) 555-0199", "US"); got != "12125550199" {
		t.Fatalf("US 10-digit phone got %q, want 12125550199", got)
	}
	if got := NormalizePhoneWithCountry("912345678", "PT"); got != "912345678" {
		t.Fatalf("PT phone must not receive BR 55 prefix, got %q", got)
	}
	preHashed := SHA256Hex("5511999998888")
	if got := SHA256Hex(preHashed); got != preHashed {
		t.Fatalf("SHA256Hex double-hashed already hashed value: %q != %q", got, preHashed)
	}
}

func TestNormalizeZipAndUnicodeNameAndAuthCheck(t *testing.T) {
	if got := NormalizeZipWithCountry("90210-1234", "US"); got != "90210" {
		t.Fatalf("NormalizeZipWithCountry US = %q, want 90210", got)
	}
	if got := NormalizeZipWithCountry("01310-100", "BR"); got != "01310100" {
		t.Fatalf("NormalizeZipWithCountry BR = %q, want 01310100", got)
	}
	if got := NormalizeName("  João   Conceição "); got != "joaoconceicao" {
		t.Fatalf("NormalizeName = %q, want joaoconceicao", got)
	}
	if got := NormalizeName("  Иван-123 "); got != "иван" {
		t.Fatalf("NormalizeName unicode = %q, want иван", got)
	}
	for _, body := range []string{
		`{"error":{"code":190,"message":"Invalid OAuth access token"}}`,
		`{"error":{"code":200,"message":"(#200) Permissions error"}}`,
		`{"error":{"code":10,"message":"(#10) Application does not have permission"}}`,
		`{"error":{"code":100,"error_subcode":33,"message":"Unsupported post request"}}`,
	} {
		if !IsMetaAuthFailure(400, body) {
			t.Fatalf("expected IsMetaAuthFailure=true for %s", body)
		}
	}
}
