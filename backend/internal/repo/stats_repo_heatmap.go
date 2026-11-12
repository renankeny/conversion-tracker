package repo

import (
	"context"
	"strings"
)

type PreviousPeriodMetrics struct {
	PageViews  int64   `json:"page_views"`
	AdMessages int64   `json:"ad_messages"`
	Leads      int64   `json:"leads"`
	Purchases  int64   `json:"purchases"`
	Revenue    float64 `json:"revenue"`
}

type DashboardHeatmap struct {
	Messages []HeatmapBucket `json:"messages"`
	Leads    []HeatmapBucket `json:"leads"`
	Sales    []HeatmapBucket `json:"sales"`
}

func (s *StatsRepo) populatePreviousAndHeatmap(ctx context.Context, f StatsFilter, out *Overview) {
	if !f.From.IsZero() && !f.To.IsZero() && f.To.After(f.From) {
		dur := f.To.Sub(f.From)
		prevFilter := f
		prevFilter.From = f.From.Add(-dur)
		prevFilter.To = f.From
		s.populatePreviousPeriod(ctx, prevFilter, &out.PreviousPeriod)
	}
	s.populateDashboardHeatmap(ctx, f, &out.Heatmap)
}

func (s *StatsRepo) populatePreviousPeriod(ctx context.Context, pf StatsFilter, dst *PreviousPeriodMetrics) {
	var msgAudit, leadAudit, purAudit AuditCounter
	s.populateAdMessagesAudit(ctx, pf, &msgAudit)
	s.populateLeadsAudit(ctx, pf, &leadAudit)
	s.populatePurchasesAudit(ctx, pf, &purAudit)
	dst.AdMessages = msgAudit.Total
	dst.Leads = leadAudit.Total
	dst.Purchases = purAudit.Total

	srcCond := strings.TrimSpace(trafficSourceUserCond(pf.TrafficSource, "u."))
	if srcCond == "" {
		_ = s.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(count_total), 0)
			FROM event_counters
			WHERE event_name = 'PageView'
			  AND ($1::bigint = 0 OR account_id = $1)
			  AND ($2::text = '' OR pixel_name = $2)
			  AND bucket_at >= $3 AND bucket_at < $4
		`, pf.AccountID, pf.PixelName, pf.From, pf.To).Scan(&dst.PageViews)
	} else {
		_ = s.pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM tracked_users u
			WHERE ($1::bigint = 0 OR u.account_id = $1)
			  AND ($2::text = '' OR u.pixel_name = $2)
			  AND ((u.last_seen_at >= $3 AND u.last_seen_at < $4) OR (u.created_at >= $3 AND u.created_at < $4))
			  AND `+srcCond,
			pf.AccountID, pf.PixelName, pf.From, pf.To,
		).Scan(&dst.PageViews)
	}

	revJoin := ""
	revExtra := ""
	if srcCond != "" {
		revJoin = " JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id "
		revExtra = " AND " + srcCond
	}
	_ = s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(max_val), 0)
		FROM (
			SELECT MAX(e.purchase_value) AS max_val
			FROM sent_events e `+revJoin+`
			WHERE ($1::bigint = 0 OR e.account_id = $1)
			  AND ($2::text = '' OR e.pixel_name = $2)
			  AND e.event_name = 'Purchase'
			  AND e.status = 'sent'
			  AND e.created_at >= $3 AND e.created_at < $4`+revExtra+`
			GROUP BY e.event_id, e.pixel_name
		) t
	`, pf.AccountID, pf.PixelName, pf.From, pf.To).Scan(&dst.Revenue)
}

func (s *StatsRepo) populateDashboardHeatmap(ctx context.Context, f StatsFilter, dst *DashboardHeatmap) {
	dst.Messages = []HeatmapBucket{}
	dst.Leads = []HeatmapBucket{}
	dst.Sales = []HeatmapBucket{}
	if f.From.IsZero() || f.To.IsZero() {
		return
	}

	srcCond := strings.TrimSpace(trafficSourceUserCond(f.TrafficSource, "u."))
	msgSrcCond := ""
	switch f.TrafficSource {
	case "meta", "meta_lp", "meta_wpp":
		msgSrcCond = " AND (c.ctwa_clid <> '' OR c.source_id <> '' OR c.origin_type IN ('meta_ads_ctwa', 'meta_ads_lp') OR (u.id IS NOT NULL AND " + metaSourceSQL + "))"
	case "google":
		msgSrcCond = " AND (c.origin_type = 'google_ads' OR (u.id IS NOT NULL AND " + googleSourceSQL + "))"
	}

	msgQ := `
		SELECT EXTRACT(DOW FROM m.timestamp AT TIME ZONE 'America/Sao_Paulo')::int,
		       EXTRACT(HOUR FROM m.timestamp AT TIME ZONE 'America/Sao_Paulo')::int,
		       COUNT(*)
		FROM whatsapp_messages m
		JOIN whatsapp_chats c ON m.chat_id = c.id
		JOIN whatsapp_instances i ON c.instance_id = i.id
		JOIN pixels p ON i.pixel_id = p.id
		LEFT JOIN tracked_users u ON c.tracked_user_id = u.id
		WHERE m.is_from_me = FALSE
		  AND ($1::bigint = 0 OR i.account_id = $1)
		  AND ($2::text = '' OR p.name = $2)
		  AND m.timestamp >= $3 AND m.timestamp < $4` + msgSrcCond + `
		GROUP BY 1, 2
	`
	dst.Messages = s.scanHeatmapQuery(ctx, msgQ, f.AccountID, f.PixelName, f.From, f.To)

	evJoin := ""
	evExtra := ""
	if srcCond != "" {
		evJoin = " JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id "
		evExtra = " AND " + srcCond
	}

	leadQ := `
		SELECT dow, hr, COUNT(*) FROM (
			SELECT DISTINCT se.event_id,
			       EXTRACT(DOW FROM se.created_at AT TIME ZONE 'America/Sao_Paulo')::int AS dow,
			       EXTRACT(HOUR FROM se.created_at AT TIME ZONE 'America/Sao_Paulo')::int AS hr
			FROM sent_events se` + evJoin + `
			WHERE ($1::bigint = 0 OR se.account_id = $1)
			  AND ($2::text = '' OR se.pixel_name = $2)
			  AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead')
			  AND se.created_at >= $3 AND se.created_at < $4` + evExtra + `
		) l GROUP BY 1, 2
	`
	dst.Leads = s.scanHeatmapQuery(ctx, leadQ, f.AccountID, f.PixelName, f.From, f.To)

	salesQ := `
		SELECT dow, hr, COUNT(*) FROM (
			SELECT DISTINCT se.event_id,
			       EXTRACT(DOW FROM se.created_at AT TIME ZONE 'America/Sao_Paulo')::int AS dow,
			       EXTRACT(HOUR FROM se.created_at AT TIME ZONE 'America/Sao_Paulo')::int AS hr
			FROM sent_events se` + evJoin + `
			WHERE ($1::bigint = 0 OR se.account_id = $1)
			  AND ($2::text = '' OR se.pixel_name = $2)
			  AND se.event_name = 'Purchase'
			  AND se.status = 'sent'
			  AND se.created_at >= $3 AND se.created_at < $4` + evExtra + `
		) p GROUP BY 1, 2
	`
	dst.Sales = s.scanHeatmapQuery(ctx, salesQ, f.AccountID, f.PixelName, f.From, f.To)
}

func (s *StatsRepo) scanHeatmapQuery(ctx context.Context, q string, args ...any) []HeatmapBucket {
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return []HeatmapBucket{}
	}
	defer rows.Close()
	out := []HeatmapBucket{}
	for rows.Next() {
		var b HeatmapBucket
		if rows.Scan(&b.Weekday, &b.Hour, &b.Count) == nil {
			out = append(out, b)
		}
	}
	return out
}

