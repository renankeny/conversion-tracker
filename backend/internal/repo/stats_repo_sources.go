package repo

import (
	"context"
	"strings"
)

type SourceMetrics struct {
	Visitors  int64   `json:"visitors"`
	Leads     int64   `json:"leads"`
	Purchases int64   `json:"purchases"`
	Revenue   float64 `json:"revenue"`
}

type SourceBreakdown struct {
	Meta   SourceMetrics `json:"meta"`
	Google SourceMetrics `json:"google"`
}

const (
	organicUTMMediumListSQL = `('organic', 'organico', 'orgânico', 'social', 'social-network', 'social_organic', 'referral', 'referencia', 'bio', 'link_rastreavel', 'linktree', 'perfil', 'qr', 'qrcode', 'direct', 'direto', 'none', '(none)', 'email', 'newsletter', 'seo')`
	paidUTMMediumRegexSQL   = `^(cpc|ppc|paid|ads|ad|cpm|cpa|paid_social|paidsocial|paid-social|social_paid|pmax|display|remarketing|retargeting|ctwa|sponsored|impulsionar|impulsionamento|trafego_pago|trafego-pago|paid_.*|.*_ads|.*pago.*)$`

	effectiveUTMSourceSQL = `LOWER(TRIM(COALESCE(NULLIF(u.last_utm_source, ''), u.first_utm_source, '')))`
	effectiveUTMMediumSQL = `LOWER(TRIM(COALESCE(NULLIF(u.last_utm_medium, ''), u.first_utm_medium, '')))`
	utmEligibleForPaidSQL = `(` + effectiveUTMMediumSQL + ` NOT IN ` + organicUTMMediumListSQL + ` AND (u.trackable_link_id IS NULL OR ` + effectiveUTMMediumSQL + ` ~ '` + paidUTMMediumRegexSQL + `'))`
)

const metaSourceSQL = `(
	u.fbc <> '' OR u.ctwa_clid <> '' OR u.wa_ad_id <> ''
	OR ` + effectiveUTMSourceSQL + ` IN ('meta_ads', 'facebook_ads', 'fb_ads', 'ig_ads', 'instagram_ads', 'an')
	OR (
		` + utmEligibleForPaidSQL + `
		AND (
			` + effectiveUTMSourceSQL + ` IN ('fb', 'ig', 'meta', 'facebook', 'instagram')
			OR ` + effectiveUTMSourceSQL + ` LIKE '%facebook%'
			OR ` + effectiveUTMSourceSQL + ` LIKE '%instagram%'
			OR ` + effectiveUTMSourceSQL + ` LIKE '%meta%'
		)
	)
)`

const googleSourceSQL = `(
	u.gclid <> '' OR u.gbraid <> '' OR u.wbraid <> ''
	OR ` + effectiveUTMSourceSQL + ` IN ('adwords', 'gads', 'google_ads', 'googleads')
	OR (
		` + utmEligibleForPaidSQL + `
		AND (
			` + effectiveUTMSourceSQL + ` IN ('google', 'youtube')
			OR ` + effectiveUTMSourceSQL + ` LIKE '%google%'
			OR ` + effectiveUTMSourceSQL + ` LIKE '%youtube%'
		)
	)
)`

const tiktokSourceSQL = `(
	u.ttclid <> ''
	OR ` + effectiveUTMSourceSQL + ` = 'tiktok_ads'
	OR (
		` + utmEligibleForPaidSQL + `
		AND (
			` + effectiveUTMSourceSQL + ` IN ('tiktok', 'tt')
			OR ` + effectiveUTMSourceSQL + ` LIKE '%tiktok%'
			OR ` + effectiveUTMSourceSQL + ` LIKE '%bytedance%'
		)
	)
)`

const googleExclusiveSourceSQL = `(NOT ` + metaSourceSQL + ` AND ` + googleSourceSQL + `)`

func trafficSourceUserCond(source, userAlias string) string {
	ua := strings.TrimSpace(userAlias)
	if ua != "" && !strings.HasSuffix(ua, ".") {
		ua += "."
	}
	repl := func(sql string) string {
		if ua == "u." {
			return sql
		}
		return strings.ReplaceAll(sql, "u.", ua)
	}
	wppCond := "(" + ua + "ctwa_clid <> '' OR " + ua + "wa_ad_id <> '' OR " + ua + "external_id LIKE 'wpp:%' OR EXISTS (SELECT 1 FROM whatsapp_chats wc WHERE wc.tracked_user_id = " + ua + "id AND wc.account_id = " + ua + "account_id))"
	switch source {
	case "meta":
		return repl(metaSourceSQL)
	case "meta_lp":
		return repl(metaSourceSQL) + " AND NOT " + wppCond
	case "meta_wpp":
		return repl(metaSourceSQL) + " AND " + wppCond
	case "google":
		return repl(googleSourceSQL)
	case "tiktok":
		return repl(tiktokSourceSQL)
	default:
		return ""
	}
}

