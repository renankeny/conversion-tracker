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
	args := []any{}
	conds := []string{
		"status = 'sent'",
		"event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead', 'Purchase')",
	}
	idx := 0
	add := func(v any) string {
		args = append(args, v)
		idx++
		return "$" + itoa(idx)
	}
	if f.PixelName != "" {
		conds = append(conds, "pixel_name = "+add(f.PixelName))
	}
	if f.AccountID != 0 {
		conds = append(conds, "account_id = "+add(f.AccountID))
	}
	if !f.From.IsZero() {
		conds = append(conds, "created_at >= "+add(f.From))
	}
	if !f.To.IsZero() {
		conds = append(conds, "created_at < "+add(f.To))
	}

	q := `
		SELECT se.event_id, se.event_name, se.max_val,
		       u.last_utm_term, u.first_utm_term, u.last_utm_campaign, u.first_utm_campaign
		FROM (
			SELECT event_id, event_name, external_id, account_id, MAX(purchase_value) AS max_val
			FROM sent_events
			WHERE ` + strings.Join(conds, " AND ") + `
			GROUP BY event_id, event_name, external_id, account_id
		) se
		JOIN tracked_users u ON se.external_id = u.external_id AND se.account_id = u.account_id
		WHERE ` + googleSourceSQL
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	type kwConv struct {
		eventID, eventName                              string
		maxVal                                          float64
		lastTerm, firstTerm, lastCampaign, firstCampaign string
	}
	var convs []kwConv
	for rows.Next() {
		var r kwConv
		if rows.Scan(&r.eventID, &r.eventName, &r.maxVal, &r.lastTerm, &r.firstTerm, &r.lastCampaign, &r.firstCampaign) == nil {
			convs = append(convs, r)
		}
	}
	if len(convs) == 0 {
		return
	}

	for i := range keywords {
		kw := &keywords[i]
		term := strings.TrimSpace(kw.Keyword)
		if term == "" {
			continue
		}
		camp := strings.TrimSpace(kw.CampaignName)
		var leads, purchases int64
		var revenue float64
		for _, r := range convs {
			matched := strings.EqualFold(r.lastTerm, term) || strings.EqualFold(r.firstTerm, term) ||
				(camp != "" && r.lastTerm == "" && r.firstTerm == "" && (strings.EqualFold(r.lastCampaign, camp) || strings.EqualFold(r.firstCampaign, camp)))
			if !matched {
				continue
			}
			if r.eventName == "Purchase" {
				purchases++
				revenue += r.maxVal
			} else {
				leads++
			}
		}
		kw.Leads = leads
		kw.Purchases = purchases
		kw.Revenue = revenue
	}
}
