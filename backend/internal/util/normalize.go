package util

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

// IsSHA256Hex verifica se s já é um digest SHA-256 hexadecimal válido de 64 caracteres,
// prevenindo double-hashing quando payloads de API/webhook já chegam hasheados.
func IsSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < 64; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// NormalizeName remove acentos latinos, espaços, números e pontuação, mantendo letras Unicode lowercase.
// Usado pra Meta CAPI advanced matching.
func NormalizeName(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	if IsSHA256Hex(trimmed) {
		return strings.ToLower(trimmed)
	}
	s = stripAccents(strings.ToLower(trimmed))
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsLetter(r) {
			out = append(out, r)
		}
	}
	return string(out)
}

// NormalizeCity igual a NormalizeName.
func NormalizeCity(s string) string { return NormalizeName(s) }

// NormalizeEmail lowercase + trim.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// SHA256Hex devolve o hash SHA256 hex (lowercase) de s. Se s vazio, devolve "".
// Se s já for um hash SHA-256 hex de 64 caracteres, preserva em lowercase sem re-hashear.
func SHA256Hex(s string) string {
	if s == "" {
		return ""
	}
	if IsSHA256Hex(s) {
		return strings.ToLower(s)
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

// NormalizeZip aplica o formato exigido pela Meta CAPI sem contexto explícito de país.
func NormalizeZip(s string) string {
	return NormalizeZipWithCountry(s, "")
}

// NormalizeZipWithCountry aplica o formato exigido pela Meta CAPI:
// lowercase, remove espaços e traços, mantém alfanumérico.
// Para códigos dos EUA (country="us" ou formato ZIP+4 "12345-6789"), mantém apenas os 5 primeiros dígitos.
func NormalizeZipWithCountry(s, country string) string {
	trimmed := strings.ToLower(strings.TrimSpace(s))
	if trimmed == "" {
		return ""
	}
	if IsSHA256Hex(trimmed) {
		return trimmed
	}
	isUSZipPlus4 := len(trimmed) == 10 && trimmed[5] == '-'
	out := make([]rune, 0, len(trimmed))
	allDigits := true
	for _, r := range trimmed {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		} else if r >= 'a' && r <= 'z' {
			allDigits = false
			out = append(out, r)
		}
	}
	normCountry := NormalizeCountry(country)
	if (normCountry == "us" || (normCountry == "" && isUSZipPlus4)) && allDigits && len(out) >= 5 {
		return string(out[:5])
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

// NormalizeCountry aplica o formato ISO 3166-1 alpha-2 lowercase exigido pela Meta CAPI.
func NormalizeCountry(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if IsSHA256Hex(s) {
		return s
	}
	s = stripAccents(s)
	if s == "" {
		return ""
	}
	if len(s) == 2 && s[0] >= 'a' && s[0] <= 'z' && s[1] >= 'a' && s[1] <= 'z' {
		return s
	}
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

// NormalizeStateISO devolve o código ANSI/ISO lowercase de 2 caracteres.
func NormalizeStateISO(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if IsSHA256Hex(s) {
		return s
	}
	s = stripAccents(s)
	if s == "" {
		return ""
	}
	if len(s) == 2 && s[0] >= 'a' && s[0] <= 'z' && s[1] >= 'a' && s[1] <= 'z' {
		return s
	}
	if code, ok := brStates[s]; ok {
		return code
	}
	if code, ok := usStates[s]; ok {
		return code
	}
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