func (s *StatsRepo) populateSourceBreakdown(ctx context.Context, f StatsFilter, out *Overview) error {
	args := []any{}
	conds := []string{"se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase')", "se.status = 'sent'"}
	idx := 0
	add := func(v any) string {
		args = append(args, v)
		idx++
		return "$" + itoa(idx)
	}
	if !f.From.IsZero() {
		conds = append(conds, "se.created_at >= "+add(f.From))
	}
	if !f.To.IsZero() {
		conds = append(conds, "se.created_at < "+add(f.To))
	}
	if f.PixelName != "" {
		conds = append(conds, "se.pixel_name = "+add(f.PixelName))
	}
	if f.AccountID != 0 {
		conds = append(conds, "se.account_id = "+add(f.AccountID))
	}

	q := `
		SELECT
			COUNT(DISTINCT CASE WHEN e.norm_event = 'Lead' AND ` + metaSourceSQL + ` THEN e.pixel_name || ':' || e.event_id END) AS meta_leads,
			COUNT(DISTINCT CASE WHEN e.norm_event = 'Purchase' AND ` + metaSourceSQL + ` THEN e.pixel_name || ':' || e.event_id END) AS meta_purchases,
			COALESCE(SUM(CASE WHEN e.norm_event = 'Purchase' AND ` + metaSourceSQL + ` THEN e.max_val ELSE 0 END), 0) AS meta_revenue,
			COUNT(DISTINCT CASE WHEN e.norm_event = 'Lead' AND ` + googleExclusiveSourceSQL + ` THEN e.pixel_name || ':' || e.event_id END) AS google_leads,
			COUNT(DISTINCT CASE WHEN e.norm_event = 'Purchase' AND ` + googleExclusiveSourceSQL + ` THEN e.pixel_name || ':' || e.event_id END) AS google_purchases,
			COALESCE(SUM(CASE WHEN e.norm_event = 'Purchase' AND ` + googleExclusiveSourceSQL + ` THEN e.max_val ELSE 0 END), 0) AS google_revenue
		FROM (
			SELECT se.pixel_name, se.event_id,
			       CASE WHEN se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead') THEN 'Lead' ELSE se.event_name END AS norm_event,
			       se.external_id, se.account_id, MAX(se.purchase_value) AS max_val
			FROM sent_events se
			WHERE ` + strings.Join(conds, " AND ") + `
			GROUP BY se.pixel_name, se.event_id, norm_event, se.external_id, se.account_id
		) e
		LEFT JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id
	`
	if err := s.pool.QueryRow(ctx, q, args...).Scan(
		&out.SourceBreakdown.Meta.Leads, &out.SourceBreakdown.Meta.Purchases, &out.SourceBreakdown.Meta.Revenue,
		&out.SourceBreakdown.Google.Leads, &out.SourceBreakdown.Google.Purchases, &out.SourceBreakdown.Google.Revenue,
	); err != nil {
		return err
	}

	vArgs := []any{}
	vConds := []string{}
	vIdx := 0
	vAdd := func(v any) string {
		vArgs = append(vArgs, v)
		vIdx++
		return "$" + itoa(vIdx)
	}
	if f.PixelName != "" {
		vConds = append(vConds, "u.pixel_name = "+vAdd(f.PixelName))
	}
	if f.AccountID != 0 {
		vConds = append(vConds, "u.account_id = "+vAdd(f.AccountID))
	}
	if !f.From.IsZero() && !f.To.IsZero() {
		fromP := vAdd(f.From)
		toP := vAdd(f.To)
		vConds = append(vConds, "((u.last_seen_at >= "+fromP+" AND u.last_seen_at < "+toP+") OR (u.created_at >= "+fromP+" AND u.created_at < "+toP+"))")
	}
	vWhere := ""
	if len(vConds) > 0 {
		vWhere = "WHERE " + strings.Join(vConds, " AND ")
	}
	vq := `
		SELECT
			COUNT(*) FILTER (WHERE ` + metaSourceSQL + `),
			COUNT(*) FILTER (WHERE ` + googleExclusiveSourceSQL + `)
		FROM tracked_users u ` + vWhere
	_ = s.pool.QueryRow(ctx, vq, vArgs...).Scan(&out.SourceBreakdown.Meta.Visitors, &out.SourceBreakdown.Google.Visitors)
	return nil
}
