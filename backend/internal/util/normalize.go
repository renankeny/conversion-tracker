package util

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var nonDigit = regexp.MustCompile(`\D+`)
var nonAlpha = regexp.MustCompile(`[^a-z]`)

// NormalizePhone remove caracteres não numéricos e prefixa 55 quando claramente
// for número brasileiro sem DDI.
func NormalizePhone(s string) string {
	s = nonDigit.ReplaceAllString(s, "")
	if len(s) >= 10 && len(s) <= 11 {
		s = "55" + s
	}
	return s
}

// NormalizeName remove acentos e caracteres não-alfabéticos, lowercase.
// Usado pra Meta CAPI advanced matching.
func NormalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = stripAccents(s)
	s = nonAlpha.ReplaceAllString(s, "")
	return s
}

// NormalizeCity igual a NormalizeName.
func NormalizeCity(s string) string { return NormalizeName(s) }

// NormalizeEmail lowercase + trim.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// SHA256Hex devolve o hash SHA256 hex (lowercase) de s. Se s vazio, devolve "".
func SHA256Hex(s string) string {
	if s == "" {
		return ""
	}
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

var accentMap = strings.NewReplacer(
	"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

func stripAccents(s string) string { return accentMap.Replace(s) }


// NormalizeZip aplica o formato exigido pela Meta CAPI:
// lowercase, remove espaços e traços, mantém alfanumérico.
// Pra códigos US, mantém só os 5 primeiros dígitos.
//
// Ref: https://developers.facebook.com/docs/marketing-api/conversions-api/parameters/customer-information-parameters
func NormalizeZip(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		}
	}
	return string(out)
}

// NormalizeCountry aplica o formato ISO 3166-1 alpha-2 lowercase
// exigido pela Meta CAPI (`br`, `us`, `pt`, ...).
// Se a entrada tiver mais de 2 letras (ex: "Brasil"), tenta extrair
// só o que for alfa e devolve em lowercase truncado em 2.
func NormalizeCountry(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			out = append(out, r)
		}
	}
	if len(out) > 2 {
		out = out[:2]
	}
	return string(out)
}

// NormalizeStateISO devolve o código ANSI lowercase de 2 caracteres
// (ex: "ca", "az", "sp", "rj"). Aceita entradas como "CA", "California",
// "São Paulo" etc. Aplica strip de acentos antes de filtrar pra que
// "São Paulo" → "sa" e não "so".
func NormalizeStateISO(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = stripAccents(s)
	if s == "" {
		return ""
	}
	out := make([]rune, 0, 2)
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			out = append(out, r)
			if len(out) == 2 {
				break
			}
		}
	}
	return string(out)
}
