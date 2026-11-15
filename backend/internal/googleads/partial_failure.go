package googleads

import (
	"encoding/json"
	"fmt"
	"strings"
)

// detectPartialFailure inspeciona respostas HTTP 2xx da Google Ads / Data Manager API
// em busca de falhas parciais (GCLID expirado, conversion action invalida, etc.).
func detectPartialFailure(rawBody string) string {
	trimmed := strings.TrimSpace(rawBody)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return ""
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
		return ""
	}

	for _, topKey := range []string{"partialFailureError", "partialFailureStatus", "error"} {
		if msg := extractStatusMessage(doc[topKey], false); msg != "" {
			return msg
		}
	}

	for _, listKey := range []string{"results", "responses", "requestStatusPerDestination"} {
		items, ok := doc[listKey].([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if msg := extractStatusMessage(m["error"], false); msg != "" {
				return msg
			}
			if msg := extractStatusMessage(m["partialFailureError"], false); msg != "" {
				return msg
			}
			if msg := extractStatusMessage(m["status"], true); msg != "" {
				return msg
			}
			if rs, ok := m["requestStatus"].(string); ok && strings.EqualFold(strings.TrimSpace(rs), "FAILURE") {
				return extractDestinationFailureReason(m)
			}
		}
	}
	return ""
}

func extractStatusMessage(v any, requireNonZeroOrError bool) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		s = strings.TrimSpace(s)
		if requireNonZeroOrError && (strings.EqualFold(s, "OK") || strings.EqualFold(s, "SUCCESS")) {
			return ""
		}
		return s
	}
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	codeVal, hasCode := m["code"].(float64)
	if hasCode && int(codeVal) == 0 {
		return ""
	}
	if msg, ok := m["message"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	if st, ok := m["status"].(string); ok {
		st = strings.TrimSpace(st)
		if st != "" && !strings.EqualFold(st, "OK") && !strings.EqualFold(st, "SUCCESS") {
			return st
		}
	}
	if hasCode && int(codeVal) != 0 {
		return fmt.Sprintf("code %d", int(codeVal))
	}
	return ""
}

func extractDestinationFailureReason(m map[string]any) string {
	if warn, ok := m["warningInfo"].(map[string]any); ok {
		if reasons, ok := warn["warningReasons"].([]any); ok && len(reasons) > 0 {
			if r0, ok := reasons[0].(map[string]any); ok {
				if reason, ok := r0["reason"].(string); ok && strings.TrimSpace(reason) != "" {
					return strings.TrimSpace(reason)
				}
			}
		}
	}
	return "destination requestStatus FAILURE"
}
