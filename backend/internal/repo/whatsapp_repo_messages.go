package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

const selectMessageColumns = `
	m.id, m.chat_id, m.whatsapp_message_id, m.is_from_me, m.message_type,
	m.content, m.has_media, m.media_url, m.media_mime_type,
	COALESCE(m.media_object_key, ''), m.media_status, m.media_size_bytes,
	m.media_duration_sec, m.media_filename, m.media_expires_at,
	COALESCE(m.ai_transcript, ''), COALESCE(m.ctwa_clid, ''), COALESCE(m.source_id, ''),
	COALESCE(m.protocol_code, ''), m.timestamp, m.created_at
`

func scanWhatsAppMessage(scanner pgx.Row, m *domain.WhatsAppMessage) error {
	return scanner.Scan(
		&m.ID, &m.ChatID, &m.WhatsAppMessageID, &m.IsFromMe, &m.MessageType,
		&m.Content, &m.HasMedia, &m.MediaUrl, &m.MediaMimeType,
		&m.MediaObjectKey, &m.MediaStatus, &m.MediaSizeBytes,
		&m.MediaDurationSec, &m.MediaFilename, &m.MediaExpiresAt,
		&m.AITranscript, &m.CtwaClid, &m.SourceID,
		&m.ProtocolCode, &m.Timestamp, &m.CreatedAt,
	)
}

func (r *WhatsAppRepo) ListMessages(ctx context.Context, chatID int64, limit int) ([]*domain.WhatsAppMessage, error) {
	msgs, _, _, err := r.ListMessagesPaginated(ctx, chatID, 0, limit)
	return msgs, err
}

// ListMessagesPaginated lista mensagens com cursor antes de beforeID (paginação de histórico para cima).
// Retorna os itens em ordem cronológica (ASC), hasMore se há mais mensagens anteriores, e nextBeforeID.
func (r *WhatsAppRepo) ListMessagesPaginated(ctx context.Context, chatID int64, beforeID int64, limit int) ([]*domain.WhatsAppMessage, bool, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	fetchLimit := limit + 1
	var rows pgx.Rows
	var err error
	if beforeID > 0 {
		rows, err = r.db.Query(ctx, `
			SELECT * FROM (
				SELECT `+selectMessageColumns+`
				FROM whatsapp_messages m
				WHERE m.chat_id = $1
				  AND (
				    m.timestamp < (SELECT timestamp FROM whatsapp_messages WHERE id = $2)
				    OR (m.timestamp = (SELECT timestamp FROM whatsapp_messages WHERE id = $2) AND m.id < $2)
				  )
				ORDER BY m.timestamp DESC, m.id DESC
				LIMIT $3
			) sub
			ORDER BY sub.timestamp ASC, sub.id ASC
		`, chatID, beforeID, fetchLimit)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT * FROM (
				SELECT `+selectMessageColumns+`
				FROM whatsapp_messages m
				WHERE m.chat_id = $1
				ORDER BY m.timestamp DESC, m.id DESC
				LIMIT $2
			) sub
			ORDER BY sub.timestamp ASC, sub.id ASC
		`, chatID, fetchLimit)
	}
	if err != nil {
		return nil, false, 0, err
	}
	defer rows.Close()

	items := make([]*domain.WhatsAppMessage, 0, fetchLimit)
	for rows.Next() {
		var m domain.WhatsAppMessage
		if err := scanWhatsAppMessage(rows, &m); err != nil {
			return nil, false, 0, err
		}
		items = append(items, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, 0, err
	}

	hasMore := false
	var nextBeforeID int64
	if len(items) > limit {
		hasMore = true
		items = items[len(items)-limit:]
		nextBeforeID = items[0].ID
	}
	return items, hasMore, nextBeforeID, nil
}


// CreateMessage insere uma mensagem com idempotência por (chat_id, whatsapp_message_id).
// Retorna inserted=true apenas quando a linha foi efetivamente criada.
func (r *WhatsAppRepo) CreateMessage(ctx context.Context, msg *domain.WhatsAppMessage) (bool, error) {
	if msg.MediaStatus == "" {
		msg.MediaStatus = domain.MediaStatusNone
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO whatsapp_messages (
			chat_id, whatsapp_message_id, from_me, is_from_me, message_type,
			content, has_media, media_url, media_mime_type,
			media_status, media_size_bytes, media_duration_sec, media_filename, media_expires_at,
			ctwa_clid, source_id, protocol_code, timestamp
		)
		VALUES ($1,$2,$3,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (chat_id, whatsapp_message_id) DO NOTHING
		RETURNING id, created_at
	`,
		msg.ChatID, msg.WhatsAppMessageID, msg.IsFromMe, msg.MessageType,
		msg.Content, msg.HasMedia, msg.MediaUrl, msg.MediaMimeType,
		msg.MediaStatus, msg.MediaSizeBytes, msg.MediaDurationSec, msg.MediaFilename, msg.MediaExpiresAt,
		msg.CtwaClid, msg.SourceID, msg.ProtocolCode, msg.Timestamp,
	).Scan(&msg.ID, &msg.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *WhatsAppRepo) UpdateMessageMediaReady(ctx context.Context, messageID int64, objectKey, mimeType string, sizeBytes int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE whatsapp_messages
		SET media_object_key = $1,
		    media_mime_type  = COALESCE(NULLIF($2, ''), media_mime_type),
		    media_size_bytes = CASE WHEN $3 > 0 THEN $3 ELSE media_size_bytes END,
		    media_status     = 'ready'
		WHERE id = $4
	`, objectKey, mimeType, sizeBytes, messageID)
	return err
}

func (r *WhatsAppRepo) UpdateMessageMediaStatus(ctx context.Context, messageID int64, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE whatsapp_messages
		SET media_status = $1
		WHERE id = $2
	`, status, messageID)
	return err
}

// GetMessageForMediaStream valida estritamente que a mensagem pertence à conta (account_id)
// antes de abrir o stream do objeto privado no MinIO.
func (r *WhatsAppRepo) GetMessageForMediaStream(ctx context.Context, accountID, messageID int64) (*domain.WhatsAppMessage, error) {
	var m domain.WhatsAppMessage
	row := r.db.QueryRow(ctx, `
		SELECT `+selectMessageColumns+`
		FROM whatsapp_messages m
		JOIN whatsapp_chats c ON m.chat_id = c.id
		JOIN whatsapp_instances i ON c.instance_id = i.id
		WHERE m.id = $1 AND i.account_id = $2
	`, messageID, accountID)
	if err := scanWhatsAppMessage(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *WhatsAppRepo) CountInboundMessages(ctx context.Context, chatID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM whatsapp_messages
		WHERE chat_id = $1 AND is_from_me = FALSE
	`, chatID).Scan(&count)
	return count, err
}

func (r *WhatsAppRepo) SaveMessageAITranscript(ctx context.Context, messageID int64, transcript string) error {
	if messageID <= 0 || transcript == "" {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE whatsapp_messages SET ai_transcript = $1 WHERE id = $2
	`, transcript, messageID)
	return err
}
