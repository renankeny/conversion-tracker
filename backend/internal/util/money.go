package util

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// moneyRe extrai dígitos, vírgula e ponto (descarta R$, espaços, etc).
var moneyRe = regexp.MustCompile(`[^\d,\.\-]`)

// ParseMoneyAny converte valores de webhooks para float64 em moeda real.
// Aceita números (float64, int) ou strings com formatos brasileiros e
// internacionais:
//
//	123.45         → 123.45
//	"123,45"       → 123.45
//	"1.234,56"     → 1234.56
//	"R$ 99,90"     → 99.90
//	"$ 1,234.56"   → 1234.56  (formato US)
//	nil ou ""      → 0
//
// Heurística de separador decimal:
//   - Se há vírgula E ponto, o último que aparece é o decimal.
//   - Se só há vírgula, ela é o decimal (formato BR).
//   - Se só há ponto, ele é o decimal (formato US).
func ParseMoneyAny(v any) float64 {
	switch x := v.(type) {
	case nil:
		return 0
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return f
		}
		return parseMoneyString(x.String())
	case string:
		return parseMoneyString(x)
	}
	return 0
}

// ParseMoneyCents converte valores expressos em centavos (int, float inteiro ou string de dígitos)
// para moeda real (dividindo por 100). Se receber string já formatada com separador decimal
// ("97.00" ou "R$ 97,00"), interpreta diretamente como moeda real sem dividir por 100.
func ParseMoneyCents(v any) float64 {
	switch x := v.(type) {
	case nil:
		return 0
	case int:
		return float64(x) / 100.0
	case int64:
		return float64(x) / 100.0
	case float64:
		if x != float64(int64(x)) {
			return x
		}
		return x / 100.0
	case json.Number:
		s := strings.TrimSpace(x.String())
		if strings.ContainsAny(s, ".,") {
			return parseMoneyString(s)
		}
		if i, err := x.Int64(); err == nil {
			return float64(i) / 100.0
		}
		return parseMoneyString(s)
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0
		}
		if strings.ContainsAny(s, ".,") {
			return parseMoneyString(s)
		}
		cleaned := moneyRe.ReplaceAllString(s, "")
		if i, err := strconv.ParseInt(cleaned, 10, 64); err == nil {
			return float64(i) / 100.0
		}
		return 0
	}
	return 0
}

func parseMoneyString(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	cleaned := moneyRe.ReplaceAllString(s, "")
	if cleaned == "" {
		return 0
	}

	hasComma := strings.Contains(cleaned, ",")
	hasDot := strings.Contains(cleaned, ".")

	switch {
	case hasComma && hasDot:
		// Decimal é o separador que aparece por último.
		if strings.LastIndex(cleaned, ",") > strings.LastIndex(cleaned, ".") {
			cleaned = strings.ReplaceAll(cleaned, ".", "")
			cleaned = strings.Replace(cleaned, ",", ".", 1)
		} else {
			cleaned = strings.ReplaceAll(cleaned, ",", "")
		}
	case hasComma:
		cleaned = strings.Replace(cleaned, ",", ".", 1)
	}

	f, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return f
}

// FormatFloat converte float64 pra string sem zeros à direita ou ponto sobrando.
// Útil pra serializar IDs numéricos vindos como float em JSON dinâmico.
//
//	123.0   → "123"
//	123.456 → "123.456"
func FormatFloat(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
