package repo

import (
	"context"
	"strings"
	"time"
	_ "time/tzdata"
)

// MonthlyGoalMetrics traz o acumulado do mês de referência da Meta de Vendas (mês atual ou mês filtrado).
type MonthlyGoalMetrics struct {
	Month     string  `json:"month"`
	Revenue   float64 `json:"revenue"`
	Purchases int64   `json:"purchases"`
}

func resolveStatsGoalMonthWindow(f StatsFilter) (string, time.Time, time.Time) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil || loc == nil {
		loc = time.FixedZone("BRT", -3*3600)
	}
	if ym := strings.TrimSpace(f.GoalMonth); len(ym) == 7 {
		if t, perr := time.ParseInLocation("2006-01", ym, loc); perr == nil {
			start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
			return start.Format("2006-01"), start, start.AddDate(0, 1, 0)
		}
	}
	ref := time.Now().In(loc)
	if !f.To.IsZero() {
		toSP := f.To.In(loc)
		if toSP.Year() < ref.Year() || (toSP.Year() == ref.Year() && toSP.Month() < ref.Month()) {
			ref = toSP
		}
	}
	start := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, loc)
	return start.Format("2006-01"), start, start.AddDate(0, 1, 0)
}

func (s *StatsRepo) populateMonthlyGoalStats(ctx context.Context, f StatsFilter, dst *MonthlyGoalMetrics) {
	ym, mStart, mEnd := resolveStatsGoalMonthWindow(f)
	dst.Month = ym

	args := []any{mStart, mEnd}
	conds := []string{
		"e.status <> 'contested'",
		"e.event_name = 'Purchase'",
		"e.created_at >= $1",
		"e.created_at < $2",
	}
	idx := 2
	add := func(v any) string {
		args = append(args, v)
		idx++
		return "$" + itoa(idx)
	}
	if f.PixelName != "" {
		conds = append(conds, "e.pixel_name = "+add(f.PixelName))
	}
	if f.AccountID != 0 {
		conds = append(conds, "e.account_id = "+add(f.AccountID))
	}

	joinClause := ""
	if srcCond := trafficSourceUserCond(f.TrafficSource, "u."); srcCond != "" {
		joinClause = " JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id"
		conds = append(conds, srcCond)
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	q := `
		SELECT COALESCE(SUM(CASE WHEN has_sent THEN max_val ELSE 0 END), 0),
		       COUNT(*)
		FROM (
			SELECT MAX(e.purchase_value) AS max_val,
			       BOOL_OR(e.status = 'sent') AS has_sent
			FROM sent_events e` + joinClause + ` ` + where + `
			GROUP BY e.event_id, e.pixel_name
		) m
	`
	_ = s.pool.QueryRow(ctx, q, args...).Scan(&dst.Revenue, &dst.Purchases)

	// Abate reembolsos e chargebacks reais do mês da meta.
	var refCount int64
	var refAmt float64
	_ = s.pool.QueryRow(ctx, `
		SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(amount), 0)
		FROM sale_refunds
		WHERE ($1 = 0 OR account_id = $1)
		  AND ($2 = '' OR pixel_name = $2)
		  AND refunded_at >= $3 AND refunded_at < $4
	`, f.AccountID, f.PixelName, mStart, mEnd).Scan(&refCount, &refAmt)
	if dst.Revenue -= refAmt; dst.Revenue < 0 {
		dst.Revenue = 0
	}
	if dst.Purchases -= refCount; dst.Purchases < 0 {
		dst.Purchases = 0
	}
}

func (s *StatsRepo) populatePeriodRefunds(ctx context.Context, f StatsFilter, out *Overview) {
	if out == nil {
		return
	}
	args := []any{}
	conds := []string{"1=1"}
	idx := 0
	add := func(v any) string {
		args = append(args, v)
		idx++
		return "$" + itoa(idx)
	}
	if !f.From.IsZero() {
		conds = append(conds, "refunded_at >= "+add(f.From))
	}
	if !f.To.IsZero() {
		conds = append(conds, "refunded_at < "+add(f.To))
	}
	if f.PixelName != "" {
		conds = append(conds, "pixel_name = "+add(f.PixelName))
	}
	if f.AccountID != 0 {
		conds = append(conds, "account_id = "+add(f.AccountID))
	}
	q := `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(amount), 0) FROM sale_refunds WHERE ` + strings.Join(conds, " AND ")
	_ = s.pool.QueryRow(ctx, q, args...).Scan(&out.RefundCount, &out.RefundAmount)
	out.NetRevenue = out.TotalRevenue - out.RefundAmount
	if out.NetRevenue < 0 {
		out.NetRevenue = 0
	}
}
