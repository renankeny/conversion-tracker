package repo

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// excludeHighVolumeEvents é sempre aplicado em List/ListGrouped: PageView e
// ViewContent são altíssimo volume e sem valor de auditoria individual, não
// aparecendo na página de Eventos, apenas agregados em event_counters.
const excludeHighVolumeEvents = "event_name NOT IN ('PageView', 'ViewContent')"

const sentEventsSearchSQL = `(event_id ILIKE $N OR external_id ILIKE $N OR EXISTS (
	SELECT 1 FROM tracked_users u
	WHERE u.external_id = sent_events.external_id
	  AND u.account_id = sent_events.account_id
	  AND (
	    (u.email <> '' AND u.email ILIKE $N)
	    OR (u.phone <> '' AND u.phone ILIKE $N)
	    OR (u.first_name <> '' AND u.first_name ILIKE $N)
	    OR (u.last_name <> '' AND u.last_name ILIKE $N)
	  )
))`

// SentEvent é a representação leve de uma linha de sent_events para
// listagem no painel admin, incluindo o payload JSONB transmitido ao canal.
type SentEvent struct {
	ID            int64     `json:"id"`
	EventID       string    `json:"event_id"`
	EventName     string    `json:"event_name"`
	PixelName     string    `json:"pixel_name"`
	ExternalID    string    `json:"external_id"`
	Channel       string    `json:"channel"`
	Status        string    `json:"status"`
	ResponseCode  int       `json:"response_code"`
	ResponseBody  string    `json:"response_body,omitempty"`
	Payload       string    `json:"payload,omitempty"`
	Error         string    `json:"error,omitempty"`
	PurchaseValue float64   `json:"purchase_value"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GroupedEvent agrega todos os envios (Meta, TikTok, Google Ads, GA4) de um
// mesmo evento (event_id + pixel) em uma única linha.
type GroupedEvent struct {
	EventID            string                  `json:"event_id"`
	EventName          string                  `json:"event_name"`
	PixelName          string                  `json:"pixel_name"`
	ExternalID         string                  `json:"external_id"`
	PurchaseValue      float64                 `json:"purchase_value"`
	CreatedAt          time.Time               `json:"created_at"`
	Channels           []ChannelBrief          `json:"channels"`
	LTV                *float64                `json:"ltv,omitempty"`
	FirstUTMSource     string                  `json:"first_utm_source,omitempty"`
	LastUTMSource      string                  `json:"last_utm_source,omitempty"`
	BuyerFirstName     string                  `json:"buyer_first_name,omitempty"`
	BuyerLastName      string                  `json:"buyer_last_name,omitempty"`
	BuyerPhone         string                  `json:"buyer_phone,omitempty"`
	AggregatedCount    int                     `json:"aggregated_count"`
	AggregatedItems    []domain.AggregatedItem `json:"aggregated_items,omitempty"`
}

// ChannelBrief é a visão resumida de um canal dentro de GroupedEvent.
type ChannelBrief struct {
	ID           int64  `json:"id"`
	Channel      string `json:"channel"`
	Status       string `json:"status"`
	ResponseCode int    `json:"response_code"`
	HasError     bool   `json:"has_error"`
}

type SentEventRepo struct {
	pool *pgxpool.Pool
}

func NewSentEventRepo(pool *pgxpool.Pool) *SentEventRepo {
	return &SentEventRepo{pool: pool}
}

// ExistsSent retorna true se já houver envio concluído (status='sent') para a conta/pixel/evento.
func (r *SentEventRepo) ExistsSent(ctx context.Context, accountID int64, eventID, pixelName, channel string) (bool, error) {
	if eventID == "" || pixelName == "" {
		return false, nil
	}
	var exists bool
	if channel != "" {
		err := r.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sent_events
				WHERE ($1::bigint = 0 OR account_id = $1)
				  AND event_id = $2 AND pixel_name = $3 AND channel = $4 AND status = 'sent'
			)
		`, accountID, eventID, pixelName, channel).Scan(&exists)
		return exists, err
	}
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sent_events
			WHERE ($1::bigint = 0 OR account_id = $1)
			  AND event_id = $2 AND pixel_name = $3 AND channel <> 'internal' AND status = 'sent'
		) AND NOT EXISTS (
			SELECT 1 FROM sent_events
			WHERE ($1::bigint = 0 OR account_id = $1)
			  AND event_id = $2 AND pixel_name = $3 AND channel <> 'internal' AND status = 'failed'
		)
	`, accountID, eventID, pixelName).Scan(&exists)
	return exists, err
}

// ExistsRecentEvent verifica se já foi registrado um evento do mesmo tipo para o mesmo
// external_id, pixel e conta dentro de uma janela recente.
func (r *SentEventRepo) ExistsRecentEvent(ctx context.Context, accountID int64, pixelName, externalID, eventName string, window time.Duration) (bool, error) {
	if pixelName == "" || externalID == "" || eventName == "" {
		return false, nil
	}
	since := time.Now().Add(-window)
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM sent_events
		WHERE ($1::bigint = 0 OR account_id = $1)
		  AND pixel_name = $2 AND external_id = $3 AND event_name = $4 AND created_at >= $5
	`, accountID, pixelName, externalID, eventName, since).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetSuccessfulChannels retorna o conjunto de canais já despachados com status = 'sent'
// para (account_id, event_id, pixel_name).
func (r *SentEventRepo) GetSuccessfulChannels(ctx context.Context, accountID int64, eventID, pixelName string) (map[string]bool, error) {
	if eventID == "" || pixelName == "" {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT channel FROM sent_events
		WHERE ($1::bigint = 0 OR account_id = $1)
		  AND event_id = $2 AND pixel_name = $3 AND channel <> 'internal' AND status = 'sent'
	`, accountID, eventID, pixelName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sent := make(map[string]bool)
	for rows.Next() {
		var ch string
		if err := rows.Scan(&ch); err == nil && ch != "" {
			sent[ch] = true
		}
	}
	return sent, rows.Err()
}

// List devolve as linhas filtradas + total count (para paginação).
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
	if f.Status != "" {
		add("status = $N", f.Status)
	}
	if f.Status != "contested" {
		conds = append(conds, "status <> 'contested'")
	}
	if !f.From.IsZero() {
		add("created_at >= $N", f.From)
	}
	if !f.To.IsZero() {
		add("created_at < $N", f.To)
	}
	conds = append(conds, excludeHighVolumeEvents)
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM sent_events "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	q := "SELECT id, event_id, event_name, pixel_name, external_id, channel, status, " +
		"response_code, response_body, error, COALESCE(payload::text, '') AS payload, " +
		"purchase_value, created_at, updated_at " +
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
			&e.ExternalID, &e.Channel, &e.Status, &e.ResponseCode, &e.ResponseBody, &e.Error,
			&e.Payload, &e.PurchaseValue, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}
