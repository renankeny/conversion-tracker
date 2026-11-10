package util

import "regexp"

// accessTokenPattern detecta access tokens da Meta no formato típico:
// EAA seguido de caracteres alfanuméricos (40+ chars). Aparece em URLs
// de erro e mensagens da Graph API.
var accessTokenPattern = regexp.MustCompile(`EAA[A-Za-z0-9]{20,}`)

// RedactSecrets substitui valores sensíveis (access tokens da Meta) por
// um placeholder mascarado. Use antes de gravar respostas/erros em logs
// ou no banco (ex: sent_events.error).
//
// Mantém os primeiros 4 e últimos 4 caracteres pra ainda ser possível
// correlacionar o token com a origem (ex: identificar qual pixel falhou).
func RedactSecrets(s string) string {
	if s == "" {
		return s
	}
	return accessTokenPattern.ReplaceAllStringFunc(s, func(match string) string {
		if len(match) <= 8 {
			return "***"
		}
		return match[:4] + "..." + match[len(match)-4:]
	})
}
