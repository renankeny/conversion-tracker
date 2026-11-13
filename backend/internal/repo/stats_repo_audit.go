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

const (
	extSentSQL    = `(se.status = 'sent' AND se.channel <> 'internal' AND COALESCE(se.response_code, 200) <> 204)`
	metaSentSQL   = `(se.status = 'sent' AND se.channel = 'meta' AND COALESCE(se.response_code, 200) <> 204)`
	googleSentSQL = `(se.status = 'sent' AND se.channel = 'google_ads' AND COALESCE(se.response_code, 200) <> 204)`
)

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

func auditScopeConds(accountCol, pixelCol string, f StatsFilter, baseIdx int) (string, []any) {
	var conds []string
	var args []any
	next := baseIdx
	if f.AccountID != 0 {
		next++
		conds = append(conds, accountCol+" = $"+itoa(next))
		args = append(args, f.AccountID)
	}
	if f.PixelName != "" {
		next++
		conds = append(conds, pixelCol+" = $"+itoa(next))
		args = append(args, f.PixelName)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(conds, " AND "), args
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
	scopeSQL, scopeArgs := auditScopeConds("i.account_id", "p.name", f, 2)
	args := append([]any{f.From, f.To}, scopeArgs...)
	q := `
		WITH chats AS (
			SELECT c.id AS chat_id, p.name AS pixel_name, i.account_id, u.id AS uid, COALESCE(u.group_id, u.id) AS gkey,
				` + waChatIsMetaSQL + ` AS is_meta,
				` + waChatIsGoogleSQL + ` AS raw_google,
				` + waChatIsTikTokSQL + ` AS is_tiktok
			FROM whatsapp_chats c
			JOIN whatsapp_instances i ON c.instance_id = i.id
			JOIN pixels p ON i.pixel_id = p.id
			LEFT JOIN tracked_users u ON c.tracked_user_id = u.id
			WHERE ` + waChatPeriodEntrySQL("$1", "$2") + scopeSQL + `
		),
		classified AS (
			SELECT is_meta, (NOT is_meta AND raw_google) AS is_google, (is_meta OR raw_google OR is_tiktok) AS is_ad,
				(uid IS NOT NULL AND EXISTS (
					SELECT 1 FROM tracked_users ug
					JOIN sent_events se ON se.external_id = ug.external_id AND se.account_id = ug.account_id
					WHERE ug.account_id = chats.account_id AND COALESCE(ug.group_id, ug.id) = chats.gkey
					  AND se.pixel_name = chats.pixel_name AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase')
					  AND ` + metaSentSQL + `
				)) AS has_meta_sent,
				(uid IS NOT NULL AND EXISTS (
					SELECT 1 FROM tracked_users ug
					JOIN sent_events se ON se.external_id = ug.external_id AND se.account_id = ug.account_id
					WHERE ug.account_id = chats.account_id AND COALESCE(ug.group_id, ug.id) = chats.gkey
					  AND se.pixel_name = chats.pixel_name AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase')
					  AND ` + googleSentSQL + `
				)) AS has_google_sent,
				(uid IS NOT NULL AND EXISTS (
					SELECT 1 FROM tracked_users ug
					JOIN sent_events se ON se.external_id = ug.external_id AND se.account_id = ug.account_id
					WHERE ug.account_id = chats.account_id AND COALESCE(ug.group_id, ug.id) = chats.gkey
					  AND se.pixel_name = chats.pixel_name AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase')
					  AND ` + extSentSQL + `
				)) AS has_ext_sent
			FROM chats
		)
		SELECT COUNT(*) FILTER (WHERE is_ad),
		       COUNT(*) FILTER (WHERE is_ad AND ((is_meta AND has_meta_sent) OR (is_google AND has_google_sent) OR (NOT is_meta AND NOT is_google AND has_ext_sent))),
		       COUNT(*) FILTER (WHERE is_meta), COUNT(*) FILTER (WHERE is_meta AND has_meta_sent),
		       COUNT(*) FILTER (WHERE is_google), COUNT(*) FILTER (WHERE is_google AND has_google_sent)
		FROM classified`
	_ = s.pool.QueryRow(ctx, q, args...).Scan(
		&dst.Total, &dst.Sent, &dst.MetaTotal, &dst.MetaSent, &dst.GoogleTotal, &dst.GoogleSent,
	)
	finalizeAuditCounter(dst, f.TrafficSource)
}

func (s *StatsRepo) populateLeadsAudit(ctx context.Context, f StatsFilter, dst *AuditCounter) {
	clScope, scopeArgs := auditScopeConds("cl.account_id", "u.pixel_name", f, 2)
	waScope, _ := auditScopeConds("i.account_id", "p.name", f, 2)
	seScope, _ := auditScopeConds("se.account_id", "se.pixel_name", f, 2)
	pxCond := ""
	if f.PixelName != "" {
		pxIdx := 3
		if f.AccountID != 0 {
			pxIdx = 4
		}
		pxCond = " AND se.pixel_name = $" + itoa(pxIdx)
	}
	args := append([]any{f.From, f.To}, scopeArgs...)
	q := `
		WITH raw_leads AS (
			SELECT 'u:' || COALESCE(u.group_id, u.id)::text AS lkey, cl.account_id, COALESCE(u.group_id, u.id) AS gkey,
				FALSE AS d_ext, FALSE AS d_meta, FALSE AS d_google, (` + metaSourceSQL + `) AS is_meta, (` + googleSourceSQL + `) AS is_google
			FROM crm_leads cl JOIN tracked_users u ON u.id = cl.tracked_user_id
			WHERE cl.created_at >= $1 AND cl.created_at < $2` + clScope + `
			UNION ALL
			SELECT 'u:' || COALESCE(u.group_id, u.id)::text AS lkey, i.account_id, COALESCE(u.group_id, u.id) AS gkey,
				FALSE AS d_ext, FALSE AS d_meta, FALSE AS d_google,
				` + waChatIsMetaSQL + ` AS is_meta,
				` + waChatIsGoogleSQL + ` AS is_google
			FROM whatsapp_chats c
			JOIN whatsapp_instances i ON c.instance_id = i.id JOIN pixels p ON i.pixel_id = p.id JOIN tracked_users u ON c.tracked_user_id = u.id
			WHERE ` + waChatPeriodEntrySQL("$1", "$2") + waScope + `
			UNION ALL
			SELECT CASE WHEN u.id IS NOT NULL THEN 'u:' || COALESCE(u.group_id, u.id)::text ELSE 'e:' || se.event_id END AS lkey,
				se.account_id, COALESCE(u.group_id, u.id, 0) AS gkey,
				` + extSentSQL + ` AS d_ext, ` + metaSentSQL + ` AS d_meta, ` + googleSentSQL + ` AS d_google,
				COALESCE(` + metaSourceSQL + `, FALSE) AS is_meta, COALESCE(` + googleSourceSQL + `, FALSE) AS is_google
			FROM sent_events se LEFT JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id
			WHERE se.status <> 'contested' AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead')
			  AND se.created_at >= $1 AND se.created_at < $2` + seScope + `
		),
		dedup AS (
			SELECT lkey, MAX(account_id) AS account_id, MAX(gkey) AS gkey,
			       BOOL_OR(d_ext) AS d_ext, BOOL_OR(d_meta) AS d_meta, BOOL_OR(d_google) AS d_google,
			       BOOL_OR(is_meta) AS is_meta, BOOL_OR(is_google) AS raw_google
			FROM raw_leads GROUP BY lkey
		),
		evaluated AS (
			SELECT is_meta, (NOT is_meta AND raw_google) AS is_google,
				(d_meta OR (gkey > 0 AND EXISTS (
					SELECT 1 FROM tracked_users ug JOIN sent_events se ON se.external_id = ug.external_id AND se.account_id = ug.account_id
					WHERE ug.account_id = dedup.account_id AND COALESCE(ug.group_id, ug.id) = dedup.gkey` + pxCond + `
					  AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase') AND ` + metaSentSQL + `
				))) AS has_meta_sent,
				(d_google OR (gkey > 0 AND EXISTS (
					SELECT 1 FROM tracked_users ug JOIN sent_events se ON se.external_id = ug.external_id AND se.account_id = ug.account_id
					WHERE ug.account_id = dedup.account_id AND COALESCE(ug.group_id, ug.id) = dedup.gkey` + pxCond + `
					  AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase') AND ` + googleSentSQL + `
				))) AS has_google_sent,
				(d_ext OR (gkey > 0 AND EXISTS (
					SELECT 1 FROM tracked_users ug JOIN sent_events se ON se.external_id = ug.external_id AND se.account_id = ug.account_id
					WHERE ug.account_id = dedup.account_id AND COALESCE(ug.group_id, ug.id) = dedup.gkey` + pxCond + `
					  AND se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase') AND ` + extSentSQL + `
				))) AS has_ext_sent
			FROM dedup
		)
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE (is_meta AND has_meta_sent) OR (is_google AND has_google_sent) OR (NOT is_meta AND NOT is_google AND has_ext_sent)),
		       COUNT(*) FILTER (WHERE is_meta), COUNT(*) FILTER (WHERE is_meta AND has_meta_sent),
		       COUNT(*) FILTER (WHERE is_google), COUNT(*) FILTER (WHERE is_google AND has_google_sent)
		FROM evaluated`
	_ = s.pool.QueryRow(ctx, q, args...).Scan(
		&dst.Total, &dst.Sent, &dst.MetaTotal, &dst.MetaSent, &dst.GoogleTotal, &dst.GoogleSent,
	)
	finalizeAuditCounter(dst, f.TrafficSource)
}

func (s *StatsRepo) populatePurchasesAudit(ctx context.Context, f StatsFilter, dst *AuditCounter) {
	extraFilter := "TRUE"
	if srcCond := strings.TrimSpace(trafficSourceUserCond(f.TrafficSource, "u.")); srcCond != "" {
		extraFilter = srcCond
	}
	seScope, scopeArgs := auditScopeConds("se.account_id", "se.pixel_name", f, 2)
	args := append([]any{f.From, f.To}, scopeArgs...)
	q := `
		WITH pur AS (
			SELECT se.event_id,
			       BOOL_OR(` + extSentSQL + `) AS has_ext_sent,
			       BOOL_OR(` + metaSentSQL + `) AS has_meta_sent,
			       BOOL_OR(` + googleSentSQL + `) AS has_google_sent,
			       BOOL_OR(COALESCE(` + metaSourceSQL + `, FALSE)) AS is_meta,
			       BOOL_OR(COALESCE(` + googleSourceSQL + `, FALSE)) AS raw_google,
			       BOOL_OR(COALESCE(` + extraFilter + `, FALSE)) AS matches_filter
			FROM sent_events se LEFT JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id
			WHERE se.status <> 'contested' AND se.event_name = 'Purchase'
			  AND se.created_at >= $1 AND se.created_at < $2` + seScope + `
			GROUP BY se.event_id
		)
		SELECT COUNT(*) FILTER (WHERE matches_filter),
		       COUNT(*) FILTER (WHERE matches_filter AND ((is_meta AND has_meta_sent) OR ((NOT is_meta AND raw_google) AND has_google_sent) OR (NOT is_meta AND NOT raw_google AND has_ext_sent))),
		       COUNT(*) FILTER (WHERE is_meta), COUNT(*) FILTER (WHERE is_meta AND has_meta_sent),
		       COUNT(*) FILTER (WHERE NOT is_meta AND raw_google), COUNT(*) FILTER (WHERE NOT is_meta AND raw_google AND has_google_sent)
		FROM pur`
	_ = s.pool.QueryRow(ctx, q, args...).Scan(
		&dst.Total, &dst.Sent, &dst.MetaTotal, &dst.MetaSent, &dst.GoogleTotal, &dst.GoogleSent,
	)
	finalizeAuditCounter(dst, f.TrafficSource)
}
