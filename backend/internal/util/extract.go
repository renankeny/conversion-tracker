package util

import (
	"encoding/json"
	"strings"
)

// ExtractUserIDFromUTM lê o `userId` que o pixel.js insere no utm_source no
// formato "<resto>|userId<ID>|<resto>". Retorna "" se não encontrar.
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

// StripUserIDFromUTM remove marcadores "|userId<ID>|" ou "|uid<ID>|" de um
// valor UTM, preservando o nome limpo da fonte (ex: "facebook|userIdABC|" -> "facebook").
func StripUserIDFromUTM(s string) string {
	if !strings.Contains(s, "|") {
		return strings.TrimSpace(s)
	}
	parts := strings.Split(s, "|")
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "userId") || strings.HasPrefix(trimmed, "uid") {
			continue
		}
		kept = append(kept, trimmed)
	}
	return strings.Join(kept, "|")
}

// ExtractUserIDFromAny tenta extrair o userId de uma lista de fontes.
func ExtractUserIDFromAny(values ...string) string {
	for _, v := range values {
		if id := ExtractUserIDFromUTM(v); id != "" {
			return id
		}
	}
	return ""
}

var trackingFields = []string{
	"utm_source", "utm_term", "utm_content", "utm_medium", "utm_campaign",
	"UtmSource", "UTMSource", "UtmTerm", "UTMTerm",
	"UtmContent", "UTMContent", "UtmMedium", "UTMMedium",
	"UtmCampaign", "UTMCampaign",
	"utmsource", "utmterm", "utmcontent", "utmmedium", "utmcampaign",
	"sourceNullable", "termNullable", "contentNullable", "mediumNullable", "campaignNullable",
	"tracker_utm_source", "tracker_utm_campaign", "tracker_utm_medium", "tracker_utm_content",
	"checkout_source", "sckPaymentLink",
	"sck", "SCK", "src", "SRC", "source", "Source",
	"userId", "user_id", "userid", "UserID", "UserId",
	"xcod", "Xcod", "XCod",
	"s1", "s2", "ppc",
	"ref", "Ref",
	"notes", "metadata", "observacoes",
}

// ExtractUserIDDeep faz varredura recursiva (até profundidade 6) em um JSON bruto.
func ExtractUserIDDeep(rawJSON []byte) string {
	if len(rawJSON) == 0 {
		return ""
	}
	var root any
	if err := json.Unmarshal(rawJSON, &root); err != nil {
		return ""
	}
	return extractUserIDFromValue(root, 0)
}

func extractUserIDFromValue(v any, depth int) string {
	if depth > 6 || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return ExtractUserIDFromUTM(x)
	case map[string]any:
		if id := ExtractUserIDFromMap(x); id != "" {
			return id
		}
		for _, val := range x {
			if id := extractUserIDFromValue(val, depth+1); id != "" {
				return id
			}
		}
	case []any:
		for _, item := range x {
			if id := extractUserIDFromValue(item, depth+1); id != "" {
				return id
			}
		}
	}
	return ""
}

// ExtractUserIDFromMap procura o userId em qualquer campo conhecido de um map JSON.
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

// SplitFullName quebra um nome completo em (primeiro, último).
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

// PhoneVariants devolve variações úteis pra fallback de webhook.
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
