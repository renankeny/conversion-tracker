package handlers

import (
	"errors"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func indexTokensByPixelID(pixelIDs, tokens []string) map[string]string {
	out := make(map[string]string, len(pixelIDs))
	for i, pid := range pixelIDs {
		if i < len(tokens) {
			out[strings.TrimSpace(pid)] = tokens[i]
		}
	}
	return out
}

func padOrTrim(s []string, n int) []string {
	out := make([]string, n)
	for i := 0; i < n && i < len(s); i++ {
		out[i] = util.CleanTokenInput(s[i])
	}
	return out
}

func isMaskedOrEmpty(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || s == "****" || strings.Contains(s, "...")
}

func mergeTokens(pixelIDs, incomingTokens []string, oldByPixelID map[string]string) []string {
	out := make([]string, len(pixelIDs))
	for i, pid := range pixelIDs {
		pid = strings.TrimSpace(pid)
		tok := ""
		if i < len(incomingTokens) {
			tok = util.CleanTokenInput(incomingTokens[i])
		}
		if tok == clearTokenSentinel {
			out[i] = ""
			continue
		}
		if !isMaskedOrEmpty(tok) {
			out[i] = tok
			continue
		}
		if prev, ok := oldByPixelID[pid]; ok && prev != "" {
			out[i] = prev
		}
	}
	return out
}

// sanitizeMetaTokens protege contra autofill de senha do navegador no campo Access Token:
// quando há Conta Meta Ads OAuth vinculada, tokens inválidos (ex: senha do painel) são
// descartados silenciosamente para priorizar o OAuth; sem OAuth, exige formato EAA válido.
func sanitizeMetaTokens(tokens []string, hasMetaOAuth bool) ([]string, error) {
	out := make([]string, len(tokens))
	for i, raw := range tokens {
		tok := util.CleanTokenInput(raw)
		if tok == "" {
			continue
		}
		if !util.IsPlausibleMetaToken(tok) {
			if hasMetaOAuth {
				out[i] = ""
				continue
			}
			return nil, errors.New("o Access Token da Meta informado é inválido (tokens da Meta começam com EAA...). Verifique se copiou o token completo do Gerenciador de Eventos ou conecte via OAuth")
		}
		out[i] = tok
	}
	return out, nil
}
