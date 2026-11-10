package repo

import (
	"context"
	"strings"
	"time"
)

type WhatsAppStatsFilter struct {
	From      time.Time
	To        time.Time
	AccountID int64
	PixelName string
}

type WhatsAppOverview struct {
	Conversations  int64            `json:"conversations"`
	Leads          int64            `json:"leads"`
	Sales          int64            `json:"sales"`
	Revenue        float64          `json:"revenue"`
	AvgTicket      float64          `json:"avg_ticket"`
	ConversionRate float64          `json:"conversion_rate"`
	BySource       map[string]int64 `json:"by_source"`
}

func attributionSource(originType, ctwaClid, fbc, gclid, gbraid, wbraid, ttclid string) string {
	if originType != "" && originType != "unknown" {
		return originType
	}
	switch {
	case ctwaClid != "":
		return "meta_ads_ctwa"
	case fbc != "":
		return "meta_ads_lp"
	case gclid != "" || gbraid != "" || wbraid != "":
		return "google_ads"
	case ttclid != "":
		return "tiktok_ads"
	default:
		return "website_organic"
	}
}

func buildWhatsAppStatsConds(f WhatsAppStatsFilter, timeCol string, extraConds ...string) ([]string, []any) {
	args := []any{f.AccountID}
	conds := append([]string{"i.account_id = $1"}, extraConds...)
	idx := 1
	if f.PixelName != "" {
		idx++
		conds = append(conds, "p.name = $"+itoa(idx))
		args = append(args, f.PixelName)
	}
	if !f.From.IsZero() {
		idx++
		conds = append(conds, timeCol+" >= $"+itoa(idx))
		args = append(args, f.From)
	}
	if !f.To.IsZero() {
		idx++
		conds = append(conds, timeCol+" < $"+itoa(idx))
		args = append(args, f.To)
	}
	return conds, args
}

func (r *WhatsAppRepo) Overview(ctx context.Context, f WhatsAppStatsFilter) (*WhatsAppOverview, error) {
	out := &WhatsAppOverview{}

	{
		conds, args := buildWhatsAppStatsConds(f, "m.timestamp")
		q := `
			SELECT COUNT(DISTINCT m.chat_id)
			FROM whatsapp_messages m
			JOIN whatsapp_chats c ON m.chat_id = c.id
			JOIN whatsapp_instances i ON c.instance_id = i.id
			JOIN pixels p ON i.pixel_id = p.id
			WHERE ` + strings.Join(conds, " AND ")
		if err := r.db.QueryRow(ctx, q, args...).Scan(&out.Conversations); err != nil {
			return nil, err
		}
	}

	{
		args := []any{f.AccountID}
		conds := []string{
			"e.account_id = $1",
			"(e.payload->'data'->0->>'action_source' = 'business_messaging' OR e.payload->'data'->0->>'messaging_channel' = 'whatsapp')",
		}
		idx := 1
		if f.PixelName != "" {
			idx++
			conds = append(conds, "e.pixel_name = $"+itoa(idx))
			args = append(args, f.PixelName)
		}
		if !f.From.IsZero() {
			idx++
			conds = append(conds, "e.created_at >= $"+itoa(idx))
			args = append(args, f.From)
		}
		if !f.To.IsZero() {
			idx++
			conds = append(conds, "e.created_at < $"+itoa(idx))
			args = append(args, f.To)
		}
		q := `
			SELECT
				COUNT(DISTINCT e.event_id) FILTER (WHERE e.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead')),
				COUNT(DISTINCT e.event_id) FILTER (WHERE e.event_name = 'Purchase' AND e.status = 'sent'),
				COALESCE(SUM(e.purchase_value) FILTER (WHERE e.event_name = 'Purchase' AND e.status = 'sent'), 0)
			FROM sent_events e
			WHERE ` + strings.Join(conds, " AND ")
		if err := r.db.QueryRow(ctx, q, args...).Scan(&out.Leads, &out.Sales, &out.Revenue); err != nil {
			return nil, err
		}
	}

	bySource, err := r.LeadsBySource(ctx, f)
	if err != nil {
		return nil, err
	}
	out.BySource = bySource
	if out.Sales > 0 {
		out.AvgTicket = out.Revenue / float64(out.Sales)
	}
	if out.Leads > 0 {
		out.ConversionRate = float64(out.Sales) / float64(out.Leads) * 100
	}
	return out, nil
}

