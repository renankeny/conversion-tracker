package queue

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func matchPhraseTriggerWithTerm(trigger, text string) (string, bool) {
	trigger = strings.TrimSpace(trigger)
	text = strings.TrimSpace(text)
	if trigger == "" || text == "" {
		return "", false
	}
	normText := foldAccentsLower(text)
	for _, rawPart := range splitTriggerTerms(trigger) {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		normPart := foldAccentsLower(part)
		if normPart != "" && strings.Contains(normText, normPart) {
			return part, true
		}
		if rx, err := regexp.Compile("(?i)" + regexp.QuoteMeta(part)); err == nil && rx.MatchString(text) {
			return part, true
		}
	}
	return "", false
}

func splitTriggerTerms(s string) []string {
	rs := []rune(s)
	var parts []string
	start := 0
	for i, r := range rs {
		isSep := r == ';'
		if r == ',' {
			decimalComma := i > 0 && i+1 < len(rs) && unicode.IsDigit(rs[i-1]) && unicode.IsDigit(rs[i+1])
			isSep = !decimalComma
		}
		if isSep {
			parts = append(parts, string(rs[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, string(rs[start:]))
	return parts
}

func foldAccentsLower(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, err := transform.String(t, strings.ToLower(s))
	if err != nil {
		return strings.ToLower(s)
	}
	return res
}
