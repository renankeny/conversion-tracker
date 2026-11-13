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
