// Package maintenance executa rotinas periódicas de limpeza de dados.
package maintenance

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PruneResult resume uma execução de limpeza.
type PruneResult struct {
	Table         string        `json:"table"`
	RetentionDays int           `json:"retention_days"`
	Threshold     time.Time     `json:"threshold"`
	Deleted       int64         `json:"deleted"`
	Duration      time.Duration `json:"duration_ms"`
}

// PruneOldSentEventsForAccount apaga rows com `created_at` mais antigos que `days`
// dentro de uma conta específica. Se days <= 0, não faz nada.
func PruneOldSentEventsForAccount(ctx context.Context, pool *pgxpool.Pool, accountID int64, days int) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{Table: "sent_events", RetentionDays: days}, nil
	}
	threshold := time.Now().UTC().AddDate(0, 0, -days)
	start := time.Now()

	const batchSize = 5000
	var totalDeleted int64
	const q = `
		WITH victims AS (
			SELECT id FROM sent_events
			WHERE account_id = $1 AND created_at < $2
			LIMIT 5000
		)
		DELETE FROM sent_events WHERE id IN (SELECT id FROM victims)
	`
	for {
		select {
		case <-ctx.Done():
			return PruneResult{Table: "sent_events", RetentionDays: days, Threshold: threshold, Deleted: totalDeleted, Duration: time.Since(start)}, ctx.Err()
		default:
		}
		tag, err := pool.Exec(ctx, q, accountID, threshold)
		if err != nil {
			return PruneResult{}, fmt.Errorf("delete sent_events for account %d: %w", accountID, err)
		}
		n := tag.RowsAffected()
		totalDeleted += n
		if n < batchSize {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return PruneResult{
		Table:         "sent_events",
		RetentionDays: days,
		Threshold:     threshold,
		Deleted:       totalDeleted,
		Duration:      time.Since(start),
	}, nil
}

// PruneOldEventCountersForAccount apaga buckets de uma conta específica.
func PruneOldEventCountersForAccount(ctx context.Context, pool *pgxpool.Pool, accountID int64, days int) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{Table: "event_counters", RetentionDays: days}, nil
	}
	threshold := time.Now().UTC().AddDate(0, 0, -days)
	start := time.Now()
	tag, err := pool.Exec(ctx,
		`DELETE FROM event_counters WHERE account_id = $1 AND bucket_at < $2`,
		accountID, threshold)
	if err != nil {
		return PruneResult{}, fmt.Errorf("delete event_counters for account %d: %w", accountID, err)
	}
	return PruneResult{
		Table:         "event_counters",
		RetentionDays: days,
		Threshold:     threshold,
		Deleted:       tag.RowsAffected(),
		Duration:      time.Since(start),
	}, nil
}

// ResetAccountData apaga TODOS os sent_events e event_counters de uma conta.
// Operação destrutiva — só deve ser chamada com confirmação explícita.
// Não toca em pixels, tracked_users, url_mappings, api_keys, admin_users
// (esses precisam de remoção manual da conta inteira pra serem afetados).
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

// PruneOldSentEvents apaga rows com `created_at` mais antigos que `days`.
// Faz DELETE em batches de até 5000 linhas pra não travar locks longos.
// Se days <= 0, não faz nada (retention desabilitado).
func PruneOldSentEvents(ctx context.Context, pool *pgxpool.Pool, days int) (PruneResult, error) {
	return pruneByCreatedAt(ctx, pool, "sent_events", days)
}

// PruneOldEventCounters apaga buckets com `bucket_at` mais antigos que `days`.
// Counter retém histórico pra dashboards, então o default sugerido é 90+ dias.
func PruneOldEventCounters(ctx context.Context, pool *pgxpool.Pool, days int) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{Table: "event_counters", RetentionDays: days}, nil
	}
	threshold := time.Now().UTC().AddDate(0, 0, -days)
	start := time.Now()
	tag, err := pool.Exec(ctx,
		`DELETE FROM event_counters WHERE bucket_at < $1`, threshold)
	if err != nil {
		return PruneResult{}, fmt.Errorf("delete event_counters: %w", err)
	}
	return PruneResult{
		Table:         "event_counters",
		RetentionDays: days,
		Threshold:     threshold,
		Deleted:       tag.RowsAffected(),
		Duration:      time.Since(start),
	}, nil
}

func pruneByCreatedAt(ctx context.Context, pool *pgxpool.Pool, table string, days int) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{Table: table, RetentionDays: days}, nil
	}
	threshold := time.Now().UTC().AddDate(0, 0, -days)
	start := time.Now()

	// DELETE em batches usando subquery pra não escanear a tabela toda.
	const batchSize = 5000
	var totalDeleted int64
	q := fmt.Sprintf(`
		WITH victims AS (
			SELECT id FROM %s WHERE created_at < $1 LIMIT %d
		)
		DELETE FROM %s WHERE id IN (SELECT id FROM victims)
	`, table, batchSize, table)

	for {
		select {
		case <-ctx.Done():
			return PruneResult{
				Table:         table,
				RetentionDays: days,
				Threshold:     threshold,
				Deleted:       totalDeleted,
				Duration:      time.Since(start),
			}, ctx.Err()
		default:
		}

		tag, err := pool.Exec(ctx, q, threshold)
		if err != nil {
			return PruneResult{}, fmt.Errorf("delete %s batch: %w", table, err)
		}
		n := tag.RowsAffected()
		totalDeleted += n
		if n < batchSize {
			break
		}
		// Pequena pausa entre batches pra liberar locks.
		time.Sleep(50 * time.Millisecond)
	}

	return PruneResult{
		Table:         table,
		RetentionDays: days,
		Threshold:     threshold,
		Deleted:       totalDeleted,
		Duration:      time.Since(start),
	}, nil
}

