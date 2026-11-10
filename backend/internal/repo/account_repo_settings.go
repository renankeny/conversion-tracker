package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// UpdateEventLabelOverrides atualiza apenas os nomes customizados de eventos
// da conta (cards/funil da dashboard). Cada empresa tem os seus próprios.
func (r *AccountRepo) UpdateEventLabelOverrides(ctx context.Context, accountID int64, overrides map[string]string) error {
	if overrides == nil {
		overrides = map[string]string{}
	}
	raw, err := json.Marshal(overrides)
	if err != nil {
		return err
	}
	const q = `UPDATE accounts SET event_label_overrides = $1, updated_at = NOW() WHERE id = $2 RETURNING id`
	var id int64
	err = r.pool.QueryRow(ctx, q, raw, accountID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// UpdateEventsFilter atualiza (substitui) o filtro de eventos da página de
// Eventos pra um contexto específico (pixelKey = nome do pixel, ou "" pra
// "Todos os Pixels"). Usa jsonb_set em vez de reescrever o mapa inteiro —
// não afeta os filtros já salvos de outros pixels da mesma conta.
func (r *AccountRepo) UpdateEventsFilter(ctx context.Context, accountID int64, pixelKey string, events []string) error {
	if events == nil {
		events = []string{}
	}
	raw, err := json.Marshal(events)
	if err != nil {
		return err
	}
	const q = `UPDATE accounts
		SET events_filter = jsonb_set(COALESCE(events_filter, '{}'::jsonb), ARRAY[$1::text], $2::jsonb, true),
		    updated_at = NOW()
		WHERE id = $3 RETURNING id`
	var id int64
	err = r.pool.QueryRow(ctx, q, pixelKey, raw, accountID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// UpdateRetention atualiza apenas as colunas de retenção. Use NULL pra
// resetar pro default global; 0 pra desabilitar; N>0 pra N dias.
func (r *AccountRepo) UpdateRetention(ctx context.Context, accountID int64, sentDays, countersDays *int) error {
	const q = `UPDATE accounts SET sent_events_retention_days = $1, event_counters_retention_days = $2, updated_at = NOW()
	           WHERE id = $3 RETURNING id`
	var id int64
	err := r.pool.QueryRow(ctx, q, sentDays, countersDays, accountID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// AccountRetention é o que ListRetentions devolve. Mesma forma que
// maintenance.PerAccountRetention pra facilitar adapters no scheduler de prune.
type AccountRetention struct {
	AccountID                  int64
	SentEventsRetentionDays    *int
	EventCountersRetentionDays *int
}

// ListRetentions devolve as contas habilitadas + seus overrides de retenção.
// Usada pelo scheduler de prune.
func (r *AccountRepo) ListRetentions(ctx context.Context) ([]AccountRetention, error) {
	const q = `SELECT id, sent_events_retention_days, event_counters_retention_days FROM accounts WHERE enabled = TRUE`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccountRetention
	for rows.Next() {
		var ar AccountRetention
		if err := rows.Scan(&ar.AccountID, &ar.SentEventsRetentionDays, &ar.EventCountersRetentionDays); err != nil {
			return nil, err
		}
		out = append(out, ar)
	}
	return out, rows.Err()
}

