package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func (r *TrackableLinkRepo) RecordClick(ctx context.Context, c *domain.TrackableLinkClick) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO whatsapp_link_clicks (
			account_id, pixel_id, link_id, tracked_user_id,
			resolved_instance_id, protocol_code, client_ip,
			is_bot, click_event_sent
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at
	`,
		c.AccountID, c.PixelID, c.LinkID, c.TrackedUserID,
		c.ResolvedInstanceID, strings.ToUpper(strings.TrimSpace(c.ProtocolCode)), c.ClientIP,
		c.IsBot, c.ClickEventSent,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *TrackableLinkRepo) FindClickByProtocol(ctx context.Context, accountID int64, protocolCode string) (*domain.TrackableLinkClick, error) {
	proto := strings.ToUpper(strings.TrimSpace(protocolCode))
	if proto == "" {
		return nil, ErrNotFound
	}
	var out domain.TrackableLinkClick
	err := r.pool.QueryRow(ctx, `
		SELECT id, account_id, pixel_id, link_id, tracked_user_id,
		       resolved_instance_id, protocol_code, client_ip,
		       is_bot, matched_chat_id, click_event_sent, message_event_sent, created_at
		FROM whatsapp_link_clicks
		WHERE account_id = $1 AND protocol_code = $2
		ORDER BY id DESC
		LIMIT 1
	`, accountID, proto).Scan(
		&out.ID, &out.AccountID, &out.PixelID, &out.LinkID, &out.TrackedUserID,
		&out.ResolvedInstanceID, &out.ProtocolCode, &out.ClientIP,
		&out.IsBot, &out.MatchedChatID, &out.ClickEventSent, &out.MessageEventSent, &out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// FindUnambiguousRecentClickForInstance busca um clique humano não vinculado nos últimos
// `window` (ex: 2 minutos) para a mesma instância WhatsApp. Se houver 0 ou >=2 cliques
// simultâneos abertos, retorna nil para evitar falso-positivo entre leads concorrentes.
func (r *TrackableLinkRepo) FindUnambiguousRecentClickForInstance(ctx context.Context, instanceID int64, window time.Duration) *domain.TrackableLinkClick {
	if instanceID <= 0 || window <= 0 {
		return nil
	}
	since := time.Now().Add(-window)
	rows, err := r.pool.Query(ctx, `
		SELECT id, account_id, pixel_id, link_id, tracked_user_id,
		       resolved_instance_id, protocol_code, client_ip,
		       is_bot, matched_chat_id, click_event_sent, message_event_sent, created_at
		FROM whatsapp_link_clicks
		WHERE resolved_instance_id = $1
		  AND is_bot = FALSE
		  AND matched_chat_id IS NULL
		  AND created_at >= $2
		ORDER BY created_at DESC
		LIMIT 2
	`, instanceID, since)
	if err != nil {
		return nil
	}
	defer rows.Close()

	found := make([]domain.TrackableLinkClick, 0, 2)
	for rows.Next() {
		var c domain.TrackableLinkClick
		if err := rows.Scan(
			&c.ID, &c.AccountID, &c.PixelID, &c.LinkID, &c.TrackedUserID,
			&c.ResolvedInstanceID, &c.ProtocolCode, &c.ClientIP,
			&c.IsBot, &c.MatchedChatID, &c.ClickEventSent, &c.MessageEventSent, &c.CreatedAt,
		); err != nil {
			return nil
		}
		found = append(found, c)
	}
	if len(found) != 1 {
		return nil
	}
	return &found[0]
}

func (r *TrackableLinkRepo) AttachChatToClick(ctx context.Context, clickID, chatID int64) error {
	if clickID <= 0 || chatID <= 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE whatsapp_link_clicks
		SET matched_chat_id = COALESCE(matched_chat_id, $2)
		WHERE id = $1
	`, clickID, chatID)
	return err
}

// TryClaimMessageEventDispatch marca atomicamente message_event_sent = TRUE para garantir
// disparo único do evento de 1ª mensagem mesmo sob reprocessamento da fila Asynq.
func (r *TrackableLinkRepo) TryClaimMessageEventDispatch(ctx context.Context, clickID int64) bool {
	if clickID <= 0 {
		return false
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE whatsapp_link_clicks
		SET message_event_sent = TRUE
		WHERE id = $1 AND message_event_sent = FALSE
	`, clickID)
	return err == nil && tag.RowsAffected() == 1
}

// ListInstancesForPixel retorna as instâncias WhatsApp habilitadas vinculadas ao Pixel,
// ordenadas por ID crescente para determinismo no Round-Robin.
func (r *TrackableLinkRepo) ListInstancesForPixel(ctx context.Context, pixelID int64) ([]domain.WhatsAppInstance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, account_id, pixel_id, name, owner_jid, connection_status, enabled
		FROM whatsapp_instances
		WHERE pixel_id = $1 AND enabled = TRUE
		ORDER BY id ASC
	`, pixelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.WhatsAppInstance, 0, 4)
	for rows.Next() {
		var i domain.WhatsAppInstance
		if err := rows.Scan(&i.ID, &i.AccountID, &i.PixelID, &i.Name, &i.OwnerJID, &i.ConnectionStatus, &i.Enabled); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
