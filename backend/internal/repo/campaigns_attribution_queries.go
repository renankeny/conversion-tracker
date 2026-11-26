package repo

import (
	"context"
	"time"
)

func (r *CampaignsAttributionRepo) queryLeads(ctx context.Context, accountID int64, pixelName string, from, to time.Time) ([]AttributionEventRow, error) {
	const q = `
		WITH dedup_leads AS (
			SELECT e.account_id, e.pixel_name, e.event_id, MAX(e.external_id) AS external_id
			FROM sent_events e
			WHERE e.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead')
			  AND e.status <> 'contested'
			  AND ($1 = 0 OR e.account_id = $1)
			  AND ($2 = '' OR e.pixel_name = $2)
			  AND e.created_at >= $3 AND e.created_at < $4
			GROUP BY e.account_id, e.pixel_name, e.event_id
		)
		SELECT
			0.0::numeric,
			COALESCE(wc.campaign_id, ''), COALESCE(wc.adset_id, ''), COALESCE(NULLIF(wc.source_id, ''), u.wa_ad_id, ''),
			COALESCE(u.first_utm_source, ''), COALESCE(u.first_utm_medium, ''), COALESCE(u.first_utm_campaign, ''),
			COALESCE(u.first_utm_content, ''), COALESCE(u.first_utm_term, ''),
			COALESCE(u.last_utm_source, ''), COALESCE(u.last_utm_medium, ''), COALESCE(u.last_utm_campaign, ''),
			COALESCE(u.last_utm_content, ''), COALESCE(u.last_utm_term, ''),
			(COALESCE(u.gclid, '') <> '' OR COALESCE(u.wbraid, '') <> '' OR COALESCE(u.gbraid, '') <> ''),
			(COALESCE(u.fbc, '') <> '' OR COALESCE(u.ctwa_clid, '') <> '' OR COALESCE(u.wa_ad_id, '') <> '' OR COALESCE(wc.source_id, '') <> '')
		FROM dedup_leads dl
		LEFT JOIN tracked_users u ON u.account_id = dl.account_id AND u.external_id = dl.external_id
		LEFT JOIN LATERAL (
			SELECT campaign_id, adset_id, source_id
			FROM whatsapp_chats
			WHERE account_id = dl.account_id
			  AND tracked_user_id = u.id
			  AND (campaign_id <> '' OR adset_id <> '' OR source_id <> '')
			ORDER BY updated_at DESC LIMIT 1
		) wc ON u.id IS NOT NULL
	`
	return r.scanAttrRows(ctx, "lead", q, accountID, pixelName, from, to)
}

func (r *CampaignsAttributionRepo) queryConversations(ctx context.Context, accountID int64, pixelName string, from, to time.Time) ([]AttributionEventRow, error) {
	const q = `
		SELECT
			wc.ai_qualified,
			COALESCE(wc.campaign_id, ''), COALESCE(wc.adset_id, ''), COALESCE(NULLIF(wc.source_id, ''), u.wa_ad_id, ''),
			COALESCE(u.first_utm_source, ''), COALESCE(u.first_utm_medium, ''), COALESCE(u.first_utm_campaign, ''),
			COALESCE(u.first_utm_content, ''), COALESCE(u.first_utm_term, ''),
			COALESCE(u.last_utm_source, ''), COALESCE(u.last_utm_medium, ''), COALESCE(u.last_utm_campaign, ''),
			COALESCE(u.last_utm_content, ''), COALESCE(u.last_utm_term, ''),
			(COALESCE(u.gclid, '') <> '' OR COALESCE(u.wbraid, '') <> '' OR COALESCE(u.gbraid, '') <> ''),
			(COALESCE(u.fbc, '') <> '' OR COALESCE(u.ctwa_clid, '') <> '' OR COALESCE(wc.ctwa_clid, '') <> '' OR COALESCE(wc.source_id, '') <> '')
		FROM whatsapp_chats wc
		JOIN whatsapp_instances wi ON wi.id = wc.instance_id
		JOIN pixels px ON px.id = wi.pixel_id
		LEFT JOIN tracked_users u ON u.id = wc.tracked_user_id
		WHERE wc.tracked_user_id IS NOT NULL
		  AND ($1 = 0 OR wc.account_id = $1)
		  AND ($2 = '' OR px.name = $2)
		  AND wc.created_at >= $3 AND wc.created_at < $4
	`
	rows, err := r.pool.Query(ctx, q, accountID, pixelName, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AttributionEventRow
	for rows.Next() {
		row := AttributionEventRow{Kind: "conversation"}
		if err := rows.Scan(
			&row.AIQualified,
			&row.CampaignID, &row.AdSetID, &row.AdID,
			&row.FirstUTMSource, &row.FirstUTMMedium, &row.FirstUTMCampaign, &row.FirstUTMContent, &row.FirstUTMTerm,
			&row.LastUTMSource, &row.LastUTMMedium, &row.LastUTMCampaign, &row.LastUTMContent, &row.LastUTMTerm,
			&row.HasGoogleClick, &row.HasMetaClick,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *CampaignsAttributionRepo) scanAttrRows(ctx context.Context, kind, q string, accountID int64, pixelName string, from, to time.Time) ([]AttributionEventRow, error) {
	rows, err := r.pool.Query(ctx, q, accountID, pixelName, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AttributionEventRow
	for rows.Next() {
		row := AttributionEventRow{Kind: kind}
		if err := rows.Scan(
			&row.Value,
			&row.CampaignID, &row.AdSetID, &row.AdID,
			&row.FirstUTMSource, &row.FirstUTMMedium, &row.FirstUTMCampaign, &row.FirstUTMContent, &row.FirstUTMTerm,
			&row.LastUTMSource, &row.LastUTMMedium, &row.LastUTMCampaign, &row.LastUTMContent, &row.LastUTMTerm,
			&row.HasGoogleClick, &row.HasMetaClick,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
