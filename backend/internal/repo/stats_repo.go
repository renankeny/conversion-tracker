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
	From          time.Time // inclusive (created_at >= From)
	To            time.Time // exclusive (created_at < To)
	PixelName     string    // vazio = todos
	AccountID     int64     // 0 = todos (owner); >0 = filtra (member)
	TrafficSource string    // all, meta_lp, meta_wpp, google, tiktok
}

// TrafficQualityStats agrupa métricas de qualidade e auditoria de tráfego.
type TrafficQualityStats struct {
	TotalVisitors       int64            `json:"total_visitors"`
	HumanCount          int64            `json:"human_count"`
	HumanPercent        float64          `json:"human_percent"`
	BotCount            int64            `json:"bot_count"`
	BotPercent          float64          `json:"bot_percent"`
	InconclusiveCount   int64            `json:"inconclusive_count"`
	InconclusivePercent float64          `json:"inconclusive_percent"`
	BotCategories       map[string]int64 `json:"bot_categories"`
}

// Overview agrupa as três séries básicas que o painel costuma mostrar.
type Overview struct {
	StatusTotals   map[string]int64    `json:"status_totals"`   // {sent: N, failed: M, queued: K}
	EventTotals    map[string]int64    `json:"event_totals"`    // {PageView: N, Purchase: M, ...}
	PerDay         []DailyBucket       `json:"per_day"`         // série temporal
	TotalEvents    int64               `json:"total_events"`    // soma de tudo
	TotalRevenue   float64             `json:"total_revenue"`   // soma de purchase_value onde event='Purchase' e status='sent'
	TrackedUsers    int64               `json:"tracked_users"`    // total de tracked_users (sem filtro de janela; com filtro de pixel)
	TrafficQuality  TrafficQualityStats `json:"traffic_quality"`  // auditoria de bots vs humanos
	SourceBreakdown SourceBreakdown     `json:"source_breakdown"` // métricas rastreadas separadas por Meta Ads e Google Ads
}

// DailyBucket é uma linha da série temporal por dia.
type DailyBucket struct {
	Date   string `json:"date"` // YYYY-MM-DD na timezone do servidor (America/Sao_Paulo)
	Sent   int64  `json:"sent"`
	Failed int64  `json:"failed"`
	Queued int64  `json:"queued"`
}

// GeoStat representa a quantidade de usuários agrupados geograficamente.
type GeoStat struct {
	Country string `json:"country"`
	State   string `json:"state"`
	City    string `json:"city"`
	Count   int64  `json:"count"`
}

// GeoStatByEvent representa a distribuição geográfica por tipo de evento.
type GeoStatByEvent struct {
	Country   string `json:"country"`
	State     string `json:"state"`
	City      string `json:"city"`
	EventName string `json:"event_name"`
	Count     int64  `json:"count"`
}

// Overview executa as queries necessárias e devolve o overview agregado, aplicando filtros de tráfego.
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
		conds = append(conds, "e.created_at >= "+add(f.From))
	}
	if !f.To.IsZero() {
		conds = append(conds, "e.created_at < "+add(f.To))
	}
	if f.PixelName != "" {
		conds = append(conds, "e.pixel_name = "+add(f.PixelName))
	}
	if f.AccountID != 0 {
		conds = append(conds, "e.account_id = "+add(f.AccountID))
	}

	fromTable := "sent_events e"
	joinClause := ""
	if srcCond := trafficSourceUserCond(f.TrafficSource, "u."); srcCond != "" {
		joinClause = " JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id"
		conds = append(conds, srcCond)
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
	statusQuery := "SELECT e.status, COUNT(DISTINCT e.event_id) FROM " + fromTable + joinClause + " " + where + " GROUP BY e.status"
	rows, err := s.pool.Query(ctx, statusQuery, args...)
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
	eventQuery := "SELECT e.event_name, COUNT(DISTINCT e.event_id) FROM " + fromTable + joinClause + " " + where + " GROUP BY e.event_name ORDER BY 2 DESC"
	rows, err = s.pool.Query(ctx, eventQuery, args...)
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

	// (3) per_day
	dayQuery := `
		SELECT to_char(date_trunc('day', e.created_at), 'YYYY-MM-DD') AS day,
		       COUNT(DISTINCT CASE WHEN e.status='sent'   THEN e.event_id END) AS sent,
		       COUNT(DISTINCT CASE WHEN e.status='failed' THEN e.event_id END) AS failed,
		       COUNT(DISTINCT CASE WHEN e.status='queued' THEN e.event_id END) AS queued
		FROM ` + fromTable + joinClause + ` ` + where + `
		GROUP BY 1
		ORDER BY 1 ASC
	`
	rows, err = s.pool.Query(ctx, dayQuery, args...)
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

	// (4) total revenue
	revenueWhere := where
	if revenueWhere == "" {
		revenueWhere = "WHERE e.event_name = 'Purchase' AND e.status = 'sent'"
	} else {
		revenueWhere += " AND e.event_name = 'Purchase' AND e.status = 'sent'"
	}
	revenueQuery := `
		SELECT COALESCE(SUM(max_value), 0)
		FROM (
			SELECT MAX(e.purchase_value) AS max_value
			FROM ` + fromTable + joinClause + ` ` + revenueWhere + `
			GROUP BY e.event_id, e.pixel_name
		) t
	`
	if err := s.pool.QueryRow(ctx, revenueQuery, args...).Scan(&out.TotalRevenue); err != nil {
		return nil, err
	}

	if err := s.populateTrackedUsersAndQuality(ctx, f, out); err != nil {
		return nil, err
	}
	_ = s.populateSourceBreakdown(ctx, f, out)

	return out, nil
}
