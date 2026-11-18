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

// IsMetaAuthFailure identifica respostas 401 ou erros OAuth/permissão (190, 10, 200, 100/33)
// na Graph API para acionar automaticamente o fallback de credencial OAuth.
func IsMetaAuthFailure(statusCode int, body string) bool {
	if statusCode == 401 {
		return true
	}
	if statusCode != 400 && statusCode != 403 {
		return false
	}
	low := strings.ToLower(body)
	if strings.Contains(low, "invalid oauth access token") ||
		strings.Contains(low, `"code":190`) || strings.Contains(low, `"code": 190`) ||
		strings.Contains(low, `"code":10,`) || strings.Contains(low, `"code": 10,`) ||
		strings.Contains(low, `"code":200`) || strings.Contains(low, `"code": 200`) {
		return true
	}
	hasCode100 := strings.Contains(low, `"code":100`) || strings.Contains(low, `"code": 100`)
	hasSubcode33 := strings.Contains(low, `"error_subcode":33`) || strings.Contains(low, `"error_subcode": 33`)
	return hasCode100 && hasSubcode33
}