// PerAccountRetentionLister é a interface usada pelo scheduler pra listar
// as contas habilitadas com seus overrides de retenção. Definida aqui pra
// evitar import cycle com o package `repo`.
type PerAccountRetentionLister interface {
	ListRetentions(ctx context.Context) ([]PerAccountRetention, error)
}

type PerAccountRetention struct {
	AccountID                  int64
	SentEventsRetentionDays    *int
	EventCountersRetentionDays *int
}

// Scheduler executa as rotinas de prune diariamente à hora especificada
// (timezone São Paulo). Loop sai quando ctx é cancelado.
//
// A retenção é aplicada por conta, com fallback para os defaults globais:
//   - Se a conta tem override (sent_events_retention_days NOT NULL), usa esse.
//   - Senão, usa SentEventsRetention (env global).
//   - Se Accounts é nil (modo legado), aplica retention global em toda a tabela.
type Scheduler struct {
	Pool                   *pgxpool.Pool
	Accounts               PerAccountRetentionLister
	SentEventsRetention    int
	EventCountersRetention int
	HourSP                 int // hora local SP em que roda (0-23)
	MinuteSP               int // minuto local SP (0-59)
}

func (s *Scheduler) Run(ctx context.Context) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		// Se zoneinfo falhou, cai pra UTC. Container já tem tzdata então é raro.
		loc = time.UTC
	}

	for {
		next := nextRunAt(time.Now().In(loc), s.HourSP, s.MinuteSP)
		wait := time.Until(next)
		slog.Info("maintenance scheduled",
			"next_run", next.Format(time.RFC3339),
			"wait_seconds", int(wait.Seconds()),
			"sent_events_retention_days", s.SentEventsRetention,
			"event_counters_retention_days", s.EventCountersRetention,
		)

		select {
		case <-ctx.Done():
			slog.Info("maintenance scheduler stopped")
			return
		case <-time.After(wait):
		}

		s.runOnce(ctx)
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	if s.Accounts == nil {
		// Modo legado: aplica retention global na tabela inteira.
		s.runOnceGlobal(ctx)
		return
	}

	retentions, err := s.Accounts.ListRetentions(ctx)
	if err != nil {
		slog.Error("maintenance list retentions failed", "err", err)
		// Fallback pra modo global se o lookup falhar.
		s.runOnceGlobal(ctx)
		return
	}

	for _, r := range retentions {
		sentDays := s.SentEventsRetention
		if r.SentEventsRetentionDays != nil {
			sentDays = *r.SentEventsRetentionDays
		}
		countersDays := s.EventCountersRetention
		if r.EventCountersRetentionDays != nil {
			countersDays = *r.EventCountersRetentionDays
		}

		if sentDays > 0 {
			if pr, err := PruneOldSentEventsForAccount(ctx, s.Pool, r.AccountID, sentDays); err != nil {
				slog.Error("prune sent_events failed",
					"account_id", r.AccountID, "err", err)
			} else if pr.RetentionDays > 0 {
				slog.Info("prune sent_events ok",
					"account_id", r.AccountID,
					"deleted", pr.Deleted,
					"days", sentDays,
					"duration_ms", pr.Duration.Milliseconds(),
				)
			}
		}
		if countersDays > 0 {
			if pr, err := PruneOldEventCountersForAccount(ctx, s.Pool, r.AccountID, countersDays); err != nil {
				slog.Error("prune event_counters failed",
					"account_id", r.AccountID, "err", err)
			} else if pr.RetentionDays > 0 {
				slog.Info("prune event_counters ok",
					"account_id", r.AccountID,
					"deleted", pr.Deleted,
					"days", countersDays,
					"duration_ms", pr.Duration.Milliseconds(),
				)
			}
		}
	}
}

func (s *Scheduler) runOnceGlobal(ctx context.Context) {
	if r, err := PruneOldSentEvents(ctx, s.Pool, s.SentEventsRetention); err != nil {
		slog.Error("prune sent_events failed", "err", err)
	} else if r.RetentionDays > 0 {
		slog.Info("prune sent_events ok",
			"deleted", r.Deleted,
			"threshold", r.Threshold.Format(time.RFC3339),
			"duration_ms", r.Duration.Milliseconds(),
		)
	}
	if r, err := PruneOldEventCounters(ctx, s.Pool, s.EventCountersRetention); err != nil {
		slog.Error("prune event_counters failed", "err", err)
	} else if r.RetentionDays > 0 {
		slog.Info("prune event_counters ok",
			"deleted", r.Deleted,
			"threshold", r.Threshold.Format(time.RFC3339),
			"duration_ms", r.Duration.Milliseconds(),
		)
	}
}

func nextRunAt(now time.Time, hour, minute int) time.Time {
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !target.After(now) {
		target = target.AddDate(0, 0, 1)
	}
	return target
}
