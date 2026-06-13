package util

import "testing"

func TestNormalizePhone(t *testing.T) {
	// NormalizePhone aplica a regra do sistema: prefixa "55" quando len entre
	// 10-11 (telefones BR sem DDI). Telefones já com DDI (>=12 dígitos) ficam
	// inalterados. Strings com poucos dígitos ficam inalteradas.
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"(11) 99999-8888", "5511999998888"}, // BR sem DDI -> prefixa 55
		{"11999998888", "5511999998888"},      // 11 dígitos BR
		{"5511999998888", "5511999998888"},    // já com DDI -> mantém
		// US "+1 (650) 555-1212" → "16505551212" (11 dígitos) → entra na regra
		// brasileira e ganha "55" prefixado. Limitação conhecida do legado;
		// pra produção multi-país conviria detectar pelo país.
		{"+1 (650) 555-1212", "5516505551212"},
		{"abc123", "123"}, // 3 dígitos < 10, sem prefixo
		{"123", "123"},
	}
	for _, c := range cases {
		got := NormalizePhone(c.in)
		if got != c.want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  John_Smith@Gmail.com ", "john_smith@gmail.com"},
		{"FOO@BAR.COM", "foo@bar.com"},
	}
	for _, c := range cases {
		if got := NormalizeEmail(c.in); got != c.want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"João", "joao"},
		{"São Paulo!", "saopaulo"},
		{"Mary-Anne", "maryanne"},
		{"Mariä Söhne", "mariasohne"},
	}
	for _, c := range cases {
		if got := NormalizeName(c.in); got != c.want {
			t.Errorf("NormalizeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeZip(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"01000-000", "01000000"},
		{"94035", "94035"},
		{"M1 1AE", "m11ae"},
		{"  m11ae ", "m11ae"},
	}
	for _, c := range cases {
		if got := NormalizeZip(c.in); got != c.want {
			t.Errorf("NormalizeZip(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeCountry(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"BR", "br"},
		{"us", "us"},
		{"USA", "us"},      // truncado em 2
		{"Brasil", "br"},   // só primeiras 2 letras
		{"  pt-br ", "pt"}, // remove não-alfa, trunca
	}
	for _, c := range cases {
		if got := NormalizeCountry(c.in); got != c.want {
			t.Errorf("NormalizeCountry(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeStateISO(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"SP", "sp"},
		{"São Paulo", "sa"},  // strip accents → "sao paulo" → primeiras 2 letras "sa"
		{"CA", "ca"},
		{"AZ", "az"},
		{"Rio de Janeiro", "ri"},
	}
	for _, c := range cases {
		if got := NormalizeStateISO(c.in); got != c.want {
			t.Errorf("NormalizeStateISO(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSHA256Hex(t *testing.T) {
	// Valor canônico da Meta CAPI docs: john_smith@gmail.com
	if got := SHA256Hex("john_smith@gmail.com"); got != "62a14e44f765419d10fea99367361a727c12365e2520f32218d505ed9aa0f62f" {
		t.Errorf("SHA256Hex hash mismatch: %s", got)
	}
	if got := SHA256Hex(""); got != "" {
		t.Errorf("SHA256Hex(\"\") should be \"\", got %q", got)
	}
}

func TestRedactSecrets(t *testing.T) {
	in := `meta returned 400: {"error":{"message":"Malformed access token EAANQ1dHS2dYBRiGZCdHOhnqJ0u4aPJxWqfBLvcaVgqDzScqxap"}}`
	got := RedactSecrets(in)
	if got == in {
		t.Errorf("RedactSecrets did not mask the token: %s", got)
	}
	if !contains(got, "EAAN") || !contains(got, "...") {
		t.Errorf("RedactSecrets result missing prefix/...: %s", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
