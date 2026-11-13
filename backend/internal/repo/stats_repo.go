package repo

import (
	"context"
	"strings"
	"sync"
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
	TrafficSource string    // all, meta, meta_lp, meta_wpp, google, tiktok
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

// Overview agrupa as séries e agregados principais da Dashboard.
type Overview struct {
	StatusTotals   map[string]int64      `json:"status_totals"`
	EventTotals    map[string]int64      `json:"event_totals"`
	PerDay         []DailyBucket         `json:"per_day"`
	TotalEvents    int64                 `json:"total_events"`
	TotalRevenue   float64               `json:"total_revenue"`
	TrackedUsers   int64                 `json:"tracked_users"`
	TrafficQuality TrafficQualityStats   `json:"traffic_quality"`
	SourceBreakdown SourceBreakdown      `json:"source_breakdown"`
	PanelAudit     PanelAudit            `json:"panel_audit"`
	PreviousPeriod PreviousPeriodMetrics `json:"previous_period"`
	Heatmap        DashboardHeatmap      `json:"heatmap"`
}

// DailyBucket é uma linha da série temporal por dia.
type DailyBucket struct {
	Date   string `json:"date"`
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

// Overview executa as agregações da Dashboard em paralelo usando o pool pgx.
func (s *StatsRepo) Overview(ctx context.Context, f StatsFilter) (*Overview, error) {
	out := &Overview{
		StatusTotals: map[string]int64{},
		EventTotals:  map[string]int64{},
		PerDay:       []DailyBucket{},
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	recordErr := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}

	wg.Add(5)
	go func() {
		defer wg.Done()
		recordErr(s.populateBaseEventStats(ctx, f, out))
	}()
	go func() {
		defer wg.Done()
		recordErr(s.populateTrackedUsersAndQuality(ctx, f, out))
	}()
	go func() {
		defer wg.Done()
		_ = s.populateSourceBreakdown(ctx, f, out)
	}()
	go func() {
		defer wg.Done()
		s.populatePanelAudit(ctx, f, out)
	}()
	go func() {
		defer wg.Done()
		s.populatePreviousAndHeatmap(ctx, f, out)
	}()
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func (s *StatsRepo) populateBaseEventStats(ctx context.Context, f StatsFilter, out *Overview) error {
	args := []any{}
	conds := []string{"e.status <> 'contested'"}
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
	where := "WHERE " + strings.Join(conds, " AND ")

	statusQuery := "SELECT e.status, COUNT(DISTINCT e.event_id) FROM " + fromTable + joinClause + " " + where + " GROUP BY e.status"
	rows, err := s.pool.Query(ctx, statusQuery, args...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var k string
		var n int64
		if err := rows.Scan(&k, &n); err != nil {
			rows.Close()
			return err
		}
		out.StatusTotals[k] = n
	}
	rows.Close()

	eventQuery := "SELECT e.event_name, COUNT(DISTINCT e.event_id) FROM " + fromTable + joinClause + " " + where + " GROUP BY e.event_name ORDER BY 2 DESC"
	rows, err = s.pool.Query(ctx, eventQuery, args...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var k string
		var n int64
		if err := rows.Scan(&k, &n); err != nil {
			rows.Close()
			return err
		}
		out.EventTotals[k] = n
		out.TotalEvents += n
	}
	rows.Close()

	dayQuery := `
		SELECT to_char(date_trunc('day', e.created_at AT TIME ZONE 'America/Sao_Paulo'), 'YYYY-MM-DD') AS day,
		       COUNT(DISTINCT CASE WHEN e.status='sent'   THEN e.event_id END) AS sent,
		       COUNT(DISTINCT CASE WHEN e.status='failed' THEN e.event_id END) AS failed,
		       COUNT(DISTINCT CASE WHEN e.status='queued' THEN e.event_id END) AS queued
		FROM ` + fromTable + joinClause + ` ` + where + `
		GROUP BY 1
		ORDER BY 1 ASC
	`
	rows, err = s.pool.Query(ctx, dayQuery, args...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var b DailyBucket
		if err := rows.Scan(&b.Date, &b.Sent, &b.Failed, &b.Queued); err != nil {
			rows.Close()
			return err
		}
		out.PerDay = append(out.PerDay, b)
	}
	rows.Close()

	revenueQuery := `
		SELECT COALESCE(SUM(max_value), 0)
		FROM (
			SELECT MAX(e.purchase_value) AS max_value
			FROM ` + fromTable + joinClause + ` ` + where + ` AND e.event_name = 'Purchase' AND e.status = 'sent'
			GROUP BY e.event_id, e.pixel_name
		) t
	`
	return s.pool.QueryRow(ctx, revenueQuery, args...).Scan(&out.TotalRevenue)
}
