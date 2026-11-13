package repo

import (
	"context"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/meta"
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

const metaSourceSQL = `(
	u.fbc <> '' OR u.ctwa_clid <> ''
	OR u.last_utm_source ILIKE '%facebook%' OR u.last_utm_source ILIKE '%instagram%' OR u.last_utm_source ILIKE '%meta%'
	OR u.first_utm_source ILIKE '%facebook%' OR u.first_utm_source ILIKE '%instagram%' OR u.first_utm_source ILIKE '%meta%'
	OR LOWER(u.last_utm_source) IN ('fb', 'ig', 'an') OR LOWER(u.first_utm_source) IN ('fb', 'ig', 'an')
)`

const googleSourceSQL = `(
	u.gclid <> '' OR u.gbraid <> '' OR u.wbraid <> ''
	OR u.last_utm_source ILIKE '%google%' OR u.first_utm_source ILIKE '%google%'
	OR u.last_utm_source ILIKE '%youtube%' OR u.first_utm_source ILIKE '%youtube%'
	OR LOWER(u.last_utm_source) IN ('adwords', 'gads') OR LOWER(u.first_utm_source) IN ('adwords', 'gads')
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
	switch source {
	case "meta":
		return repl(metaSourceSQL)
	case "meta_lp":
		return repl(metaSourceSQL) + " AND (" + ua + "wa_chat_id = '' OR " + ua + "wa_chat_id IS NULL)"
	case "meta_wpp":
		return repl(metaSourceSQL) + " AND (" + ua + "wa_chat_id <> '' AND " + ua + "wa_chat_id IS NOT NULL)"
	case "google":
		return repl(googleSourceSQL)
	case "tiktok":
		return "(" + ua + "ttclid <> '' OR " + ua + "last_utm_source ILIKE '%tiktok%' OR " + ua + "first_utm_source ILIKE '%tiktok%')"
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
			COUNT(DISTINCT CASE WHEN e.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead') AND ` + metaSourceSQL + ` THEN e.event_id END) AS meta_leads,
			COUNT(DISTINCT CASE WHEN e.event_name = 'Purchase' AND ` + metaSourceSQL + ` THEN e.event_id END) AS meta_purchases,
			COALESCE(SUM(CASE WHEN e.event_name = 'Purchase' AND ` + metaSourceSQL + ` THEN e.max_val ELSE 0 END), 0) AS meta_revenue,
			COUNT(DISTINCT CASE WHEN e.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead') AND ` + googleExclusiveSourceSQL + ` THEN e.event_id END) AS google_leads,
			COUNT(DISTINCT CASE WHEN e.event_name = 'Purchase' AND ` + googleExclusiveSourceSQL + ` THEN e.event_id END) AS google_purchases,
			COALESCE(SUM(CASE WHEN e.event_name = 'Purchase' AND ` + googleExclusiveSourceSQL + ` THEN e.max_val ELSE 0 END), 0) AS google_revenue
		FROM (
			SELECT se.event_id, se.event_name, se.external_id, se.account_id, MAX(se.purchase_value) AS max_val
			FROM sent_events se
			WHERE ` + strings.Join(conds, " AND ") + `
			GROUP BY se.event_id, se.event_name, se.external_id, se.account_id
		) e
		LEFT JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id
	`
	if err := s.pool.QueryRow(ctx, q, args...).Scan(
		&out.SourceBreakdown.Meta.Leads,
		&out.SourceBreakdown.Meta.Purchases,
		&out.SourceBreakdown.Meta.Revenue,
		&out.SourceBreakdown.Google.Leads,
		&out.SourceBreakdown.Google.Purchases,
		&out.SourceBreakdown.Google.Revenue,
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

func (s *StatsRepo) EnrichCreativesWithTrackedConversions(ctx context.Context, f StatsFilter, creatives []meta.CreativeInsight) {
	if len(creatives) == 0 {
		return
	}
	args := []any{}
	conds := []string{
		"se.status = 'sent'",
		"se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase')",
	}
	idx := 0
	add := func(v any) string {
		args = append(args, v)
		idx++
		return "$" + itoa(idx)
	}
	if f.PixelName != "" {
		conds = append(conds, "se.pixel_name = "+add(f.PixelName))
	}
	if f.AccountID != 0 {
		conds = append(conds, "se.account_id = "+add(f.AccountID))
	}
	if !f.From.IsZero() {
		conds = append(conds, "se.created_at >= "+add(f.From))
	}
	if !f.To.IsZero() {
		conds = append(conds, "se.created_at < "+add(f.To))
	}

	q := `
		SELECT DISTINCT ON (se.event_id)
			se.event_id, se.event_name,
			u.wa_ad_id, COALESCE(wc.source_id, ''), u.last_utm_content, u.first_utm_content,
			COALESCE(wc.ad_name, ''), COALESCE(wc.ad_thumbnail_url, ''),
			COALESCE(NULLIF(wc.source_app, ''), u.wa_source_app, '')
		FROM sent_events se
		JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id
		LEFT JOIN whatsapp_chats wc ON wc.tracked_user_id = u.id AND wc.account_id = u.account_id
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY se.event_id
	`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	type convRow struct {
		eventID, eventName, waAdID, srcID, lastContent, firstContent, adName, thumb, srcApp string
	}
	var convs []convRow
	for rows.Next() {
		var r convRow
		if rows.Scan(&r.eventID, &r.eventName, &r.waAdID, &r.srcID, &r.lastContent, &r.firstContent, &r.adName, &r.thumb, &r.srcApp) == nil {
			convs = append(convs, r)
		}
	}
	if len(convs) == 0 {
		return
	}

	for i := range creatives {
		c := &creatives[i]
		adID := strings.TrimSpace(c.AdID)
		adName := strings.TrimSpace(c.AdName)
		if adID == "" && adName == "" {
			continue
		}
		var leads, purchases int64
		var thumb, srcApp string
		for _, r := range convs {
			matched := (adID != "" && (r.waAdID == adID || r.srcID == adID || r.lastContent == adID || r.firstContent == adID)) ||
				(adName != "" && (strings.EqualFold(r.adName, adName) || strings.EqualFold(r.lastContent, adName) || strings.EqualFold(r.firstContent, adName)))
			if !matched {
				continue
			}
			if r.eventName == "Purchase" {
				purchases++
			} else {
				leads++
			}
			if thumb == "" && r.thumb != "" {
				thumb = r.thumb
			}
			if srcApp == "" && r.srcApp != "" {
				srcApp = r.srcApp
			}
		}
		if leads > c.Leads {
			c.Leads = leads
		}
		if purchases > c.Purchases {
			c.Purchases = purchases
		}
		if c.ThumbnailURL == "" && c.ImageURL == "" && thumb != "" {
			c.ThumbnailURL = thumb
		}
		if c.Placement == "" && srcApp != "" {
			if strings.EqualFold(srcApp, "instagram") {
				c.Placement = "Instagram"
			} else if strings.EqualFold(srcApp, "facebook") {
				c.Placement = "Facebook"
			}
		}
	}
}
