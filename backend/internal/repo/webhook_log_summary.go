package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

const maxRawPayloadBytes = 128 * 1024

// normalizeRawPayloadJSON garante que qualquer corpo bruto salvo em raw_payload
// seja JSON válido e limitado em tamanho antes do cast ::jsonb no Postgres.
func normalizeRawPayloadJSON(raw []byte) *string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}
	if len(trimmed) > maxRawPayloadBytes {
		wrapped, err := json.Marshal(map[string]any{
			"truncated": true,
			"bytes":     len(trimmed),
			"preview":   string(trimmed[:4096]),
		})
		if err != nil {
			return nil
		}
		s := string(wrapped)
		return &s
	}
	if json.Valid(trimmed) {
		s := string(trimmed)
		return &s
	}
	wrapped, err := json.Marshal(map[string]string{
		"raw_body": string(trimmed),
	})
	if err != nil {
		return nil
	}
	s := string(wrapped)
	return &s
}

// SummaryBySource retorna contadores e última data de recebimento por gateway.
func (r *WebhookLogRepo) SummaryBySource(ctx context.Context, accountID int64) (map[string]domain.PlatformWebhookStats, error) {
	out := make(map[string]domain.PlatformWebhookStats)
	if accountID < 0 {
		return out, nil
	}

	const q = `
		SELECT LOWER(source) AS src,
		       COUNT(*) FILTER (WHERE status IN ('queued', 'buffered', 'replayed')) AS accepted,
		       COUNT(*) FILTER (WHERE status = 'failed') AS failed,
		       MAX(received_at) AS last_at
		FROM webhook_logs
		WHERE ($1 = 0 OR account_id = $1)
		GROUP BY LOWER(source)
	`
	rows, err := r.pool.Query(ctx, q, accountID)
	if err != nil {
		return nil, fmt.Errorf("summary webhook_logs by source: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var src string
		var st domain.PlatformWebhookStats
		var lastAt *time.Time
		if err := rows.Scan(&src, &st.TotalAccepted, &st.TotalFailed, &lastAt); err != nil {
			return nil, fmt.Errorf("scan webhook_logs summary: %w", err)
		}
		st.LastReceivedAt = lastAt
		out[strings.TrimSpace(src)] = st
	}
	return out, rows.Err()
}
