package repo

import (
	"context"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
)

func (s *StatsRepo) EnrichGoogleKeywordsWithTrackedConversions(
	ctx context.Context,
	f StatsFilter,
	keywords []googleads.KeywordInsight,
) {
	if len(keywords) == 0 {
		return
	}
	var fromPtr, toPtr any
	if !f.From.IsZero() {
		fromPtr = f.From
	}
	if !f.To.IsZero() {
		toPtr = f.To
	}

	for i := range keywords {
		kw := &keywords[i]
		term := strings.TrimSpace(kw.Keyword)
		if term == "" {
			continue
		}
		q := `
			SELECT
				COUNT(DISTINCT CASE WHEN se.event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead') THEN se.event_id END),
				COUNT(DISTINCT CASE WHEN se.event_name = 'Purchase' THEN se.event_id END),
				COALESCE(SUM(CASE WHEN se.event_name = 'Purchase' THEN se.max_val ELSE 0 END), 0)
			FROM (
				SELECT event_id, event_name, external_id, account_id, MAX(purchase_value) AS max_val
				FROM sent_events
				WHERE status = 'sent'
				  AND ($1::text = '' OR pixel_name = $1)
				  AND ($2::bigint = 0 OR account_id = $2)
				  AND ($3::timestamptz IS NULL OR created_at >= $3)
				  AND ($4::timestamptz IS NULL OR created_at < $4)
				  AND event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase')
				GROUP BY event_id, event_name, external_id, account_id
			) se
			JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id
			WHERE ` + googleSourceSQL + `
			  AND (
				LOWER(u.last_utm_term) = LOWER($5)
				OR LOWER(u.first_utm_term) = LOWER($5)
				OR ($6 <> '' AND u.last_utm_term = '' AND u.first_utm_term = '' AND (
					LOWER(u.last_utm_campaign) = LOWER($6) OR LOWER(u.first_utm_campaign) = LOWER($6)
				))
			  )
		`
		_ = s.pool.QueryRow(ctx, q, f.PixelName, f.AccountID, fromPtr, toPtr, term, strings.TrimSpace(kw.CampaignName)).Scan(
			&kw.Leads, &kw.Purchases, &kw.Revenue,
		)
	}
}

