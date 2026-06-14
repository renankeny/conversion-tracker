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
	AccountID int64     // 0 = todos (owner); >0 = filtra (member)
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
	if f.AccountID > 0 {
		conds = append(conds, "account_id = "+add(f.AccountID))
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

	// (4) total tracked users (filtra só por pixel/account, não por janela — costuma ser
	// "quantos tracked users tem cadastrados nesse pixel/conta")
	tuConds := []string{}
	tuArgs := []any{}
	tuIdx := 0
	tuAdd := func(v any) string {
		tuArgs = append(tuArgs, v)
		tuIdx++
		return "$" + itoa(tuIdx)
	}
	if f.PixelName != "" {
		tuConds = append(tuConds, "pixel_name = "+tuAdd(f.PixelName))
	}
	if f.AccountID > 0 {
		tuConds = append(tuConds, "account_id = "+tuAdd(f.AccountID))
	}
	tuWhere := ""
	if len(tuConds) > 0 {
		tuWhere = "WHERE " + strings.Join(tuConds, " AND ")
	}
	if err := s.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM tracked_users "+tuWhere, tuArgs...,
	).Scan(&out.TrackedUsers); err != nil {
		return nil, err
	}

	return out, nil
}

// GeoStat representa a quantidade de usuários agrupados geograficamente.
type GeoStat struct {
	Country string `json:"country"`
	State   string `json:"state"`
	City    string `json:"city"`
	Count   int64  `json:"count"`
}

// GeoDistribution retorna a distribuição geográfica dos usuários rastreados no período.
func (s *StatsRepo) GeoDistribution(ctx context.Context, f StatsFilter) ([]GeoStat, error) {
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
	if f.AccountID > 0 {
		conds = append(conds, "account_id = "+add(f.AccountID))
	}
	conds = append(conds, "country <> ''")

	where := "WHERE " + strings.Join(conds, " AND ")

	q := `
		SELECT country, state, city, COUNT(*) as count
		FROM tracked_users
		` + where + `
		GROUP BY country, state, city
		ORDER BY count DESC
		LIMIT 200
	`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []GeoStat
	for rows.Next() {
		var g GeoStat
		if err := rows.Scan(&g.Country, &g.State, &g.City, &g.Count); err != nil {
			return nil, err
		}
		stats = append(stats, g)
	}
	return stats, rows.Err()
}

