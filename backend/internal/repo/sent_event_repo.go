package repo

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SentEvent é a representação leve de uma linha de sent_events pra
// listagem no painel admin (não inclui payload pra evitar JSON pesado).
type SentEvent struct {
	ID           int64     `json:"id"`
	EventID      string    `json:"event_id"`
	EventName    string    `json:"event_name"`
	PixelName    string    `json:"pixel_name"`
	ExternalID   string    `json:"external_id"`
	Status       string    `json:"status"`
	ResponseCode int       `json:"response_code"`
	ResponseBody string    `json:"response_body,omitempty"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SentEventRepo struct {
	pool *pgxpool.Pool
}

func NewSentEventRepo(pool *pgxpool.Pool) *SentEventRepo {
	return &SentEventRepo{pool: pool}
}

// ExistsSent retorna true se já houver uma tentativa BEM-SUCEDIDA (status='sent')
// pra esse par (event_id, pixel_name). Usado pra dedup antes do enqueue.
//
// Importante: NÃO bloqueia se a única tentativa anterior falhou — o asynq pode
// querer fazer retry. Bloqueia só quando já chegou com sucesso na Meta.
func (r *SentEventRepo) ExistsSent(ctx context.Context, eventID, pixelName string) (bool, error) {
	if eventID == "" || pixelName == "" {
		return false, nil
	}
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM sent_events
		WHERE event_id = $1 AND pixel_name = $2 AND status = 'sent'
	`, eventID, pixelName).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ListFilter agrupa os filtros aceitos por List.
type ListFilter struct {
	AccountID int64     // 0 = todos (owner); >0 = filtra por conta (member)
	PixelName string    // exato (vazio = todos)
	EventName string    // exato (vazio = todos)
	Status    string    // queued|sent|failed (vazio = todos)
	From      time.Time // created_at >= From (zero = sem filtro)
	To        time.Time // created_at < To (zero = sem filtro)
	Limit     int       // default 50, max 500
	Offset    int
}

// List devolve as linhas filtradas + total count (pra paginação).
func (r *SentEventRepo) List(ctx context.Context, f ListFilter) ([]SentEvent, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 500 {
		f.Limit = 500
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	args := make([]any, 0, 6)
	conds := make([]string, 0, 6)
	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, strings.Replace(cond, "$N", "$"+itoa(len(args)), 1))
	}
	if f.AccountID > 0 {
		add("account_id = $N", f.AccountID)
	}
	if f.PixelName != "" {
		add("pixel_name = $N", f.PixelName)
	}
	if f.EventName != "" {
		add("event_name = $N", f.EventName)
	}
	if f.Status != "" {
		add("status = $N", f.Status)
	}
	if !f.From.IsZero() {
		add("created_at >= $N", f.From)
	}
	if !f.To.IsZero() {
		add("created_at < $N", f.To)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	// Count total
	var total int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM sent_events "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Page
	args = append(args, f.Limit, f.Offset)
	q := "SELECT id, event_id, event_name, pixel_name, external_id, status, " +
		"response_code, response_body, error, created_at, updated_at " +
		"FROM sent_events " + where + " ORDER BY id DESC " +
		"LIMIT $" + itoa(len(args)-1) + " OFFSET $" + itoa(len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]SentEvent, 0, f.Limit)
	for rows.Next() {
		var e SentEvent
		if err := rows.Scan(&e.ID, &e.EventID, &e.EventName, &e.PixelName,
			&e.ExternalID, &e.Status, &e.ResponseCode, &e.ResponseBody, &e.Error,
			&e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// itoa minimal pra montar placeholders sem importar strconv só pra isso
// (evita uma alocação por chamada).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	const digits = "0123456789"
	buf := make([]byte, 0, 4)
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		buf = append([]byte{digits[i%10]}, buf...)
		i /= 10
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}
