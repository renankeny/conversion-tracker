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
	conds := []string{"se.event_name IN ('Lead', 'LeadSubmitted', 'Purchase')", "se.status = 'sent'"}
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
			COUNT(DISTINCT CASE WHEN e.event_name IN ('Lead', 'LeadSubmitted') AND ` + metaSourceSQL + ` THEN e.event_id END) AS meta_leads,
			COUNT(DISTINCT CASE WHEN e.event_name = 'Purchase' AND ` + metaSourceSQL + ` THEN e.event_id END) AS meta_purchases,
			COALESCE(SUM(CASE WHEN e.event_name = 'Purchase' AND ` + metaSourceSQL + ` THEN e.max_val ELSE 0 END), 0) AS meta_revenue,
			COUNT(DISTINCT CASE WHEN e.event_name IN ('Lead', 'LeadSubmitted') AND ` + googleSourceSQL + ` THEN e.event_id END) AS google_leads,
			COUNT(DISTINCT CASE WHEN e.event_name = 'Purchase' AND ` + googleSourceSQL + ` THEN e.event_id END) AS google_purchases,
			COALESCE(SUM(CASE WHEN e.event_name = 'Purchase' AND ` + googleSourceSQL + ` THEN e.max_val ELSE 0 END), 0) AS google_revenue
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

	// Visitantes por fonte no período
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
		vConds = append(vConds, "((u.last_seen_at >= "+vAdd(f.From)+" AND u.last_seen_at < "+vAdd(f.To)+") OR (u.created_at >= "+vAdd(f.From)+" AND u.created_at < "+vAdd(f.To)+"))")
	}
	vWhere := ""
	if len(vConds) > 0 {
		vWhere = "WHERE " + strings.Join(vConds, " AND ")
	}
	vq := `
		SELECT
			COUNT(*) FILTER (WHERE ` + metaSourceSQL + `),
			COUNT(*) FILTER (WHERE ` + googleSourceSQL + `)
		FROM tracked_users u ` + vWhere
	_ = s.pool.QueryRow(ctx, vq, vArgs...).Scan(&out.SourceBreakdown.Meta.Visitors, &out.SourceBreakdown.Google.Visitors)
	return nil
}

func (s *StatsRepo) EnrichCreativesWithTrackedConversions(ctx context.Context, f StatsFilter, creatives []meta.CreativeInsight) {
	if len(creatives) == 0 {
		return
	}
	for i := range creatives {
		c := &creatives[i]
		adID := strings.TrimSpace(c.AdID)
		adName := strings.TrimSpace(c.AdName)
		if adID == "" && adName == "" {
			continue
		}
		q := `
			SELECT
				COUNT(DISTINCT CASE WHEN se.event_name IN ('Lead', 'LeadSubmitted') THEN se.event_id END) AS leads,
				COUNT(DISTINCT CASE WHEN se.event_name = 'Purchase' THEN se.event_id END) AS purchases,
				COALESCE(MAX(NULLIF(wc.ad_thumbnail_url, '')), '') AS thumb,
				COALESCE(MAX(NULLIF(wc.source_app, '')), MAX(NULLIF(u.wa_source_app, '')), '') AS src_app
			FROM sent_events se
			JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id
			LEFT JOIN whatsapp_chats wc ON wc.tracked_user_id = u.id AND wc.account_id = u.account_id
			WHERE se.status = 'sent'
			  AND se.pixel_name = $1
			  AND ($2::bigint = 0 OR se.account_id = $2)
			  AND ($3::timestamptz IS NULL OR se.created_at >= $3)
			  AND ($4::timestamptz IS NULL OR se.created_at < $4)
			  AND (
				($5 <> '' AND (u.wa_ad_id = $5 OR wc.source_id = $5 OR u.last_utm_content = $5 OR u.first_utm_content = $5))
				OR ($6 <> '' AND (LOWER(wc.ad_name) = LOWER($6) OR LOWER(u.last_utm_content) = LOWER($6) OR LOWER(u.first_utm_content) = LOWER($6)))
			  )
		`
		var trackedLeads, trackedPurchases int64
		var thumb, srcApp string
		var fromPtr, toPtr any
		if !f.From.IsZero() {
			fromPtr = f.From
		}
		if !f.To.IsZero() {
			toPtr = f.To
		}
		if err := s.pool.QueryRow(ctx, q, f.PixelName, f.AccountID, fromPtr, toPtr, adID, adName).Scan(
			&trackedLeads, &trackedPurchases, &thumb, &srcApp,
		); err == nil {
			if trackedLeads > c.Leads {
				c.Leads = trackedLeads
			}
			if trackedPurchases > c.Purchases {
				c.Purchases = trackedPurchases
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
}

