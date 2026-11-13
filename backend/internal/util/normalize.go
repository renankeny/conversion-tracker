package util

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var nonDigit = regexp.MustCompile(`\D+`)
var nonAlpha = regexp.MustCompile(`[^a-z]`)

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

// NormalizePhone remove caracteres não numéricos, remove zero à esquerda no DDD brasileiro
// se presente (ex: 011999999999 -> 5511999999999) e prefixa 55 apenas quando o DDD brasileiro for válido.
// Quando a entrada já começa explicitamente com "+" ou "00" internacional, preserva o DDI original.
func NormalizePhone(s string) string {
	trimmed := strings.TrimSpace(s)
	hasExplicitPlus := strings.HasPrefix(trimmed, "+")
	s = nonDigit.ReplaceAllString(s, "")
	if !hasExplicitPlus {
		if strings.HasPrefix(s, "00") && len(s) > 4 {
			return strings.TrimPrefix(s, "00")
		}
		if len(s) >= 11 && len(s) <= 12 && strings.HasPrefix(s, "0") {
			candidate := strings.TrimPrefix(s, "0")
			if len(candidate) >= 2 && validBRDDDs[candidate[:2]] {
				s = candidate
			}
		}
		if (len(s) == 10 || len(s) == 11) && validBRDDDs[s[:2]] {
			s = "55" + s
		}
	}
	return s
}

// NormalizePhoneE164 devolve o telefone no formato E.164 com "+" inicial
// (ex: "+5511999999999"), exigido por TikTok Events API e Google Ads Data Manager.
func NormalizePhoneE164(s string) string {
	digits := NormalizePhone(s)
	if digits == "" {
		return ""
	}
	return "+" + digits
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

var countryMap = map[string]string{
	"brasil": "br", "brazil": "br",
	"estados unidos": "us", "united states": "us", "united states of america": "us", "usa": "us", "eua": "us",
	"portugal": "pt",
	"espanha": "es", "spain": "es",
	"alemanha": "de", "germany": "de", "deutschland": "de",
	"reino unido": "gb", "united kingdom": "gb", "uk": "gb", "inglaterra": "gb", "england": "gb",
	"franca": "fr", "france": "fr",
	"italia": "it", "italy": "it",
	"argentina": "ar",
	"paraguai": "py", "paraguay": "py",
	"uruguai": "uy", "uruguay": "uy",
	"chile": "cl",
	"colombia": "co",
	"mexico": "mx",
	"canada": "ca",
	"japao": "jp", "japan": "jp",
	"australia": "au",
	"angola": "ao",
	"mocambique": "mz", "mozambique": "mz",
	"suica": "ch", "switzerland": "ch",
	"irlanda": "ie", "ireland": "ie",
	"paises baixos": "nl", "holanda": "nl", "netherlands": "nl",
	"belgica": "be", "belgium": "be",
}

// NormalizeCountry aplica o formato ISO 3166-1 alpha-2 lowercase
// exigido pela Meta CAPI (`br`, `us`, `pt`, ...).
func NormalizeCountry(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = stripAccents(s)
	if s == "" {
		return ""
	}

	// Se já for código ISO de 2 letras
	if len(s) == 2 && s[0] >= 'a' && s[0] <= 'z' && s[1] >= 'a' && s[1] <= 'z' {
		return s
	}

	// Busca no mapa canônico; nomes desconhecidos retornam "" em vez de truncar 2 letras erradas.
	if code, ok := countryMap[s]; ok {
		return code
	}
	return ""
}

var brStates = map[string]string{
	"acre": "ac", "alagoas": "al", "amapa": "ap", "amazonas": "am",
	"bahia": "ba", "ceara": "ce", "distrito federal": "df", "espirito santo": "es",
	"goias": "go", "maranhao": "ma", "mato grosso": "mt", "mato grosso do sul": "ms",
	"minas gerais": "mg", "para": "pa", "paraiba": "pb", "parana": "pr",
	"pernambuco": "pe", "piaui": "pi", "rio de janeiro": "rj", "rio grande do norte": "rn",
	"rio grande do sul": "rs", "rondonia": "ro", "roraima": "rr", "santa catarina": "sc",
	"sao paulo": "sp", "sergipe": "se", "tocantins": "to",
}

var usStates = map[string]string{
	"alabama": "al", "alaska": "ak", "arizona": "az", "arkansas": "ar", "california": "ca",
	"colorado": "co", "connecticut": "ct", "delaware": "de", "florida": "fl", "georgia": "ga",
	"hawaii": "hi", "idaho": "id", "illinois": "il", "indiana": "in", "iowa": "ia",
	"kansas": "ks", "kentucky": "ky", "louisiana": "la", "maine": "me", "maryland": "md",
	"massachusetts": "ma", "michigan": "mi", "minnesota": "mn", "mississippi": "ms",
	"missouri": "mo", "montana": "mt", "nebraska": "ne", "nevada": "nv",
	"new hampshire": "nh", "new jersey": "nj", "new mexico": "nm", "new york": "ny",
	"north carolina": "nc", "north dakota": "nd", "ohio": "oh", "oklahoma": "ok",
	"oregon": "or", "pennsylvania": "pa", "rhode island": "ri", "south carolina": "sc",
	"south dakota": "sd", "tennessee": "tn", "texas": "tx", "utah": "ut",
	"vermont": "vt", "virginia": "va", "washington": "wa", "west virginia": "wv",
	"wisconsin": "wi", "wyoming": "wy", "district of columbia": "dc",
}

// NormalizeStateISO devolve o código ANSI/ISO lowercase de 2 caracteres
// (ex: "sp", "rj", "mg", "ca", "ny"). Mapeia nomes por extenso do Brasil e EUA,
// ou preserva a sigla de 2 caracteres se já vier formatada.
func NormalizeStateISO(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = stripAccents(s)
	if s == "" {
		return ""
	}

	// Se já for código de 2 letras
	if len(s) == 2 && s[0] >= 'a' && s[0] <= 'z' && s[1] >= 'a' && s[1] <= 'z' {
		return s
	}

	// Mapeia estados brasileiros
	if code, ok := brStates[s]; ok {
		return code
	}

	// Mapeia estados americanos
	if code, ok := usStates[s]; ok {
		return code
	}

	// Fallback: primeiras 2 letras alfabéticas
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
