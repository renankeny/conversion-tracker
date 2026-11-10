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

// TimeSeries devolve evolução temporal por bucket (já está em hora).
// Se WantDaily for true, agrupa por dia em vez de hora.
func (r *EventCounterRepo) TimeSeries(ctx context.Context, f SummaryFilters, daily bool) ([]SeriesPoint, error) {
	if f.TrafficSource != "" && f.TrafficSource != "all" {
		bucketExpr := "date_trunc('hour', e.created_at)"
		if daily {
			bucketExpr = "date_trunc('day', e.created_at)"
		}
		q := `
			SELECT ` + bucketExpr + ` AS bucket,
			       e.event_name,
			       COUNT(DISTINCT e.event_id) AS total,
			       COUNT(DISTINCT CASE WHEN e.status = 'sent' THEN e.event_id END) AS sent,
			       COUNT(DISTINCT CASE WHEN e.status = 'failed' THEN e.event_id END) AS failed
			FROM sent_events e
			JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id
			WHERE 1=1
		`
		args := []any{}
		idx := 1
		if f.From != nil {
			q += " AND e.created_at >= $" + strconv.Itoa(idx)
			args = append(args, f.From.UTC())
			idx++
		}
		if f.To != nil {
			q += " AND e.created_at < $" + strconv.Itoa(idx)
			args = append(args, f.To.UTC())
			idx++
		}
		if f.PixelName != "" {
			q += " AND e.pixel_name = $" + strconv.Itoa(idx)
			args = append(args, f.PixelName)
			idx++
		}
		if f.EventName != "" {
			q += " AND e.event_name = $" + strconv.Itoa(idx)
			args = append(args, f.EventName)
			idx++
		}
		if f.AccountID != 0 {
			q += " AND e.account_id = $" + strconv.Itoa(idx)
			args = append(args, f.AccountID)
			idx++
		}

		switch f.TrafficSource {
		case "meta_lp":
			q += " AND (u.last_utm_source ILIKE '%facebook%' OR u.last_utm_source ILIKE '%instagram%' OR u.last_utm_source ILIKE '%meta%')"
			q += " AND (e.payload->'data'->0->>'action_source' IS DISTINCT FROM 'business_messaging')"
		case "meta_wpp":
			q += " AND (u.last_utm_source ILIKE '%facebook%' OR u.last_utm_source ILIKE '%instagram%' OR u.last_utm_source ILIKE '%meta%')"
			q += " AND (e.payload->'data'->0->>'action_source' = 'business_messaging' OR e.payload->'data'->0->>'messaging_channel' = 'whatsapp')"
		case "google":
			q += " AND (u.gclid <> '' OR u.last_utm_source ILIKE '%google%')"
		case "tiktok":
			q += " AND (u.ttclid <> '' OR u.last_utm_source ILIKE '%tiktok%')"
		}

		q += " GROUP BY bucket, e.event_name ORDER BY bucket ASC"

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
	`
	args := []any{}
	idx := 1
	if f.From != nil {
		q1 += " AND bucket_at >= $" + strconv.Itoa(idx)
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
