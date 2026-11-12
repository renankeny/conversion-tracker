package util

import (
	"reflect"
	"testing"
)

func TestExtractUserIDFromUTM(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"fb|userIdABC123|seo", "ABC123"},
		{"userIdXYZ", "XYZ"},  // só userId
		{"|userId|", ""},      // userId sem id -> ""
		{"|uidQQQ|", "QQQ"},   // alias uid
		{"sem_marcador", ""},  // não tem userId/uid
	}
	for _, c := range cases {
		if got := ExtractUserIDFromUTM(c.in); got != c.want {
			t.Errorf("ExtractUserIDFromUTM(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSplitFullName(t *testing.T) {
	cases := []struct {
		in    string
		first string
		last  string
	}{
		{"", "", ""},
		{"Fulano", "Fulano", ""},
		{"Fulano da Silva", "Fulano", "Silva"},
		{"  Maria   Santos  ", "Maria", "Santos"},
		{"Joao Pedro de Souza", "Joao", "Souza"}, // último token é o sobrenome
	}
	for _, c := range cases {
		f, l := SplitFullName(c.in)
		if f != c.first || l != c.last {
			t.Errorf("SplitFullName(%q) = (%q, %q), want (%q, %q)", c.in, f, l, c.first, c.last)
		}
	}
}

func TestPhoneVariants(t *testing.T) {
	// PhoneVariants normaliza primeiro (NormalizePhone), depois agrega
	// variantes com/sem prefixo "55" pra fallback de webhook.
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		// "11999998888" → NormalizePhone prefixa 55 → "5511999998888".
		// HasPrefix("55") + len > 11 → adiciona variante sem 55.
		{"11999998888", []string{"5511999998888", "11999998888"}},
		{"5511999998888", []string{"5511999998888", "11999998888"}},
		// 9 dígitos: NormalizePhone NÃO prefixa (regra é 10-11).
		// Sai como "999998888". Aí PhoneVariants vê !HasPrefix(55) && len<=11
		// e adiciona "55"+s.
		{"999998888", []string{"999998888", "55999998888"}},
	}
	for _, c := range cases {
		got := PhoneVariants(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("PhoneVariants(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := FirstNonEmpty("", "", "x"); got != "x" {
		t.Errorf("FirstNonEmpty: got %q", got)
	}
	if got := FirstNonEmpty(); got != "" {
		t.Errorf("FirstNonEmpty(): got %q", got)
	}
}

func TestExtractEmail(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"", ""},
		{"Olá, meu email é cliente@gmail.com", "cliente@gmail.com"},
		{"Pode mandar para CONTATO@EMPRESA.COM.BR!", "contato@empresa.com.br"},
		{"meu email: (teste.lead+promo@dominio.io).", "teste.lead+promo@dominio.io"},
		{"email: 'financeiro@empresa.com' obrigado", "financeiro@empresa.com"},
		{"não tem email aqui", ""},
		{"email quebrado @dominio sem arroba", ""},
	}
	for _, c := range cases {
		got := ExtractEmail(c.text)
		if got != c.want {
			t.Errorf("ExtractEmail(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

