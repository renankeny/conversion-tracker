package repo

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// excludeHighVolumeEvents é sempre aplicado em List/ListGrouped: PageView e
// ViewContent são altíssimo volume e sem valor de auditoria individual — não
// aparecem na página de Eventos, só agregados em event_counters. É uma
// exclusão fixa (não vem de input do usuário), por isso vai direto no SQL.
const excludeHighVolumeEvents = "event_name NOT IN ('PageView', 'ViewContent')"

// SentEvent é a representação leve de uma linha de sent_events pra
// listagem no painel admin. Inclui também o payload enviado para a Meta
// (campo `payload` JSONB no banco) — útil pra debugar o que de fato foi
// transmitido (PII hasheada, custom_data, etc).
type SentEvent struct {
	ID            int64     `json:"id"`
	EventID       string    `json:"event_id"`
	EventName     string    `json:"event_name"`
	PixelName     string    `json:"pixel_name"`
	ExternalID    string    `json:"external_id"`
	Channel       string    `json:"channel"` // 'meta' | 'tiktok' | 'google_ads'
	Status        string    `json:"status"`
	ResponseCode  int       `json:"response_code"`
	ResponseBody  string    `json:"response_body,omitempty"`
	Payload       string    `json:"payload,omitempty"` // JSON enviado à plataforma
	Error         string    `json:"error,omitempty"`
	PurchaseValue float64   `json:"purchase_value"` // valor monetário (0 se não aplicável)
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GroupedEvent agrega todos os envios (Meta, TikTok, Google Ads) de um
// mesmo evento (event_id + pixel) em uma unica linha. Usado pelo painel
// pra mostrar 1 row por evento com pills de status por canal.
type GroupedEvent struct {
	EventID       string         `json:"event_id"`
	EventName     string         `json:"event_name"`
	PixelName     string         `json:"pixel_name"`
	ExternalID    string         `json:"external_id"`
	PurchaseValue float64        `json:"purchase_value"` // valor monetário (0 se não aplicável)
	CreatedAt     time.Time      `json:"created_at"`
	Channels      []ChannelBrief `json:"channels"`
	// LTV é o valor total (todas as compras, todo o histórico) do external_id
	// dono desse evento. Só preenchido quando ListFilter.IncludeLTV=true —
	// usado na Dashboard pra mostrar o valor vendido junto do LTV do cliente.
	LTV *float64 `json:"ltv,omitempty"`
	// FirstUTMSource/LastUTMSource vêm de tracked_users (primeiro e último
	// acesso do dono do evento). Só preenchidos quando
	// ListFilter.IncludeUserContext=true — usado na página de Eventos.
	FirstUTMSource string `json:"first_utm_source,omitempty"`
	LastUTMSource  string `json:"last_utm_source,omitempty"`

	// Agrupamento de Conversões (Order Bump & Upsell)
	AggregatedCount int                     `json:"aggregated_count"`
	AggregatedItems []domain.AggregatedItem `json:"aggregated_items,omitempty"`
}

// ChannelBrief e a vista resumida de um canal dentro de GroupedEvent.
// Sem payload completo - so o necessario pra mostrar status na lista.
type ChannelBrief struct {
	ID           int64  `json:"id"` // pra o front conseguir buscar detalhe
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

// ExistsRecentEvent verifica se já foi registrado um evento do mesmo tipo para o mesmo external_id e pixel
// dentro de uma janela recente (ex: 2 segundos). Evita duplicações por duplo clique ou cliques simultâneos de form.
func (r *SentEventRepo) ExistsRecentEvent(ctx context.Context, pixelName, externalID, eventName string, window time.Duration) (bool, error) {
	if pixelName == "" || externalID == "" || eventName == "" {
		return false, nil
	}
	since := time.Now().Add(-window)
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM sent_events
		WHERE pixel_name = $1 AND external_id = $2 AND event_name = $3 AND created_at >= $4
	`, pixelName, externalID, eventName, since).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetSuccessfulChannels retorna o conjunto de canais (meta, tiktok, google_ads, ga4)
// que já foram despachados com status = 'sent' para este (event_id, pixel_name).
// Usado para garantir que retries reenviem APENAS os canais que falharam.
func (r *SentEventRepo) GetSuccessfulChannels(ctx context.Context, eventID, pixelName string) (map[string]bool, error) {
	if eventID == "" || pixelName == "" {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT channel FROM sent_events
		WHERE event_id = $1 AND pixel_name = $2 AND status = 'sent'
	`, eventID, pixelName)
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
	if f.AccountID != 0 {
		add("account_id = $N", f.AccountID)
	}
	if f.PixelName != "" {
		add("pixel_name = $N", f.PixelName)
	}
	if len(f.EventNames) > 0 {
		add("event_name = ANY($N::text[])", f.EventNames)
	}
	if len(f.ExternalIDs) > 0 {
		add("external_id = ANY($N::text[])", f.ExternalIDs)
	} else if f.ExternalID != "" {
		add("external_id = $N", f.ExternalID)
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
	conds = append(conds, excludeHighVolumeEvents)
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM sent_events "+where, args...,
	).Scan(&total); err != nil {
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
