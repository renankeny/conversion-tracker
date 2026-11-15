package util

import (
	"regexp"
	"strings"
)

// accessTokenPattern detecta access tokens da Meta no formato típico:
// EAA seguido de caracteres alfanuméricos (20+ chars). Aparece em URLs
// de erro e mensagens da Graph API.
var accessTokenPattern = regexp.MustCompile(`EAA[A-Za-z0-9]{20,}`)

// sensitiveQueryParamPattern detecta segredos em query strings de URLs
// (como api_secret do GA4, access_token, client_secret, refresh_token, secret, token).
var sensitiveQueryParamPattern = regexp.MustCompile(`(?i)(api_secret|access_token|client_secret|refresh_token|secret|token)=([^&\s"']+)`)

// bearerTokenPattern detecta cabeçalhos ou logs com Authorization Bearer.
var bearerTokenPattern = regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9._\-]{12,}`)

// RedactSecrets substitui valores sensíveis (tokens Meta, segredos de query string
// como api_secret do GA4 e Bearer tokens) por placeholders mascarados.
// Use antes de gravar respostas ou erros em logs e no banco (ex: sent_events.error).
func RedactSecrets(s string) string {
	if s == "" {
		return s
	}
	out := bearerTokenPattern.ReplaceAllString(s, "${1}***")
	out = accessTokenPattern.ReplaceAllStringFunc(out, func(match string) string {
		if len(match) <= 8 {
			return "***"
		}
		return match[:4] + "..." + match[len(match)-4:]
	})
	out = sensitiveQueryParamPattern.ReplaceAllStringFunc(out, func(match string) string {
		eq := strings.IndexByte(match, '=')
		if eq < 0 {
			return "***"
		}
		key := match[:eq]
		val := match[eq+1:]
		if len(val) == 11 && strings.HasPrefix(val, "EAA") && val[4:7] == "..." {
			return key + "=" + val
		}
		return key + "=***"
	})
	return out
}
