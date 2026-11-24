package repo

import (
	"context"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/meta"
)

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
		SELECT DISTINCT ON (se.pixel_name, se.event_id)
			se.event_id,
			CASE WHEN se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead') THEN 'Lead' ELSE se.event_name END,
			u.wa_ad_id, COALESCE(wc.source_id, ''), u.last_utm_content, u.first_utm_content,
			COALESCE(wc.ad_name, ''), COALESCE(wc.ad_thumbnail_url, ''),
			COALESCE(NULLIF(wc.source_app, ''), u.wa_source_app, '')
		FROM sent_events se
		JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id
		LEFT JOIN whatsapp_chats wc ON wc.tracked_user_id = u.id AND wc.account_id = u.account_id
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY se.pixel_name, se.event_id, (COALESCE(wc.source_id, '') <> '') DESC, wc.last_message_time DESC NULLS LAST, se.id DESC
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
		if c.Placement == "" && strings.EqualFold(srcApp, "instagram") {
			c.Placement = "Instagram"
		} else if c.Placement == "" && strings.EqualFold(srcApp, "facebook") {
			c.Placement = "Facebook"
		}
	}
}
