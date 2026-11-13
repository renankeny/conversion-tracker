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
			"e.status <> 'contested'",
			"(e.input_json->>'gateway' = 'whatsapp' OR e.payload->'data'->0->>'action_source' = 'business_messaging' OR e.payload->'data'->0->>'messaging_channel' = 'whatsapp' OR e.event_id LIKE 'ctwa:%' OR e.event_id LIKE 'wa_trig:%' OR e.event_id LIKE 'wpp_manual_%' OR e.event_id LIKE 'ai_%')",
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
				COUNT(*) FILTER (WHERE event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead') AND has_sent),
				COUNT(*) FILTER (WHERE event_name = 'Purchase' AND has_sent),
				COALESCE(SUM(max_val) FILTER (WHERE event_name = 'Purchase' AND has_sent), 0)
			FROM (
				SELECT e.event_id, MAX(e.event_name) AS event_name,
				       BOOL_OR(e.status = 'sent') AS has_sent, MAX(e.purchase_value) AS max_val
				FROM sent_events e
				WHERE ` + strings.Join(conds, " AND ") + `
				GROUP BY e.event_id
			) dedup`
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
	q := `SELECT EXTRACT(DOW FROM m.timestamp AT TIME ZONE 'America/Sao_Paulo')::int,
	             EXTRACT(HOUR FROM m.timestamp AT TIME ZONE 'America/Sao_Paulo')::int, COUNT(*)
	      FROM whatsapp_messages m
	      JOIN whatsapp_chats c ON m.chat_id = c.id
	      JOIN whatsapp_instances i ON c.instance_id = i.id
	      JOIN pixels p ON i.pixel_id = p.id
	      WHERE ` + strings.Join(conds, " AND ") + ` GROUP BY 1, 2`
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
	conds, args := buildWhatsAppPortalStatsConds(f)
	q := `
		SELECT c.origin_type, u.ctwa_clid, u.fbc, u.gclid, u.gbraid, u.wbraid, u.ttclid
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		JOIN tracked_users u ON c.tracked_user_id = u.id
		JOIN pixels p ON i.pixel_id = p.id
		WHERE ` + strings.Join(conds, " AND ") + `
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
	conds, args := buildWhatsAppPortalStatsConds(f, "(c.ctwa_clid <> '' OR u.ctwa_clid <> '')")
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

