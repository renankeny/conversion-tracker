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

// ExtractUserIDFromAny tenta extrair o userId de uma lista de fontes
// (utm_source, utm_term, utm_content, sck, src, xcod, s1, s2 etc.).
// Retorna o primeiro match encontrado, ou "" se nenhum bater.
//
// Útil quando o gateway permite ao cliente injetar o tracking em campos
// arbitrários — varrer várias fontes evita perder a atribuição quando o
// usuário troca o nome do parâmetro no checkout.
func ExtractUserIDFromAny(values ...string) string {
	for _, v := range values {
		if id := ExtractUserIDFromUTM(v); id != "" {
			return id
		}
	}
	return ""
}

// trackingFields é a lista de chaves comuns onde o userId pode aparecer.
// Inclui variações de capitalização porque alguns gateways (Hotmart v1
// legacy, etc) usam PascalCase no JSON.
var trackingFields = []string{
	"utm_source", "utm_term", "utm_content", "utm_medium", "utm_campaign",
	"UtmSource", "UTMSource", "UtmTerm", "UTMTerm",
	"UtmContent", "UTMContent", "UtmMedium", "UTMMedium",
	"UtmCampaign", "UTMCampaign",
	"sck", "SCK", "src", "SRC", "source", "Source",
	"userId", "user_id", "userid", "UserID", "UserId",
	"xcod", "Xcod", "XCod",
	"s1", "s2",
	"ref", "Ref",
	"notes", "metadata", "observacoes",
}

// ExtractUserIDFromMap procura o userId em qualquer campo conhecido de um
// objeto JSON decodificado como map[string]any. Aceita strings em qualquer
// nível de aninhamento simples (não recursivo — usar paths fixos pra subpath).
func ExtractUserIDFromMap(m map[string]any) string {
	if m == nil {
		return ""
	}
	for _, k := range trackingFields {
		v, ok := m[k]
		if !ok {
			continue
		}
		if s, ok := v.(string); ok {
			if id := ExtractUserIDFromUTM(s); id != "" {
				return id
			}
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
