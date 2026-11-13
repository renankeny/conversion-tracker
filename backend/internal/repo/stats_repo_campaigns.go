package repo

import (
	"context"
	"strings"
)

func (s *StatsRepo) populateTrackedUsersAndQuality(ctx context.Context, f StatsFilter, out *Overview) error {
	tuConds := []string{}
	tuArgs := []any{}
	tuIdx := 0
	tuAdd := func(v any) string {
		tuArgs = append(tuArgs, v)
		tuIdx++
		return "$" + itoa(tuIdx)
	}
	if f.PixelName != "" {
		tuConds = append(tuConds, "pixel_name = "+tuAdd(f.PixelName))
	}
	if f.AccountID != 0 {
		tuConds = append(tuConds, "account_id = "+tuAdd(f.AccountID))
	}
	if !f.From.IsZero() && !f.To.IsZero() {
		fromP := tuAdd(f.From)
		toP := tuAdd(f.To)
		tuConds = append(tuConds, "((last_seen_at >= "+fromP+" AND last_seen_at < "+toP+") OR (created_at >= "+fromP+" AND created_at < "+toP+"))")
	} else if !f.From.IsZero() {
		fromP := tuAdd(f.From)
		tuConds = append(tuConds, "(last_seen_at >= "+fromP+" OR created_at >= "+fromP+")")
	} else if !f.To.IsZero() {
		toP := tuAdd(f.To)
		tuConds = append(tuConds, "(last_seen_at < "+toP+" OR created_at < "+toP+")")
	}

	if srcCond := trafficSourceUserCond(f.TrafficSource, ""); srcCond != "" {
		tuConds = append(tuConds, srcCond)
	}

	tuWhere := ""
	if len(tuConds) > 0 {
		tuWhere = "WHERE " + strings.Join(tuConds, " AND ")
	}

	tqQuery := `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE traffic_type = 'human' OR traffic_type = '' OR traffic_type IS NULL),
			COUNT(*) FILTER (WHERE traffic_type = 'bot'),
			COUNT(*) FILTER (WHERE traffic_type = 'inconclusive')
		FROM tracked_users ` + tuWhere
	var tqTotal, tqHuman, tqBot, tqInconclusive int64
	if err := s.pool.QueryRow(ctx, tqQuery, tuArgs...).Scan(&tqTotal, &tqHuman, &tqBot, &tqInconclusive); err != nil {
		return err
	}

	out.TrackedUsers = tqTotal
	out.TrafficQuality.TotalVisitors = tqTotal
	out.TrafficQuality.HumanCount = tqHuman
	out.TrafficQuality.BotCount = tqBot
	out.TrafficQuality.InconclusiveCount = tqInconclusive
	out.TrafficQuality.BotCategories = make(map[string]int64)

	if tqTotal > 0 {
		out.TrafficQuality.HumanPercent = float64(int(float64(tqHuman)/float64(tqTotal)*1000+0.5)) / 10
		out.TrafficQuality.BotPercent = float64(int(float64(tqBot)/float64(tqTotal)*1000+0.5)) / 10
		out.TrafficQuality.InconclusivePercent = float64(int(float64(tqInconclusive)/float64(tqTotal)*1000+0.5)) / 10
	}

	if tqBot > 0 {
		catWhere := tuWhere
		if catWhere == "" {
			catWhere = "WHERE traffic_type = 'bot'"
		} else {
			catWhere += " AND traffic_type = 'bot'"
		}
		catQuery := "SELECT COALESCE(NULLIF(traffic_category, ''), 'generic_bot'), COUNT(*) FROM tracked_users " + catWhere + " GROUP BY 1"
		if catRows, err := s.pool.Query(ctx, catQuery, tuArgs...); err == nil {
			for catRows.Next() {
				var cat string
				var cnt int64
				if err := catRows.Scan(&cat, &cnt); err == nil {
					out.TrafficQuality.BotCategories[cat] = cnt
				}
			}
			catRows.Close()
		}
	}
	return nil
}

