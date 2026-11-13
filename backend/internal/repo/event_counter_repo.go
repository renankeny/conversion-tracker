package repo

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type EventCounterRepo struct {
	pool *pgxpool.Pool
}

func NewEventCounterRepo(pool *pgxpool.Pool) *EventCounterRepo {
	return &EventCounterRepo{pool: pool}
}

// Increment faz UPSERT na bucket de hora correspondente a `at`.
func (r *EventCounterRepo) Increment(
	ctx context.Context,
	accountID int64,
	pixelName, eventName string,
	at time.Time,
	delta domain.CounterDelta,
) error {
	if accountID == 0 {
		accountID = 1
	}
	bucket := at.UTC().Truncate(time.Hour)
	const q = `
		INSERT INTO event_counters
			(account_id, pixel_name, event_name, bucket_at, count_total, count_sent, count_failed)
		VALUES
			($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (account_id, pixel_name, event_name, bucket_at) DO UPDATE
		SET count_total  = GREATEST(0, event_counters.count_total  + EXCLUDED.count_total),
		    count_sent   = GREATEST(0, event_counters.count_sent   + EXCLUDED.count_sent),
		    count_failed = GREATEST(0, event_counters.count_failed + EXCLUDED.count_failed),
		    updated_at   = NOW()
	`
	_, err := r.pool.Exec(ctx, q,
		accountID, pixelName, eventName, bucket,
		delta.Total, delta.Sent, delta.Failed,
	)
	return err
}

// SummaryRow é o agregado retornado pelas queries de dashboard.
type SummaryRow struct {
	PixelName   string `json:"pixel_name,omitempty"`
	EventName   string `json:"event_name,omitempty"`
	CountTotal  int64  `json:"count_total"`
	CountSent   int64  `json:"count_sent"`
	CountFailed int64  `json:"count_failed"`
}

// SummaryFilters limita o agregado a um intervalo de tempo + filtros opcionais.
type SummaryFilters struct {
	From          *time.Time
	To            *time.Time
	PixelName     string
	EventName     string
	AccountID     int64
	TrafficSource string
}

// SummaryByEvent agrega por evento dentro do intervalo.
func (r *EventCounterRepo) SummaryByEvent(ctx context.Context, f SummaryFilters) ([]SummaryRow, error) {
	if f.TrafficSource != "" && f.TrafficSource != "all" {
		q := `
			SELECT e.event_name,
			       COUNT(DISTINCT e.event_id) AS total,
			       COUNT(DISTINCT CASE WHEN e.status = 'sent' THEN e.event_id END) AS sent,
			       COUNT(DISTINCT CASE WHEN e.status = 'failed' THEN e.event_id END) AS failed
			FROM sent_events e
			JOIN tracked_users u ON e.external_id = u.external_id AND e.account_id = u.account_id
			WHERE e.status <> 'contested'
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
		if srcCond := trafficSourceUserCond(f.TrafficSource, "u."); srcCond != "" {
			q += " AND " + srcCond
		}
		q += " GROUP BY e.event_name ORDER BY total DESC"

		rows, err := r.pool.Query(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []SummaryRow
		var pvSent, pvFailed int64
		for rows.Next() {
			var row SummaryRow
			if err := rows.Scan(&row.EventName, &row.CountTotal, &row.CountSent, &row.CountFailed); err != nil {
				return nil, err
			}
			if row.EventName == "PageView" {
				pvSent += row.CountSent
				pvFailed += row.CountFailed
				continue
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if f.EventName == "" || f.EventName == "PageView" {
			visitorsPV := r.countVisitorsBySource(ctx, f)
			totalPV := max64(visitorsPV, pvSent+pvFailed)
			if totalPV > 0 {
				sentPV := max64(0, totalPV-pvFailed)
				out = append([]SummaryRow{{
					EventName:   "PageView",
					CountTotal:  totalPV,
					CountSent:   sentPV,
					CountFailed: pvFailed,
				}}, out...)
			}
		}
		return out, nil
	}

	q1 := `
		SELECT event_name,
		       COALESCE(SUM(count_total), 0)  AS total,
		       COALESCE(SUM(count_sent), 0)   AS sent,
		       COALESCE(SUM(count_failed), 0) AS failed
		FROM event_counters
		WHERE event_name IN ('PageView', 'ViewContent')
	`
	q2 := `
		SELECT event_name,
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

	q1 += " GROUP BY event_name"
	q2 += " GROUP BY event_name"

	q := q1 + " UNION ALL " + q2 + " ORDER BY total DESC"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SummaryRow
	for rows.Next() {
		var row SummaryRow
		if err := rows.Scan(&row.EventName, &row.CountTotal, &row.CountSent, &row.CountFailed); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
