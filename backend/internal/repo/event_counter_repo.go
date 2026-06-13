package repo

import (
	"context"
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

// Increment faz UPSERT na bucket de hora correspondente a `at`. Cria a row se
// não existir, ou incrementa os contadores se já existir.
//
// O ON CONFLICT garante atomicidade — se 1000 workers chamarem em paralelo
// pra mesma bucket, todos os deltas são aplicados sem race.
func (r *EventCounterRepo) Increment(
	ctx context.Context,
	pixelName, eventName string,
	at time.Time,
	delta domain.CounterDelta,
) error {
	bucket := at.UTC().Truncate(time.Hour)
	const q = `
		INSERT INTO event_counters
			(pixel_name, event_name, bucket_at, count_total, count_sent, count_failed)
		VALUES
			($1, $2, $3, $4, $5, $6)
		ON CONFLICT (pixel_name, event_name, bucket_at) DO UPDATE
		SET count_total  = event_counters.count_total  + EXCLUDED.count_total,
		    count_sent   = event_counters.count_sent   + EXCLUDED.count_sent,
		    count_failed = event_counters.count_failed + EXCLUDED.count_failed,
		    updated_at   = NOW()
	`
	_, err := r.pool.Exec(ctx, q,
		pixelName, eventName, bucket,
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
	From      *time.Time
	To        *time.Time
	PixelName string // opcional, filtra por pixel
	EventName string // opcional, filtra por evento
}

// SummaryByEvent agrega por evento dentro do intervalo. Retorna { event_name → totais }.
// Use isso para o card "PageView 1300, InitiateCheckout 900, Purchase 30".
func (r *EventCounterRepo) SummaryByEvent(ctx context.Context, f SummaryFilters) ([]SummaryRow, error) {
	q := `
		SELECT event_name,
		       COALESCE(SUM(count_total), 0)  AS total,
		       COALESCE(SUM(count_sent), 0)   AS sent,
		       COALESCE(SUM(count_failed), 0) AS failed
		FROM event_counters
		WHERE 1 = 1
	`
	args := []any{}
	idx := 1
	if f.From != nil {
		q += " AND bucket_at >= $" + itoa(idx)
		args = append(args, f.From.UTC())
		idx++
	}
	if f.To != nil {
		q += " AND bucket_at < $" + itoa(idx)
		args = append(args, f.To.UTC())
		idx++
	}
	if f.PixelName != "" {
		q += " AND pixel_name = $" + itoa(idx)
		args = append(args, f.PixelName)
		idx++
	}
	if f.EventName != "" {
		q += " AND event_name = $" + itoa(idx)
		args = append(args, f.EventName)
		idx++
	}
	q += " GROUP BY event_name ORDER BY total DESC"

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
	bucketExpr := "bucket_at"
	if daily {
		bucketExpr = "date_trunc('day', bucket_at)"
	}
	q := `
		SELECT ` + bucketExpr + ` AS bucket,
		       event_name,
		       COALESCE(SUM(count_total), 0)  AS total,
		       COALESCE(SUM(count_sent), 0)   AS sent,
		       COALESCE(SUM(count_failed), 0) AS failed
		FROM event_counters
		WHERE 1 = 1
	`
	args := []any{}
	idx := 1
	if f.From != nil {
		q += " AND bucket_at >= $" + itoa(idx)
		args = append(args, f.From.UTC())
		idx++
	}
	if f.To != nil {
		q += " AND bucket_at < $" + itoa(idx)
		args = append(args, f.To.UTC())
		idx++
	}
	if f.PixelName != "" {
		q += " AND pixel_name = $" + itoa(idx)
		args = append(args, f.PixelName)
		idx++
	}
	if f.EventName != "" {
		q += " AND event_name = $" + itoa(idx)
		args = append(args, f.EventName)
		idx++
	}
	q += " GROUP BY bucket, event_name ORDER BY bucket ASC"

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

// itoa pequeno helper para construir queries numeradas sem importar strconv.
func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	// Suficiente até $99 (mais que precisamos).
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
