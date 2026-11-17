package util

import "strings"

// CleanTokenInput remove espaços e aspas acidentais ao redor de tokens colados manualmente.
func CleanTokenInput(raw string) string {
	return strings.Trim(strings.TrimSpace(raw), "\"'`")
}

// IsPlausibleMetaToken valida se uma string tem formato real de Access Token da Meta Graph/CAPI
// (sempre inicia com "EAA", sem espaços e com comprimento mínimo), rejeitando senhas preenchidas
// por autofill de navegadores, IDs numéricos de Pixel ou ciphertexts hexadecimais corrompidos.
func IsPlausibleMetaToken(raw string) bool {
	s := CleanTokenInput(raw)
	if len(s) < 50 || !strings.HasPrefix(s, "EAA") {
		return false
	}
	return !strings.ContainsAny(s, " \t\r\n")
}

// IsMetaAuthFailure identifica respostas 401 ou OAuthException 190 de token inválido/expirado na CAPI.
func IsMetaAuthFailure(statusCode int, body string) bool {
	if statusCode == 401 {
		return true
	}
	if statusCode != 400 && statusCode != 403 {
		return false
	}
	low := strings.ToLower(body)
	return strings.Contains(low, "invalid oauth access token") ||
		strings.Contains(low, `"code":190`) ||
		strings.Contains(low, `"code": 190`)
}
