package repo

import (
	"strings"
	"time"
)

// ListFilter agrupa os filtros aceitos por List e ListGrouped.
type ListFilter struct {
	AccountID          int64
	PixelName          string
	Channel            string
	Search             string
	EventNames         []string
	ExternalID         string
	ExternalIDs        []string
	Status             string
	From               time.Time
	To                 time.Time
	Limit              int
	Offset             int
	IncludeLTV         bool
	IncludeBuyer       bool
	IncludeUserContext bool
	TrafficSource      string
}

func buildGroupedWhere(f ListFilter) (string, []any) {
	args := make([]any, 0, 8)
	conds := make([]string, 0, 8)
	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, strings.ReplaceAll(cond, "$N", "$"+itoa(len(args))))
	}
	if f.AccountID != 0 {
		add("account_id = $N", f.AccountID)
	}
	if f.PixelName != "" {
		add("pixel_name = $N", f.PixelName)
	}
	if f.Channel != "" {
		add("channel = $N", f.Channel)
	}
	if len(f.EventNames) > 0 {
		add("event_name = ANY($N::text[])", f.EventNames)
	}
	if len(f.ExternalIDs) > 0 {
		add("external_id = ANY($N::text[])", f.ExternalIDs)
	} else if f.ExternalID != "" {
		add("external_id = $N", f.ExternalID)
	}
	if search := strings.TrimSpace(f.Search); search != "" {
		add(sentEventsSearchSQL, "%"+search+"%")
	}

	switch f.Status {
	case "contested":
		conds = append(conds, "status = 'contested'")
	case "sent":
		conds = append(conds,
			"status = 'sent'",
			"NOT EXISTS (SELECT 1 FROM sent_events s2 WHERE s2.account_id = sent_events.account_id AND s2.pixel_name = sent_events.pixel_name AND s2.event_id = sent_events.event_id AND s2.status = 'failed')",
		)
	case "partial":
		conds = append(conds,
			"status = 'failed'",
			"EXISTS (SELECT 1 FROM sent_events s2 WHERE s2.account_id = sent_events.account_id AND s2.pixel_name = sent_events.pixel_name AND s2.event_id = sent_events.event_id AND s2.status = 'sent')",
		)
	case "failed":
		conds = append(conds, "status = 'failed'")
	case "":
		conds = append(conds, "status <> 'contested'")
	default:
		add("status = $N", f.Status)
		conds = append(conds, "status <> 'contested'")
	}

	if !f.From.IsZero() {
		add("created_at >= $N", f.From)
	}
	if !f.To.IsZero() {
		add("created_at < $N", f.To)
	}
	if srcCond := trafficSourceUserCond(f.TrafficSource, "u."); srcCond != "" {
		conds = append(conds, "EXISTS (SELECT 1 FROM tracked_users u WHERE u.external_id = sent_events.external_id AND u.account_id = sent_events.account_id AND "+srcCond+")")
	}
	conds = append(conds, excludeHighVolumeEvents)
	return "WHERE " + strings.Join(conds, " AND "), args
}
