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
		tuConds = append(tuConds, "((last_seen_at >= "+tuAdd(f.From)+" AND last_seen_at < "+tuAdd(f.To)+") OR (created_at >= "+tuAdd(f.From)+" AND created_at < "+tuAdd(f.To)+"))")
	} else if !f.From.IsZero() {
		tuConds = append(tuConds, "(last_seen_at >= "+tuAdd(f.From)+" OR created_at >= "+tuAdd(f.From)+")")
	} else if !f.To.IsZero() {
		tuConds = append(tuConds, "(last_seen_at < "+tuAdd(f.To)+" OR created_at < "+tuAdd(f.To)+")")
	}

	switch f.TrafficSource {
	case "meta_lp":
		tuConds = append(tuConds, "(last_utm_source ILIKE '%facebook%' OR last_utm_source ILIKE '%instagram%' OR last_utm_source ILIKE '%meta%')", "(wa_chat_id = '' OR wa_chat_id IS NULL)")
	case "meta_wpp":
		tuConds = append(tuConds, "(last_utm_source ILIKE '%facebook%' OR last_utm_source ILIKE '%instagram%' OR last_utm_source ILIKE '%meta%')", "(wa_chat_id <> '' AND wa_chat_id IS NOT NULL)")
	case "google":
		tuConds = append(tuConds, "(gclid <> '' OR last_utm_source ILIKE '%google%')")
	case "tiktok":
		tuConds = append(tuConds, "(ttclid <> '' OR last_utm_source ILIKE '%tiktok%')")
	}

	tuWhere := ""
	if len(tuConds) > 0 {
		tuWhere = "WHERE " + strings.Join(tuConds, " AND ")
	}
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tracked_users "+tuWhere, tuArgs...).Scan(&out.TrackedUsers); err != nil {
		return err
	}

	tqQuery := `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE traffic_type = 'human' OR traffic_type = '' OR traffic_type IS NULL),
			COUNT(*) FILTER (WHERE traffic_type = 'bot'),
			COUNT(*) FILTER (WHERE traffic_type = 'inconclusive')
		FROM tracked_users ` + tuWhere
	var tqTotal, tqHuman, tqBot, tqInconclusive int64
	if err := s.pool.QueryRow(ctx, tqQuery, tuArgs...).Scan(&tqTotal, &tqHuman, &tqBot, &tqInconclusive); err == nil {
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
	if !f.From.IsZero() {
		conds = append(conds, prefix+"created_at >= "+add(f.From))
	}
	if !f.To.IsZero() {
		conds = append(conds, prefix+"created_at < "+add(f.To))
	}
	if f.PixelName != "" {
		conds = append(conds, prefix+"pixel_name = "+add(f.PixelName))
	}
	if f.AccountID != 0 {
		conds = append(conds, prefix+"account_id = "+add(f.AccountID))
	}
	if extraEventName != "" {
		conds = append(conds, prefix+"event_name = "+add(extraEventName))
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
// Para PageView (visitas), busca em tracked_users (todo usuário rastreado teve PageView).
// Para outros eventos (Purchase, Lead, etc), busca em sent_events JOIN tracked_users.
func (s *StatsRepo) GeoDistributionByEvent(ctx context.Context, f StatsFilter, eventName string) ([]GeoStatByEvent, error) {
	if eventName == "PageView" || eventName == "" {
		conds, args := buildGeoConds(f, "", "")
		conds = append(conds, "country <> ''")

		q := `
			SELECT country, state, city, 'PageView' as event_name, COUNT(*) as count
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
		INNER JOIN tracked_users u ON e.external_id = u.external_id
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
