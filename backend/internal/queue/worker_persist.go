package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func purchaseValue(in domain.SendEventInput) float64 {
	if in.Value != nil {
		return *in.Value
	}
	return 0
}

func summarizeChannels(results []dispatchResult) string {
	seen := make(map[string]bool, 4)
	out := make([]string, 0, 4)
	for _, r := range results {
		if !seen[r.Channel] {
			seen[r.Channel] = true
			out = append(out, r.Channel)
		}
	}
	return strings.Join(out, ",")
}

// isPermanentHTTPStatus identifica erros definitivos de payload/autenticação nos canais
// (400, 401, 403), que não devem ser retentados 25 vezes pelo Asynq.
func isPermanentHTTPStatus(code int) bool {
	return code == 400 || code == 401 || code == 403
}

// collapseChannelResults ordena para que, dentro de um mesmo canal com múltiplos sub-pixels,
// uma falha prevaleça sobre sucesso parcial na linha consolidada de sent_events.
func collapseChannelResults(results []dispatchResult) []dispatchResult {
	sorted := make([]dispatchResult, len(results))
	copy(sorted, results)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Err == nil && sorted[j].Err != nil
	})
	return sorted
}

func persistResult(ctx context.Context, pool *pgxpool.Pool, accountID int64, in domain.SendEventInput, r dispatchResult) {
	if pool == nil {
		return
	}
	if accountID == 0 {
		slog.Warn("sent_event persist skipped: no account", "event_id", in.EventID, "pixel_name", in.PixelName)
		return
	}
	status := "sent"
	errMsg := ""
	if r.Err != nil {
		status = "failed"
		errMsg = util.RedactSecrets(r.Err.Error())
	}
	// NULL SQL de verdade quando não houver payload: o COALESCE preserva o payload bom anterior.
	var payload *string
	if redacted := util.RedactSecrets(r.RequestBody); redacted != "" {
		payload = &redacted
	}
	channel := r.Channel
	if channel == "" {
		channel = "meta"
	}
	inputJSON := "null"
	if raw, err := json.Marshal(in); err == nil {
		inputJSON = string(raw)
	}
	aggCount := in.AggregatedCount
	if aggCount <= 0 {
		aggCount = 1
	}
	aggItemsJSON := "null"
	if len(in.AggregatedItems) > 0 {
		if raw, err := json.Marshal(in.AggregatedItems); err == nil {
			aggItemsJSON = string(raw)
		}
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO sent_events
			(event_id, event_name, pixel_name, external_id, account_id,
			 channel, status, response_code, response_body, error, payload, purchase_value, input_json,
			 aggregated_count, aggregated_items, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13::jsonb,$14,$15::jsonb,NOW())
		ON CONFLICT (account_id, pixel_name, channel, event_id) DO UPDATE SET
			status           = CASE WHEN sent_events.status = 'sent' THEN 'sent' ELSE EXCLUDED.status END,
			response_code    = EXCLUDED.response_code,
			response_body    = EXCLUDED.response_body,
			error            = EXCLUDED.error,
			payload          = COALESCE(EXCLUDED.payload, sent_events.payload),
			purchase_value   = CASE WHEN EXCLUDED.purchase_value > 0 THEN EXCLUDED.purchase_value ELSE sent_events.purchase_value END,
			input_json       = COALESCE(EXCLUDED.input_json, sent_events.input_json),
			aggregated_count = GREATEST(sent_events.aggregated_count, EXCLUDED.aggregated_count),
			aggregated_items = COALESCE(EXCLUDED.aggregated_items, sent_events.aggregated_items),
			updated_at       = NOW()
	`,
		in.EventID, in.EventName, in.PixelName, in.ExternalID, accountID,
		channel, status, r.StatusCode, util.RedactSecrets(r.Body), errMsg, payload, purchaseValue(in), inputJSON,
		aggCount, aggItemsJSON,
	)
	if err != nil {
		slog.Error("sent_event persist failed",
			"event_id", in.EventID, "pixel_name", in.PixelName, "channel", channel, "error", err)
	}
}
