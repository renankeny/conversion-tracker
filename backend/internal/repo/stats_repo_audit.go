package repo

import (
	"context"
	"strings"
)

type AuditCounter struct {
	Total         int64 `json:"total"`
	Sent          int64 `json:"sent"`
	NotSent       int64 `json:"not_sent"`
	MetaTotal     int64 `json:"meta_total"`
	MetaSent      int64 `json:"meta_sent"`
	MetaNotSent   int64 `json:"meta_not_sent"`
	GoogleTotal   int64 `json:"google_total"`
	GoogleSent    int64 `json:"google_sent"`
	GoogleNotSent int64 `json:"google_not_sent"`
}

type PanelAudit struct {
	AdMessages AuditCounter `json:"ad_messages"`
	Leads      AuditCounter `json:"leads"`
	Purchases  AuditCounter `json:"purchases"`
}

func finalizeAuditCounter(c *AuditCounter, trafficSource string) {
	c.MetaNotSent = max64(0, c.MetaTotal-c.MetaSent)
	c.GoogleNotSent = max64(0, c.GoogleTotal-c.GoogleSent)
	switch trafficSource {
	case "meta", "meta_lp", "meta_wpp":
		c.Total, c.Sent = c.MetaTotal, c.MetaSent
	case "google":
		c.Total, c.Sent = c.GoogleTotal, c.GoogleSent
	}
	c.NotSent = max64(0, c.Total-c.Sent)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (s *StatsRepo) populatePanelAudit(ctx context.Context, f StatsFilter, out *Overview) {
	if f.From.IsZero() || f.To.IsZero() {
		return
	}
	s.populateAdMessagesAudit(ctx, f, &out.PanelAudit.AdMessages)
	s.populateLeadsAudit(ctx, f, &out.PanelAudit.Leads)
	s.populatePurchasesAudit(ctx, f, &out.PanelAudit.Purchases)
}

func (s *StatsRepo) populateAdMessagesAudit(ctx context.Context, f StatsFilter, dst *AuditCounter) {
	q := `
		WITH chats AS (
			SELECT c.id AS chat_id, p.name AS pixel_name, i.account_id, u.id AS uid, COALESCE(u.group_id, u.id) AS gkey,
				(c.ctwa_clid <> '' OR c.source_id <> '' OR c.origin_type IN ('meta_ads_ctwa', 'meta_ads_lp') OR (u.id IS NOT NULL AND ` + metaSourceSQL + `)) AS is_meta,
				(c.origin_type = 'google_ads' OR (u.id IS NOT NULL AND ` + googleSourceSQL + `)) AS raw_google,
				(c.origin_type = 'tiktok_ads' OR (u.id IS NOT NULL AND (u.ttclid <> '' OR u.last_utm_source ILIKE '%tiktok%' OR u.first_utm_source ILIKE '%tiktok%'))) AS is_tiktok
			FROM whatsapp_chats c
			JOIN whatsapp_instances i ON c.instance_id = i.id
			JOIN pixels p ON i.pixel_id = p.id
			LEFT JOIN tracked_users u ON c.tracked_user_id = u.id
			WHERE ($1::bigint = 0 OR i.account_id = $1) AND ($2::text = '' OR p.name = $2)
			  AND ((c.created_at >= $3 AND c.created_at < $4) OR EXISTS (
				SELECT 1 FROM whatsapp_messages m WHERE m.chat_id = c.id AND m.is_from_me = FALSE AND m.timestamp >= $3 AND m.timestamp < $4
			  ))
		),
		classified AS (
			SELECT is_meta, (NOT is_meta AND raw_google) AS is_google, (is_meta OR raw_google OR is_tiktok) AS is_ad,
				(uid IS NOT NULL AND EXISTS (
					SELECT 1 FROM tracked_users ug
					JOIN sent_events se ON se.external_id = ug.external_id AND se.account_id = ug.account_id
					WHERE ug.account_id = chats.account_id AND COALESCE(ug.group_id, ug.id) = chats.gkey
					  AND se.pixel_name = chats.pixel_name AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase') AND se.status = 'sent'
				)) AS has_sent
			FROM chats
		)
		SELECT COUNT(*) FILTER (WHERE is_ad), COUNT(*) FILTER (WHERE is_ad AND has_sent),
		       COUNT(*) FILTER (WHERE is_meta), COUNT(*) FILTER (WHERE is_meta AND has_sent),
		       COUNT(*) FILTER (WHERE is_google), COUNT(*) FILTER (WHERE is_google AND has_sent)
		FROM classified`
	_ = s.pool.QueryRow(ctx, q, f.AccountID, f.PixelName, f.From, f.To).Scan(
		&dst.Total, &dst.Sent, &dst.MetaTotal, &dst.MetaSent, &dst.GoogleTotal, &dst.GoogleSent,
	)
	finalizeAuditCounter(dst, f.TrafficSource)
}

func (s *StatsRepo) populateLeadsAudit(ctx context.Context, f StatsFilter, dst *AuditCounter) {
	q := `
		WITH raw_leads AS (
			SELECT 'u:' || COALESCE(u.group_id, u.id)::text AS lkey, cl.account_id, COALESCE(u.group_id, u.id) AS gkey,
				FALSE AS direct_sent, (` + metaSourceSQL + `) AS is_meta, (` + googleSourceSQL + `) AS is_google
			FROM crm_leads cl JOIN tracked_users u ON u.id = cl.tracked_user_id
			WHERE ($1::bigint = 0 OR cl.account_id = $1) AND ($2::text = '' OR u.pixel_name = $2) AND cl.created_at >= $3 AND cl.created_at < $4
			UNION ALL
			SELECT 'u:' || COALESCE(u.group_id, u.id)::text AS lkey, i.account_id, COALESCE(u.group_id, u.id) AS gkey, FALSE AS direct_sent,
				(c.ctwa_clid <> '' OR c.source_id <> '' OR c.origin_type IN ('meta_ads_ctwa', 'meta_ads_lp') OR ` + metaSourceSQL + `) AS is_meta,
				(c.origin_type = 'google_ads' OR ` + googleSourceSQL + `) AS is_google
			FROM whatsapp_chats c
			JOIN whatsapp_instances i ON c.instance_id = i.id JOIN pixels p ON i.pixel_id = p.id JOIN tracked_users u ON c.tracked_user_id = u.id
			WHERE ($1::bigint = 0 OR i.account_id = $1) AND ($2::text = '' OR p.name = $2) AND c.created_at >= $3 AND c.created_at < $4
			UNION ALL
			SELECT CASE WHEN u.id IS NOT NULL THEN 'u:' || COALESCE(u.group_id, u.id)::text ELSE 'e:' || se.event_id END AS lkey,
				se.account_id, COALESCE(u.group_id, u.id, 0) AS gkey, (se.status = 'sent') AS direct_sent,
				COALESCE(` + metaSourceSQL + `, FALSE) AS is_meta, COALESCE(` + googleSourceSQL + `, FALSE) AS is_google
			FROM sent_events se LEFT JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id
			WHERE ($1::bigint = 0 OR se.account_id = $1) AND ($2::text = '' OR se.pixel_name = $2)
			  AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead') AND se.created_at >= $3 AND se.created_at < $4
		),
		dedup AS (
			SELECT lkey, MAX(account_id) AS account_id, MAX(gkey) AS gkey,
			       BOOL_OR(direct_sent) AS direct_sent, BOOL_OR(is_meta) AS is_meta, BOOL_OR(is_google) AS raw_google
			FROM raw_leads GROUP BY lkey
		),
		evaluated AS (
			SELECT is_meta, (NOT is_meta AND raw_google) AS is_google,
				(direct_sent OR (gkey > 0 AND EXISTS (
					SELECT 1 FROM tracked_users ug JOIN sent_events se2 ON se2.external_id = ug.external_id AND se2.account_id = ug.account_id
					WHERE ug.account_id = dedup.account_id AND COALESCE(ug.group_id, ug.id) = dedup.gkey
					  AND ($2::text = '' OR se2.pixel_name = $2) AND se2.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase') AND se2.status = 'sent'
				))) AS has_sent
			FROM dedup
		)
		SELECT COUNT(*), COUNT(*) FILTER (WHERE has_sent),
		       COUNT(*) FILTER (WHERE is_meta), COUNT(*) FILTER (WHERE is_meta AND has_sent),
		       COUNT(*) FILTER (WHERE is_google), COUNT(*) FILTER (WHERE is_google AND has_sent)
		FROM evaluated`
	_ = s.pool.QueryRow(ctx, q, f.AccountID, f.PixelName, f.From, f.To).Scan(
		&dst.Total, &dst.Sent, &dst.MetaTotal, &dst.MetaSent, &dst.GoogleTotal, &dst.GoogleSent,
	)
	finalizeAuditCounter(dst, f.TrafficSource)
}

func (s *StatsRepo) populatePurchasesAudit(ctx context.Context, f StatsFilter, dst *AuditCounter) {
	extraFilter := "TRUE"
	if srcCond := strings.TrimSpace(trafficSourceUserCond(f.TrafficSource, "u.")); srcCond != "" {
		extraFilter = srcCond
	}
	q := `
		WITH pur AS (
			SELECT se.event_id, BOOL_OR(se.status = 'sent') AS has_sent,
			       BOOL_OR(COALESCE(` + metaSourceSQL + `, FALSE)) AS is_meta,
			       BOOL_OR(COALESCE(` + googleSourceSQL + `, FALSE)) AS raw_google,
			       BOOL_OR(COALESCE(` + extraFilter + `, FALSE)) AS matches_filter
			FROM sent_events se LEFT JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id
			WHERE ($1::bigint = 0 OR se.account_id = $1) AND ($2::text = '' OR se.pixel_name = $2)
			  AND se.event_name = 'Purchase' AND se.created_at >= $3 AND se.created_at < $4
			GROUP BY se.event_id
		)
		SELECT COUNT(*) FILTER (WHERE matches_filter), COUNT(*) FILTER (WHERE matches_filter AND has_sent),
		       COUNT(*) FILTER (WHERE is_meta), COUNT(*) FILTER (WHERE is_meta AND has_sent),
		       COUNT(*) FILTER (WHERE NOT is_meta AND raw_google), COUNT(*) FILTER (WHERE NOT is_meta AND raw_google AND has_sent)
		FROM pur`
	_ = s.pool.QueryRow(ctx, q, f.AccountID, f.PixelName, f.From, f.To).Scan(
		&dst.Total, &dst.Sent, &dst.MetaTotal, &dst.MetaSent, &dst.GoogleTotal, &dst.GoogleSent,
	)
	finalizeAuditCounter(dst, f.TrafficSource)
}
