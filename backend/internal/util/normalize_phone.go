package util

import (
	"regexp"
	"strings"
)

var nonDigit = regexp.MustCompile(`\D+`)

var validBRDDDs = map[string]bool{
	"11": true, "12": true, "13": true, "14": true, "15": true, "16": true, "17": true, "18": true, "19": true,
	"21": true, "22": true, "24": true, "27": true, "28": true,
	"31": true, "32": true, "33": true, "34": true, "35": true, "37": true, "38": true,
	"41": true, "42": true, "43": true, "44": true, "45": true, "46": true, "47": true, "48": true, "49": true,
	"51": true, "53": true, "54": true, "55": true,
	"61": true, "62": true, "63": true, "64": true, "65": true, "66": true, "67": true, "68": true, "69": true,
	"71": true, "73": true, "74": true, "75": true, "77": true, "79": true,
	"81": true, "82": true, "83": true, "84": true, "85": true, "86": true, "87": true, "88": true, "89": true,
	"91": true, "92": true, "93": true, "94": true, "95": true, "96": true, "97": true, "98": true, "99": true,
}

func isAllSameByte(s string) bool {
	if len(s) < 2 {
		return false
	}
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}

func stripBRTrunkZeroAfter55(s string) string {
	if strings.HasPrefix(s, "550") && (len(s) == 13 || len(s) == 14) && validBRDDDs[s[3:5]] {
		return "55" + s[3:]
	}
	return s
}

// NormalizePhone normaliza telefone sem contexto explícito de país (assume BR para DDDs brasileiros).
func NormalizePhone(s string) string {
	return NormalizePhoneWithCountry(s, "")
}

// NormalizePhoneWithCountry remove caracteres não numéricos, descarta telefones dummy (ex: 00000000000),
// remove zero de tronco após DDI 55 (+55 011... -> 5511...) e só prefixa 55 quando o país for vazio ou BR.
// Se já receber um hash SHA-256 hex de 64 caracteres, preserva em lowercase sem destruir letras hex.
func NormalizePhoneWithCountry(s, country string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	if IsSHA256Hex(trimmed) {
		return strings.ToLower(trimmed)
	}

	hasExplicitPlus := strings.HasPrefix(trimmed, "+")
	digits := nonDigit.ReplaceAllString(trimmed, "")
	if digits == "" || isAllSameByte(digits) {
		return ""
	}

	if hasExplicitPlus {
		return stripBRTrunkZeroAfter55(digits)
	}

	if strings.HasPrefix(digits, "00") && len(digits) > 4 {
		rest := strings.TrimPrefix(digits, "00")
		if rest == "" || isAllSameByte(rest) {
			return ""
		}
		return stripBRTrunkZeroAfter55(rest)
	}

	digits = stripBRTrunkZeroAfter55(digits)
	normCountry := NormalizeCountry(country)

	if normCountry == "us" || normCountry == "ca" {
		if len(digits) == 10 {
			return "1" + digits
		}
		return digits
	}

	if normCountry != "" && normCountry != "br" {
		return digits
	}

	if len(digits) >= 11 && len(digits) <= 12 && strings.HasPrefix(digits, "0") {
		candidate := strings.TrimPrefix(digits, "0")
		if len(candidate) >= 2 && validBRDDDs[candidate[:2]] {
			digits = candidate
		}
	}
	if (len(digits) == 10 || len(digits) == 11) && validBRDDDs[digits[:2]] {
		digits = "55" + digits
	}
	return digits
}

// NormalizePhoneE164 devolve o telefone no formato E.164 com "+" inicial
// (ex: "+5511999999999"), exigido por TikTok Events API e Google Ads Data Manager.
func NormalizePhoneE164(s string) string {
	digits := NormalizePhone(s)
	if digits == "" || IsSHA256Hex(digits) {
		return digits
	}
	return "+" + digits
}
