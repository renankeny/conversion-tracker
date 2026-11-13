package repo

import (
	"context"
	"strings"
	"time"
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
		now := time.Now()
		effectiveTo := f.To
		if effectiveTo.After(now) && now.After(f.From) {
			effectiveTo = now
		}
		dur := f.To.Sub(f.From)
		prevFilter := f
		prevFilter.From = f.From.Add(-dur)
		prevFilter.To = prevFilter.From.Add(effectiveTo.Sub(f.From))
		s.populatePreviousPeriod(ctx, prevFilter, &out.PreviousPeriod)
	}
	s.populateDashboardHeatmap(ctx, f, &out.Heatmap)
}

func (s *StatsRepo) populatePreviousPeriod(ctx context.Context, pf StatsFilter, dst *PreviousPeriodMetrics) {
	var msgAudit, leadAudit AuditCounter
	s.populateAdMessagesAudit(ctx, pf, &msgAudit)
	s.populateLeadsAudit(ctx, pf, &leadAudit)
	dst.AdMessages = msgAudit.Total
	dst.Leads = leadAudit.Total

	srcCond := strings.TrimSpace(trafficSourceUserCond(pf.TrafficSource, "u."))
	if srcCond == "" {
		ecScope, ecArgs := auditScopeConds("account_id", "pixel_name", pf, 2)
		args := append([]any{pf.From.UTC().Truncate(time.Hour), pf.To.UTC()}, ecArgs...)
		_ = s.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(count_total), 0)
			FROM event_counters
			WHERE event_name = 'PageView'
			  AND bucket_at >= $1 AND bucket_at < $2`+ecScope,
			args...,
		).Scan(&dst.PageViews)
	} else {
		tuScope, tuArgs := auditScopeConds("u.account_id", "u.pixel_name", pf, 2)
		args := append([]any{pf.From.UTC(), pf.To.UTC()}, tuArgs...)
		_ = s.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(CASE WHEN u.created_at >= $1 THEN GREATEST(u.pageview_count, 1) ELSE 1 END), 0)
			FROM tracked_users u
			WHERE ((u.last_seen_at >= $1 AND u.last_seen_at < $2) OR (u.created_at >= $1 AND u.created_at < $2))`+tuScope+`
			  AND `+srcCond,
			args...,
		).Scan(&dst.PageViews)
	}

	revJoin := ""
	revExtra := ""
	if srcCond != "" {
		revJoin = " JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id "
		revExtra = " AND " + srcCond
	}
	seScope, seArgs := auditScopeConds("e.account_id", "e.pixel_name", pf, 2)
	purArgs := append([]any{pf.From, pf.To}, seArgs...)
	_ = s.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(max_val), 0)
		FROM (
			SELECT MAX(e.purchase_value) AS max_val
			FROM sent_events e `+revJoin+`
			WHERE e.event_name = 'Purchase'
			  AND e.status = 'sent'
			  AND e.created_at >= $1 AND e.created_at < $2`+seScope+revExtra+`
			GROUP BY e.event_id, e.pixel_name
		) t
	`, purArgs...).Scan(&dst.Purchases, &dst.Revenue)
}

func (s *StatsRepo) populateDashboardHeatmap(ctx context.Context, f StatsFilter, dst *DashboardHeatmap) {
	dst.Messages = []HeatmapBucket{}
	dst.Leads = []HeatmapBucket{}
	dst.Sales = []HeatmapBucket{}
	if f.From.IsZero() || f.To.IsZero() {
		return
	}

	srcCond := strings.TrimSpace(trafficSourceUserCond(f.TrafficSource, "u."))
	msgSrcCond := " AND (" + waChatIsMetaSQL + " OR " + waChatIsGoogleSQL + " OR " + waChatIsTikTokSQL + ")"
	switch f.TrafficSource {
	case "meta", "meta_lp", "meta_wpp":
		msgSrcCond = " AND " + waChatIsMetaSQL
	case "google":
		msgSrcCond = " AND (NOT " + waChatIsMetaSQL + " AND " + waChatIsGoogleSQL + ")"
	}

	msgScope, msgScopeArgs := auditScopeConds("i.account_id", "p.name", f, 2)
	msgArgs := append([]any{f.From, f.To}, msgScopeArgs...)
	entryTS := waChatEntryTimeSQL("$1", "$2")
	msgQ := `
		SELECT EXTRACT(DOW FROM entry_ts AT TIME ZONE 'America/Sao_Paulo')::int,
		       EXTRACT(HOUR FROM entry_ts AT TIME ZONE 'America/Sao_Paulo')::int,
		       COUNT(*)
		FROM (
			SELECT ` + entryTS + ` AS entry_ts
			FROM whatsapp_chats c
			JOIN whatsapp_instances i ON c.instance_id = i.id
			JOIN pixels p ON i.pixel_id = p.id
			LEFT JOIN tracked_users u ON c.tracked_user_id = u.id
			WHERE ` + waChatPeriodEntrySQL("$1", "$2") + msgScope + msgSrcCond + `
		) entries
		GROUP BY 1, 2
	`
	dst.Messages = s.scanHeatmapQuery(ctx, msgQ, msgArgs...)

	evJoin := ""
	evExtra := ""
	if srcCond != "" {
		evJoin = " JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id "
		evExtra = " AND " + srcCond
	}
	seScope, seScopeArgs := auditScopeConds("se.account_id", "se.pixel_name", f, 2)
	seArgs := append([]any{f.From, f.To}, seScopeArgs...)

	leadQ := `
		SELECT dow, hr, COUNT(*) FROM (
			SELECT DISTINCT se.event_id,
			       EXTRACT(DOW FROM se.created_at AT TIME ZONE 'America/Sao_Paulo')::int AS dow,
			       EXTRACT(HOUR FROM se.created_at AT TIME ZONE 'America/Sao_Paulo')::int AS hr
			FROM sent_events se` + evJoin + `
			WHERE se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead')
			  AND se.status = 'sent'
			  AND se.created_at >= $1 AND se.created_at < $2` + seScope + evExtra + `
		) l GROUP BY 1, 2
	`
	dst.Leads = s.scanHeatmapQuery(ctx, leadQ, seArgs...)

	salesQ := `
		SELECT dow, hr, COUNT(*) FROM (
			SELECT DISTINCT se.event_id,
			       EXTRACT(DOW FROM se.created_at AT TIME ZONE 'America/Sao_Paulo')::int AS dow,
			       EXTRACT(HOUR FROM se.created_at AT TIME ZONE 'America/Sao_Paulo')::int AS hr
			FROM sent_events se` + evJoin + `
			WHERE se.event_name = 'Purchase'
			  AND se.status = 'sent'
			  AND se.created_at >= $1 AND se.created_at < $2` + seScope + evExtra + `
		) p GROUP BY 1, 2
	`
	dst.Sales = s.scanHeatmapQuery(ctx, salesQ, seArgs...)
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