func buildGeoConds(f StatsFilter, prefix string, extraEventName string) ([]string, []any) {
	args := []any{}
	conds := []string{}
	idx := 0
	add := func(v any) string {
		args = append(args, v)
		idx++
		return "$" + itoa(idx)
	}
	if prefix == "" {
		if !f.From.IsZero() && !f.To.IsZero() {
			fromP := add(f.From)
			toP := add(f.To)
			conds = append(conds, "((last_seen_at >= "+fromP+" AND last_seen_at < "+toP+") OR (created_at >= "+fromP+" AND created_at < "+toP+"))")
		} else if !f.From.IsZero() {
			fromP := add(f.From)
			conds = append(conds, "(last_seen_at >= "+fromP+" OR created_at >= "+fromP+")")
		} else if !f.To.IsZero() {
			toP := add(f.To)
			conds = append(conds, "(last_seen_at < "+toP+" OR created_at < "+toP+")")
		}
	} else {
		if !f.From.IsZero() {
			conds = append(conds, prefix+"created_at >= "+add(f.From))
		}
		if !f.To.IsZero() {
			conds = append(conds, prefix+"created_at < "+add(f.To))
		}
	}
	if f.PixelName != "" {
		conds = append(conds, prefix+"pixel_name = "+add(f.PixelName))
	}
	if f.AccountID != 0 {
		conds = append(conds, prefix+"account_id = "+add(f.AccountID))
	}
	if extraEventName != "" {
		if extraEventName == "Lead" {
			conds = append(conds, prefix+"event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead')")
		} else {
			conds = append(conds, prefix+"event_name = "+add(extraEventName))
		}
	}
	if srcCond := trafficSourceUserCond(f.TrafficSource, "u."); srcCond != "" {
		if prefix == "" {
			conds = append(conds, trafficSourceUserCond(f.TrafficSource, ""))
		} else {
			conds = append(conds, srcCond)
		}
	}
	return conds, args
}

// GeoDistribution retorna a distribuição geográfica dos usuários rastreados no período.
func (s *StatsRepo) GeoDistribution(ctx context.Context, f StatsFilter) ([]GeoStat, error) {
	conds, args := buildGeoConds(f, "", "")
	conds = append(conds, "country <> ''")

	q := `
		SELECT country, state, city, COUNT(*) as count
		FROM tracked_users
		WHERE ` + strings.Join(conds, " AND ") + `
		GROUP BY country, state, city
		ORDER BY count DESC
		LIMIT 200
	`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []GeoStat
	for rows.Next() {
		var g GeoStat
		if err := rows.Scan(&g.Country, &g.State, &g.City, &g.Count); err != nil {
			return nil, err
		}
		stats = append(stats, g)
	}
	return stats, rows.Err()
}

// GeoDistributionByEvent retorna a distribuição geográfica por tipo de evento.
func (s *StatsRepo) GeoDistributionByEvent(ctx context.Context, f StatsFilter, eventName string) ([]GeoStatByEvent, error) {
	if eventName == "PageView" || eventName == "ViewContent" || eventName == "" {
		evLabel := "PageView"
		if eventName == "ViewContent" {
			evLabel = "ViewContent"
		}
		conds, args := buildGeoConds(f, "", "")
		conds = append(conds, "country <> ''")

		q := `
			SELECT country, state, city, '` + evLabel + `' as event_name, COUNT(*) as count
			FROM tracked_users
			WHERE ` + strings.Join(conds, " AND ") + `
			GROUP BY country, state, city
			ORDER BY count DESC
			LIMIT 200
		`
		rows, err := s.pool.Query(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var stats []GeoStatByEvent
		for rows.Next() {
			var g GeoStatByEvent
			if err := rows.Scan(&g.Country, &g.State, &g.City, &g.EventName, &g.Count); err != nil {
				return nil, err
			}
			stats = append(stats, g)
		}
		return stats, rows.Err()
	}

	conds, args := buildGeoConds(f, "e.", eventName)
	conds = append(conds, "u.country <> ''", "e.status = 'sent'")

	q := `
		SELECT u.country, u.state, u.city, e.event_name, COUNT(DISTINCT e.event_id) as count
		FROM sent_events e
		INNER JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id
		WHERE ` + strings.Join(conds, " AND ") + `
		GROUP BY u.country, u.state, u.city, e.event_name
		ORDER BY count DESC
		LIMIT 200
	`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []GeoStatByEvent
	for rows.Next() {
		var g GeoStatByEvent
		if err := rows.Scan(&g.Country, &g.State, &g.City, &g.EventName, &g.Count); err != nil {
			return nil, err
		}
		stats = append(stats, g)
	}
	return stats, rows.Err()
}
