package repo

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StatsRepo agrega informações de sent_events e tracked_users pra exibição
// no painel (totais por status, por evento, série temporal por dia).
type StatsRepo struct {
	pool *pgxpool.Pool
}

func NewStatsRepo(pool *pgxpool.Pool) *StatsRepo {
	return &StatsRepo{pool: pool}
}

// StatsFilter filtra a janela e o pixel das queries.
type StatsFilter struct {
	From      time.Time // inclusive (created_at >= From)
	To        time.Time // exclusive (created_at < To)
	PixelName string    // vazio = todos
}

// Overview agrupa as três séries básicas que o painel costuma mostrar.
type Overview struct {
	StatusTotals  map[string]int64        `json:"status_totals"`   // {sent: N, failed: M, queued: K}
	EventTotals   map[string]int64        `json:"event_totals"`    // {PageView: N, Purchase: M, ...}
	PerDay        []DailyBucket           `json:"per_day"`         // série temporal
	TotalEvents   int64                   `json:"total_events"`    // soma de tudo
	TrackedUsers  int64                   `json:"tracked_users"`   // total de tracked_users (sem filtro de janela; com filtro de pixel)
}

// DailyBucket é uma linha da série temporal por dia.
type DailyBucket struct {
	Date   string `json:"date"`   // YYYY-MM-DD na timezone do servidor (America/Sao_Paulo)
	Sent   int64  `json:"sent"`
	Failed int64  `json:"failed"`
	Queued int64  `json:"queued"`
}

// Overview executa as 4 queries necessárias e devolve o overview agregado.
func (s *StatsRepo) Overview(ctx context.Context, f StatsFilter) (*Overview, error) {
	args := []any{}
	conds := []string{}
	idx := 0
	add := func(v any) string {
		args = append(args, v)
		idx++
		return "$" + itoa(idx)
	}
	if !f.From.IsZero() {
		conds = append(conds, "created_at >= "+add(f.From))
	}
	if !f.To.IsZero() {
		conds = append(conds, "created_at < "+add(f.To))
	}
	if f.PixelName != "" {
		conds = append(conds, "pixel_name = "+add(f.PixelName))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	out := &Overview{
		StatusTotals: map[string]int64{},
		EventTotals:  map[string]int64{},
		PerDay:       []DailyBucket{},
	}

	// (1) status totals
	rows, err := s.pool.Query(ctx, "SELECT status, COUNT(*) FROM sent_events "+where+" GROUP BY status", args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var n int64
		if err := rows.Scan(&k, &n); err != nil {
			rows.Close()
			return nil, err
		}
		out.StatusTotals[k] = n
		out.TotalEvents += n
	}
	rows.Close()

	// (2) event_name totals
	rows, err = s.pool.Query(ctx, "SELECT event_name, COUNT(*) FROM sent_events "+where+" GROUP BY event_name ORDER BY 2 DESC", args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var n int64
		if err := rows.Scan(&k, &n); err != nil {
			rows.Close()
			return nil, err
		}
		out.EventTotals[k] = n
	}
	rows.Close()

	// (3) per_day (no fuso da sessão = America/Sao_Paulo via SET TIME ZONE)
	rows, err = s.pool.Query(ctx, `
		SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day,
		       SUM(CASE WHEN status='sent'   THEN 1 ELSE 0 END) AS sent,
		       SUM(CASE WHEN status='failed' THEN 1 ELSE 0 END) AS failed,
		       SUM(CASE WHEN status='queued' THEN 1 ELSE 0 END) AS queued
		FROM sent_events `+where+`
		GROUP BY 1
		ORDER BY 1 ASC
	`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var b DailyBucket
		if err := rows.Scan(&b.Date, &b.Sent, &b.Failed, &b.Queued); err != nil {
			rows.Close()
			return nil, err
		}
		out.PerDay = append(out.PerDay, b)
	}
	rows.Close()

	// (4) total tracked users (filtra só por pixel, não por janela — costuma ser
	// "quantos tracked users tem cadastrados nesse pixel")
	if f.PixelName != "" {
		if err := s.pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM tracked_users WHERE pixel_name = $1", f.PixelName,
		).Scan(&out.TrackedUsers); err != nil {
			return nil, err
		}
	} else {
		if err := s.pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM tracked_users",
		).Scan(&out.TrackedUsers); err != nil {
			return nil, err
		}
	}

	return out, nil
}