type HeatmapBucket struct {
	Weekday int   `json:"weekday"`
	Hour    int   `json:"hour"`
	Count   int64 `json:"count"`
}

func (r *WhatsAppRepo) Heatmap(ctx context.Context, f WhatsAppStatsFilter) ([]HeatmapBucket, error) {
	conds, args := buildWhatsAppStatsConds(f, "m.timestamp", "m.is_from_me = FALSE")
	q := `
		SELECT EXTRACT(DOW FROM m.timestamp)::int AS weekday,
		       EXTRACT(HOUR FROM m.timestamp)::int AS hour,
		       COUNT(*)
		FROM whatsapp_messages m
		JOIN whatsapp_chats c ON m.chat_id = c.id
		JOIN whatsapp_instances i ON c.instance_id = i.id
		JOIN pixels p ON i.pixel_id = p.id
		WHERE ` + strings.Join(conds, " AND ") + `
		GROUP BY 1, 2
	`
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HeatmapBucket
	for rows.Next() {
		var b HeatmapBucket
		if err := rows.Scan(&b.Weekday, &b.Hour, &b.Count); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *WhatsAppRepo) LeadsBySource(ctx context.Context, f WhatsAppStatsFilter) (map[string]int64, error) {
	conds, args := buildWhatsAppStatsConds(f, "m.timestamp")
	q := `
		SELECT DISTINCT ON (m.chat_id) c.origin_type, u.ctwa_clid, u.fbc, u.gclid, u.gbraid, u.wbraid, u.ttclid
		FROM whatsapp_messages m
		JOIN whatsapp_chats c ON m.chat_id = c.id
		JOIN whatsapp_instances i ON c.instance_id = i.id
		JOIN tracked_users u ON c.tracked_user_id = u.id
		JOIN pixels p ON i.pixel_id = p.id
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY m.chat_id
	`
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var originType, ctwaClid, fbc, gclid, gbraid, wbraid, ttclid string
		if err := rows.Scan(&originType, &ctwaClid, &fbc, &gclid, &gbraid, &wbraid, &ttclid); err != nil {
			return nil, err
		}
		src := attributionSource(originType, ctwaClid, fbc, gclid, gbraid, wbraid, ttclid)
		out[src]++
	}
	return out, rows.Err()
}

type CTWALead struct {
	ChatID          int64     `json:"chat_id"`
	ContactName     string    `json:"contact_name"`
	Phone           string    `json:"phone"`
	CampaignName    string    `json:"campaign_name"`
	AdsetName       string    `json:"adset_name"`
	AdName          string    `json:"ad_name"`
	LastMessageTime time.Time `json:"last_message_time"`
}

func (r *WhatsAppRepo) CTWALeads(ctx context.Context, f WhatsAppStatsFilter) ([]CTWALead, error) {
	conds, args := buildWhatsAppStatsConds(f, "c.last_message_time", "(c.ctwa_clid <> '' OR u.ctwa_clid <> '')")
	q := `
		SELECT c.id, c.contact_name, c.last_message_time,
		       COALESCE(NULLIF(c.campaign_name, ''), u.last_utm_campaign),
		       COALESCE(NULLIF(c.adset_name, ''), u.last_utm_medium),
		       COALESCE(NULLIF(c.ad_name, ''), u.last_utm_content),
		       COALESCE(NULLIF(c.phone, ''), u.phone)
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		JOIN tracked_users u ON c.tracked_user_id = u.id
		JOIN pixels p ON i.pixel_id = p.id
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY c.last_message_time DESC
		LIMIT 50
	`
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CTWALead{}
	for rows.Next() {
		var lead CTWALead
		var contactName *string
		if err := rows.Scan(&lead.ChatID, &contactName, &lead.LastMessageTime,
			&lead.CampaignName, &lead.AdsetName, &lead.AdName, &lead.Phone); err != nil {
			return nil, err
		}
		if contactName != nil {
			lead.ContactName = *contactName
		}
		out = append(out, lead)
	}
	return out, rows.Err()
}

