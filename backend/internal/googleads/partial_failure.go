package googleads

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func isDestinationNotFoundBody(rawBody string) bool {
	lower := strings.ToLower(rawBody)
	return strings.Contains(lower, "destination_references") &&
		(strings.Contains(lower, "not_found") || strings.Contains(lower, "resource not found"))
}

func isUploadCompatibleType(actionType string) bool {
	tp := strings.ToUpper(strings.TrimSpace(actionType))
	return tp == "" || tp == "UPLOAD_CLICKS" || tp == "UPLOAD_CALLS" || tp == "STORE_SALES" || tp == "STORE_SALES_DIRECT_UPLOAD"
}

func formatGoogleDataManagerError(statusCode int, rawBody string, action *domain.GoogleAdsConversionAction, operatingID string) string {
	if isDestinationNotFoundBody(rawBody) {
		actID := ""
		actType := ""
		if action != nil {
			actID = strings.TrimSpace(action.ConversionActionID)
			actType = strings.ToUpper(strings.TrimSpace(action.ActionType))
		}
		if !isUploadCompatibleType(actType) {
			return fmt.Sprintf(
				"A Ação de Conversão selecionada (ID %s) é do tipo %s, que não aceita envios via API Offline. Escolha ou clique em '+ Criar Conversão API' para usar uma conversão do tipo Importação de Cliques (UPLOAD_CLICKS).",
				actID, actType,
			)
		}
		return fmt.Sprintf(
			"Ação de Conversão ID %s não encontrada como destino de API na conta %s. Confirme se ela é do tipo 'Importar -> Cliques' (UPLOAD_CLICKS) ou, se acabou de criá-la no Google Ads, aguarde alguns minutos até o Google propagar na Data Manager API.",
			actID, operatingID,
		)
	}
	return fmt.Sprintf("google ads api returned %d: %s", statusCode, rawBody)
}

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
