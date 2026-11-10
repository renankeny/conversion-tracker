package repo

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type WebhookLogRepo struct {
	pool *pgxpool.Pool
}

func NewWebhookLogRepo(pool *pgxpool.Pool) *WebhookLogRepo {
	return &WebhookLogRepo{pool: pool}
}

// Create insere um novo webhook_log.
// input é o WebhookPurchaseInput serializado para replay (pode ser nil se o evento falhou antes).
func (r *WebhookLogRepo) Create(ctx context.Context, log *domain.WebhookLog, input *domain.WebhookPurchaseInput) error {
	return r.CreateWithRawPayload(ctx, log, input, nil, "")
}

// CreateWithRawPayload insere log com payload cru para auditoria de custom webhooks.
func (r *WebhookLogRepo) CreateWithRawPayload(
	ctx context.Context,
	log *domain.WebhookLog,
	input *domain.WebhookPurchaseInput,
	rawPayload []byte,
	customWebhookID string,
) error {
	var inputJSON, rawPayloadStr, cwID *string
	if input != nil {
		if raw, err := json.Marshal(input); err == nil {
			s := string(raw)
			inputJSON = &s
		}
	}
	if rawPayload != nil {
		s := string(rawPayload)
		rawPayloadStr = &s
	}
	if customWebhookID != "" {
		cwID = &customWebhookID
	}

	const q = `
		INSERT INTO webhook_logs
			(account_id, source, pixel_name, external_id, event_name,
			 event_id, input_json, status, error, received_at, raw_payload, custom_webhook_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11::jsonb,$12)
		RETURNING id
	`
	return r.pool.QueryRow(ctx, q,
		log.AccountID, log.Source, log.PixelName, log.ExternalID,
		log.EventName, log.EventID, inputJSON,
		log.Status, log.Error, log.ReceivedAt, rawPayloadStr, cwID,
	).Scan(&log.ID)
}

// CustomWebhookAuditLog é o retorno para a tela de auditoria.
type CustomWebhookAuditLog struct {
	ID             int64     `json:"id"`
	ReceivedAt     time.Time `json:"received_at"`
	Status         string    `json:"status"`
	EventName      string    `json:"event_name"`
	Email          string    `json:"email"`
	ExternalID     string    `json:"external_id"`
	Error          string    `json:"error"`
	RawPayload     any       `json:"raw_payload"`
	ProcessedInput any       `json:"processed_input"`
}

// ListByCustomWebhook retorna logs de auditoria para uma integração específica.
func (r *WebhookLogRepo) ListByCustomWebhook(ctx context.Context, customWebhookID string, accountID int64, limit int) ([]CustomWebhookAuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := `
		SELECT id, received_at, status, event_name, external_id, error,
		       raw_payload::text, input_json::text
		FROM webhook_logs
		WHERE custom_webhook_id = $1 AND account_id = $2
		ORDER BY received_at DESC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, customWebhookID, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CustomWebhookAuditLog
	for rows.Next() {
		var l CustomWebhookAuditLog
		var rawPayload, inputJSON *string
		if err := rows.Scan(&l.ID, &l.ReceivedAt, &l.Status, &l.EventName,
			&l.ExternalID, &l.Error, &rawPayload, &inputJSON); err != nil {
			return nil, err
		}
		if rawPayload != nil {
			var v any
			if json.Unmarshal([]byte(*rawPayload), &v) == nil {
				l.RawPayload = v
			}
		}
		if inputJSON != nil {
			var v any
			if json.Unmarshal([]byte(*inputJSON), &v) == nil {
				l.ProcessedInput = v
			}
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// WebhookLogFilter agrupa filtros para List.
type WebhookLogFilter struct {
	AccountID int64
	Source    string
	Status    string
	From      time.Time
	To        time.Time
	Limit     int
	Offset    int
}

// List retorna logs paginados com total.
func (r *WebhookLogRepo) List(ctx context.Context, f WebhookLogFilter) ([]domain.WebhookLog, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 500 {
		f.Limit = 500
	}

	args := make([]any, 0, 6)
	conds := make([]string, 0, 5)
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, strings.Replace(cond, "$N", "$"+itoa(len(args)), 1))
	}

	if f.AccountID != 0 {
		add("account_id = $N", f.AccountID)
	}
	if f.Source != "" {
		add("source = $N", f.Source)
	}
	if f.Status != "" {
		add("status = $N", f.Status)
	}
	if !f.From.IsZero() {
		add("received_at >= $N", f.From)
	}
	if !f.To.IsZero() {
		add("received_at < $N", f.To)
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM webhook_logs "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	q := "SELECT id, account_id, source, pixel_name, external_id, event_name, " +
		"event_id, status, error, received_at FROM webhook_logs " + where +
		" ORDER BY received_at DESC LIMIT $" + itoa(len(args)-1) + " OFFSET $" + itoa(len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.WebhookLog, 0, f.Limit)
	for rows.Next() {
		var l domain.WebhookLog
		if err := rows.Scan(&l.ID, &l.AccountID, &l.Source, &l.PixelName,
			&l.ExternalID, &l.EventName, &l.EventID,
			&l.Status, &l.Error, &l.ReceivedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}

// FindInputByID retorna o WebhookPurchaseInput de um log específico (para replay).
func (r *WebhookLogRepo) FindInputByID(ctx context.Context, id, accountID int64) (*domain.WebhookPurchaseInput, *domain.WebhookLog, error) {
	where := "WHERE id = $1"
	args := []any{id}
	if accountID != 0 {
		where += " AND account_id = $2"
		args = append(args, accountID)
	}

	var log domain.WebhookLog
	var inputRaw []byte
	err := r.pool.QueryRow(ctx,
		"SELECT id, account_id, source, pixel_name, external_id, event_name, "+
			"event_id, status, error, received_at, input_json::text FROM webhook_logs "+where,
		args...,
	).Scan(&log.ID, &log.AccountID, &log.Source, &log.PixelName,
		&log.ExternalID, &log.EventName, &log.EventID,
		&log.Status, &log.Error, &log.ReceivedAt, &inputRaw)
	if err != nil || inputRaw == nil {
		return nil, &log, err
	}

	var input domain.WebhookPurchaseInput
	if err := json.Unmarshal(inputRaw, &input); err != nil {
		return nil, &log, nil
	}
	return &input, &log, nil
}

// UpdateStatus atualiza o status e error de um log.
func (r *WebhookLogRepo) UpdateStatus(ctx context.Context, id int64, status, errMsg string) error {
	_, err := r.pool.Exec(ctx, "UPDATE webhook_logs SET status=$1, error=$2 WHERE id=$3", status, errMsg, id)
	return err
}
