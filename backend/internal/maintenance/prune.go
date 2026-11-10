// Package maintenance executa rotinas periódicas de limpeza de dados.
package maintenance

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PruneResult struct {
	Table         string        `json:"table"`
	RetentionDays int           `json:"retention_days"`
	Threshold     time.Time     `json:"threshold"`
	Deleted       int64         `json:"deleted"`
	Duration      time.Duration `json:"duration_ms"`
}

// PruneOldSentEventsForAccount limpa payloads pesados de rows antigas preservando LTV e metadados.
func PruneOldSentEventsForAccount(ctx context.Context, pool *pgxpool.Pool, accountID int64, days int) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{Table: "sent_events", RetentionDays: days}, nil
	}
	const q = `
		WITH victims AS (
			SELECT id FROM sent_events
			WHERE account_id = $1 AND created_at < $2
			  AND (payload IS NOT NULL OR response_body <> '' OR input_json IS NOT NULL)
			LIMIT 5000
		)
		UPDATE sent_events SET
			payload = NULL,
			response_body = '',
			input_json = NULL
		WHERE id IN (SELECT id FROM victims)
	`
	return runBatchedThreshold(ctx, pool, "sent_events", days, func(ctx context.Context, threshold time.Time) (int64, error) {
		tag, err := pool.Exec(ctx, q, accountID, threshold)
		if err != nil {
			return 0, fmt.Errorf("prune sent_events for account %d: %w", accountID, err)
		}
		return tag.RowsAffected(), nil
	})
}

func PruneOldSentEvents(ctx context.Context, pool *pgxpool.Pool, days int) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{Table: "sent_events", RetentionDays: days}, nil
	}
	const q = `
		WITH victims AS (
			SELECT id FROM sent_events
			WHERE created_at < $1
			  AND (payload IS NOT NULL OR response_body <> '' OR input_json IS NOT NULL)
			LIMIT 5000
		)
		UPDATE sent_events SET
			payload = NULL,
			response_body = '',
			input_json = NULL
		WHERE id IN (SELECT id FROM victims)
	`
	return runBatchedThreshold(ctx, pool, "sent_events", days, func(ctx context.Context, threshold time.Time) (int64, error) {
		tag, err := pool.Exec(ctx, q, threshold)
		if err != nil {
			return 0, fmt.Errorf("prune sent_events: %w", err)
		}
		return tag.RowsAffected(), nil
	})
}

func PruneOldEventCountersForAccount(ctx context.Context, pool *pgxpool.Pool, accountID int64, days int) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{Table: "event_counters", RetentionDays: days}, nil
	}
	threshold := time.Now().UTC().AddDate(0, 0, -days)
	start := time.Now()
	tag, err := pool.Exec(ctx, `DELETE FROM event_counters WHERE account_id = $1 AND bucket_at < $2`, accountID, threshold)
	if err != nil {
		return PruneResult{}, fmt.Errorf("delete event_counters for account %d: %w", accountID, err)
	}
	return PruneResult{Table: "event_counters", RetentionDays: days, Threshold: threshold, Deleted: tag.RowsAffected(), Duration: time.Since(start)}, nil
}

func PruneOldEventCounters(ctx context.Context, pool *pgxpool.Pool, days int) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{Table: "event_counters", RetentionDays: days}, nil
	}
	threshold := time.Now().UTC().AddDate(0, 0, -days)
	start := time.Now()
	tag, err := pool.Exec(ctx, `DELETE FROM event_counters WHERE bucket_at < $1`, threshold)
	if err != nil {
		return PruneResult{}, fmt.Errorf("delete event_counters: %w", err)
	}
	return PruneResult{Table: "event_counters", RetentionDays: days, Threshold: threshold, Deleted: tag.RowsAffected(), Duration: time.Since(start)}, nil
}

// PruneOldWebhookLogs limpa raw_payload e input_json antigos para evitar crescimento infinito de webhook_logs.
func PruneOldWebhookLogs(ctx context.Context, pool *pgxpool.Pool, days int) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{Table: "webhook_logs", RetentionDays: days}, nil
	}
	const q = `
		WITH victims AS (
			SELECT id FROM webhook_logs
			WHERE received_at < $1
			  AND (raw_payload IS NOT NULL OR input_json IS NOT NULL)
			LIMIT 5000
		)
		UPDATE webhook_logs SET raw_payload = NULL, input_json = NULL
		WHERE id IN (SELECT id FROM victims)
	`
	return runBatchedThreshold(ctx, pool, "webhook_logs", days, func(ctx context.Context, threshold time.Time) (int64, error) {
		tag, err := pool.Exec(ctx, q, threshold)
		if err != nil {
			return 0, fmt.Errorf("prune webhook_logs: %w", err)
		}
		return tag.RowsAffected(), nil
	})
}

func ResetAccountData(ctx context.Context, pool *pgxpool.Pool, accountID int64) (sentEvents, counters int64, err error) {
	if accountID <= 0 {
		return 0, 0, fmt.Errorf("account_id obrigatório")
	}
	tag1, err := pool.Exec(ctx, `DELETE FROM sent_events WHERE account_id = $1`, accountID)
	if err != nil {
		return 0, 0, fmt.Errorf("delete sent_events: %w", err)
	}
	tag2, err := pool.Exec(ctx, `DELETE FROM event_counters WHERE account_id = $1`, accountID)
	if err != nil {
		return tag1.RowsAffected(), 0, fmt.Errorf("delete event_counters: %w", err)
	}
	return tag1.RowsAffected(), tag2.RowsAffected(), nil
}

func runBatchedThreshold(
	ctx context.Context,
	pool *pgxpool.Pool,
	table string,
	days int,
	execBatch func(context.Context, time.Time) (int64, error),
) (PruneResult, error) {
	if pool == nil {
		return PruneResult{Table: table, RetentionDays: days}, nil
	}
	threshold := time.Now().UTC().AddDate(0, 0, -days)
	start := time.Now()
	const batchSize = 5000
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return PruneResult{Table: table, RetentionDays: days, Threshold: threshold, Deleted: total, Duration: time.Since(start)}, err
		}
		n, err := execBatch(ctx, threshold)
		if err != nil {
			return PruneResult{}, err
		}
		total += n
		if n < batchSize {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return PruneResult{Table: table, RetentionDays: days, Threshold: threshold, Deleted: total, Duration: time.Since(start)}, nil
}
