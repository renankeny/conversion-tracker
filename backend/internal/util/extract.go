package util

import "strings"

// ExtractUserIDFromUTM lê o `userId` que o pixel.js insere no utm_source no
// formato "<resto>|userId<ID>|<resto>". Retorna "" se não encontrar.
//
// O legado também aceita "uid<ID>" como fallback; mantemos o mesmo comportamento.
func ExtractUserIDFromUTM(s string) string {
	if s == "" {
		return ""
	}
	for _, part := range strings.Split(s, "|") {
		part = strings.TrimSpace(part)
		if v, ok := strings.CutPrefix(part, "userId"); ok && v != "" {
			return v
		}
		if v, ok := strings.CutPrefix(part, "uid"); ok && v != "" {
			return v
		}
	}
	return ""
}

// FirstNonEmpty devolve o primeiro valor não vazio.
func FirstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

// SplitFullName quebra um nome completo em (primeiro, último). Se houver só
// uma palavra, retorna (palavra, ""). Se vazio, ("", "").
func SplitFullName(full string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(full))
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		return parts[0], parts[len(parts)-1]
	}
}

// PhoneVariants devolve variações úteis pra fallback de webhook:
// se vier sem 55, adiciona "55"+s; se vier com 55 e tiver mais de 11 chars,
// adiciona a versão sem o 55. Sempre inclui o telefone original.
func PhoneVariants(phone string) []string {
	phone = NormalizePhone(phone)
	if phone == "" {
		return nil
	}
	out := []string{phone}
	if strings.HasPrefix(phone, "55") && len(phone) > 11 {
		out = append(out, strings.TrimPrefix(phone, "55"))
	}
	if !strings.HasPrefix(phone, "55") && len(phone) <= 11 {
		out = append(out, "55"+phone)
	}
	return out
}
