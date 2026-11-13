package repo

import (
	"context"
	"strconv"
	"time"
)

// SeriesPoint é um ponto da série temporal (eventos por hora ou por dia).
type SeriesPoint struct {
	BucketAt    time.Time `json:"bucket_at"`
	EventName   string    `json:"event_name,omitempty"`
	CountTotal  int64     `json:"count_total"`
	CountSent   int64     `json:"count_sent"`
	CountFailed int64     `json:"count_failed"`
}

// TimeSeries devolve evolução temporal por bucket (hora ou dia).
func (r *EventCounterRepo) TimeSeries(ctx context.Context, f SummaryFilters, daily bool) ([]SeriesPoint, error) {
	if f.TrafficSource != "" && f.TrafficSource != "all" {
		unit := "hour"
		if daily {
			unit = "day"
		}
		args := []any{}
		idx := 1

		qEvents := `
			SELECT date_trunc('` + unit + `', e.created_at) AS bucket,
			       e.event_name,
			       COUNT(DISTINCT e.event_id) AS total,
			       COUNT(DISTINCT CASE WHEN e.status = 'sent' THEN e.event_id END) AS sent,
			       COUNT(DISTINCT CASE WHEN e.status = 'failed' THEN e.event_id END) AS failed
			FROM sent_events e
			JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id
			WHERE e.status <> 'contested'
			  AND e.event_name NOT IN ('PageView', 'ViewContent')
		`
		if f.From != nil {
			qEvents += " AND e.created_at >= $" + strconv.Itoa(idx)
			args = append(args, f.From.UTC())
			idx++
		}
		if f.To != nil {
			qEvents += " AND e.created_at < $" + strconv.Itoa(idx)
			args = append(args, f.To.UTC())
			idx++
		}
		if f.PixelName != "" {
			qEvents += " AND e.pixel_name = $" + strconv.Itoa(idx)
			args = append(args, f.PixelName)
			idx++
		}
		if f.EventName != "" {
			qEvents += " AND e.event_name = $" + strconv.Itoa(idx)
			args = append(args, f.EventName)
			idx++
		}
		if f.AccountID != 0 {
			qEvents += " AND e.account_id = $" + strconv.Itoa(idx)
			args = append(args, f.AccountID)
			idx++
		}
		srcCond := trafficSourceUserCond(f.TrafficSource, "u.")
		if srcCond != "" {
			qEvents += " AND " + srcCond
		}
		qEvents += " GROUP BY bucket, e.event_name"

		q := qEvents
		if (f.EventName == "" || f.EventName == "PageView") && srcCond != "" {
			tsExpr := "COALESCE(u.last_seen_at, u.created_at)"
			pvWhere := "WHERE " + srcCond
			if f.PixelName != "" {
				pvWhere += " AND u.pixel_name = $" + strconv.Itoa(idx)
				args = append(args, f.PixelName)
				idx++
			}
			if f.AccountID != 0 {
				pvWhere += " AND u.account_id = $" + strconv.Itoa(idx)
				args = append(args, f.AccountID)
				idx++
			}
			if f.From != nil && f.To != nil {
				fromIdx := strconv.Itoa(idx)
				toIdx := strconv.Itoa(idx + 1)
				args = append(args, f.From.UTC(), f.To.UTC())
				idx += 2
				tsExpr = "CASE WHEN u.last_seen_at >= $" + fromIdx + " AND u.last_seen_at < $" + toIdx + " THEN u.last_seen_at ELSE u.created_at END"
				pvWhere += " AND ((u.last_seen_at >= $" + fromIdx + " AND u.last_seen_at < $" + toIdx + ") OR (u.created_at >= $" + fromIdx + " AND u.created_at < $" + toIdx + "))"
			}
			qPV := `
				SELECT date_trunc('` + unit + `', ` + tsExpr + `) AS bucket,
				       'PageView' AS event_name,
				       COUNT(*) AS total,
				       COUNT(*) AS sent,
				       0::bigint AS failed
				FROM tracked_users u
				` + pvWhere + `
				GROUP BY bucket
			`
			q = qPV + " UNION ALL " + qEvents
		}
		q += " ORDER BY bucket ASC"

		rows, err := r.pool.Query(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []SeriesPoint
		for rows.Next() {
			var p SeriesPoint
			if err := rows.Scan(&p.BucketAt, &p.EventName, &p.CountTotal, &p.CountSent, &p.CountFailed); err != nil {
				return nil, err
			}
			out = append(out, p)
		}
		return out, rows.Err()
	}

	bucketExpr1 := "bucket_at"
	bucketExpr2 := "date_trunc('hour', created_at)"
	if daily {
		bucketExpr1 = "date_trunc('day', bucket_at)"
		bucketExpr2 = "date_trunc('day', created_at)"
	}
	q1 := `
		SELECT ` + bucketExpr1 + ` AS bucket,
		       event_name,
		       COALESCE(SUM(count_total), 0)  AS total,
		       COALESCE(SUM(count_sent), 0)   AS sent,
		       COALESCE(SUM(count_failed), 0) AS failed
		FROM event_counters
		WHERE event_name IN ('PageView', 'ViewContent')
	`
	q2 := `
		SELECT ` + bucketExpr2 + ` AS bucket,
		       event_name,
		       COUNT(DISTINCT event_id) AS total,
		       COUNT(DISTINCT CASE WHEN status = 'sent' THEN event_id END) AS sent,
		       COUNT(DISTINCT CASE WHEN status = 'failed' THEN event_id END) AS failed
		FROM sent_events
		WHERE event_name NOT IN ('PageView', 'ViewContent')
		  AND status <> 'contested'
	`
	args := []any{}
	idx := 1
	if f.From != nil {
		q1 += " AND bucket_at >= $" + strconv.Itoa(idx)
		args = append(args, f.From.UTC().Truncate(time.Hour))
		idx++
		q2 += " AND created_at >= $" + strconv.Itoa(idx)
		args = append(args, f.From.UTC())
		idx++
	}
	if f.To != nil {
		q1 += " AND bucket_at < $" + strconv.Itoa(idx)
		q2 += " AND created_at < $" + strconv.Itoa(idx)
		args = append(args, f.To.UTC())
		idx++
	}
	if f.PixelName != "" {
		q1 += " AND pixel_name = $" + strconv.Itoa(idx)
		q2 += " AND pixel_name = $" + strconv.Itoa(idx)
		args = append(args, f.PixelName)
		idx++
	}
	if f.EventName != "" {
		q1 += " AND event_name = $" + strconv.Itoa(idx)
		q2 += " AND event_name = $" + strconv.Itoa(idx)
		args = append(args, f.EventName)
		idx++
	}
	if f.AccountID != 0 {
		q1 += " AND account_id = $" + strconv.Itoa(idx)
		q2 += " AND account_id = $" + strconv.Itoa(idx)
		args = append(args, f.AccountID)
		idx++
	}

	q1 += " GROUP BY bucket, event_name"
	q2 += " GROUP BY bucket, event_name"

	q := q1 + " UNION ALL " + q2 + " ORDER BY bucket ASC"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SeriesPoint
	for rows.Next() {
		var p SeriesPoint
		if err := rows.Scan(&p.BucketAt, &p.EventName, &p.CountTotal, &p.CountSent, &p.CountFailed); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *EventCounterRepo) countVisitorsBySource(ctx context.Context, f SummaryFilters) int64 {
	srcCond := trafficSourceUserCond(f.TrafficSource, "u.")
	if srcCond == "" {
		return 0
	}
	args := []any{}
	idx := 1
	selectExpr := "COUNT(*)"
	conds := "WHERE " + srcCond
	if f.PixelName != "" {
		conds += " AND u.pixel_name = $" + strconv.Itoa(idx)
		args = append(args, f.PixelName)
		idx++
	}
	if f.AccountID != 0 {
		conds += " AND u.account_id = $" + strconv.Itoa(idx)
		args = append(args, f.AccountID)
		idx++
	}
	if f.From != nil && f.To != nil {
		fromIdx := strconv.Itoa(idx)
		toIdx := strconv.Itoa(idx + 1)
		selectExpr = "COALESCE(SUM(CASE WHEN u.created_at >= $" + fromIdx + " THEN GREATEST(u.pageview_count, 1) ELSE 1 END), 0)"
		conds += " AND ((u.last_seen_at >= $" + fromIdx + " AND u.last_seen_at < $" + toIdx + ") OR (u.created_at >= $" + fromIdx + " AND u.created_at < $" + toIdx + "))"
		args = append(args, f.From.UTC(), f.To.UTC())
	}
	var cnt int64
	_ = r.pool.QueryRow(ctx, "SELECT "+selectExpr+" FROM tracked_users u "+conds, args...).Scan(&cnt)
	return cnt
}
