package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func (r *WhatsAppRepo) GetChatByID(ctx context.Context, accountID, chatID int64) (*domain.WhatsAppChat, error) {
	var c domain.WhatsAppChat
	row := r.db.QueryRow(ctx, `
		SELECT `+selectChatColumns+`
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		WHERE c.id = $1 AND i.account_id = $2
	`, chatID, accountID)
	if err := scanWhatsAppChat(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// GetChatByTrackedUser busca a conversa mais recente do visitante (incluindo membros do mesmo group_id).
func (r *WhatsAppRepo) GetChatByTrackedUser(ctx context.Context, accountID, trackedUserID int64) (*domain.WhatsAppChat, error) {
	var c domain.WhatsAppChat
	row := r.db.QueryRow(ctx, `
		SELECT `+selectChatColumns+`
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		WHERE i.account_id = $1
		  AND (
		    c.tracked_user_id = $2
		    OR c.tracked_user_id IN (
		      SELECT u2.id FROM tracked_users u1
		      JOIN tracked_users u2 ON u2.group_id = u1.group_id
		      WHERE u1.id = $2 AND u1.group_id IS NOT NULL
		    )
		  )
		ORDER BY c.last_message_time DESC
		LIMIT 1
	`, accountID, trackedUserID)
	if err := scanWhatsAppChat(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *WhatsAppRepo) GetChatByCRMLead(ctx context.Context, accountID, crmLeadID int64) (*domain.WhatsAppChat, error) {
	var c domain.WhatsAppChat
	row := r.db.QueryRow(ctx, `
		SELECT `+selectChatColumns+`
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		WHERE i.account_id = $1
		  AND (
		    c.crm_lead_id = $2
		    OR c.tracked_user_id = (SELECT tracked_user_id FROM crm_leads WHERE id = $2 AND account_id = $1)
		  )
		ORDER BY c.last_message_time DESC
		LIMIT 1
	`, accountID, crmLeadID)
	if err := scanWhatsAppChat(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *WhatsAppRepo) FindPageIDByCtwaClid(ctx context.Context, accountID int64, ctwaClid string) string {
	if ctwaClid == "" {
		return ""
	}
	var pageID string
	_ = r.db.QueryRow(ctx, `
		SELECT c.page_id FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		WHERE i.account_id = $1 AND c.ctwa_clid = $2 AND c.page_id <> ''
		ORDER BY c.updated_at DESC LIMIT 1
	`, accountID, ctwaClid).Scan(&pageID)
	return pageID
}

type ChatConversionContext struct {
	PixelName          string
	MetaPermanentToken string
	ExternalID         string
	Phone              string
	FirstName          string
	CtwaClid           string
	SourceID           string
	PageID             string
}

func (r *WhatsAppRepo) GetChatConversionContext(ctx context.Context, accountID, chatID int64) (*ChatConversionContext, error) {
	var out ChatConversionContext
	var metadata []byte
	err := r.db.QueryRow(ctx, `
		SELECT p.name, p.meta_permanent_token, u.external_id, COALESCE(NULLIF(u.phone, ''), c.phone),
		       u.first_name, COALESCE(NULLIF(c.ctwa_clid, ''), u.ctwa_clid), c.source_id, c.page_id, c.metadata
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		JOIN pixels p ON i.pixel_id = p.id
		JOIN tracked_users u ON c.tracked_user_id = u.id
		WHERE c.id = $1 AND i.account_id = $2
	`, chatID, accountID).Scan(
		&out.PixelName, &out.MetaPermanentToken, &out.ExternalID, &out.Phone,
		&out.FirstName, &out.CtwaClid, &out.SourceID, &out.PageID, &metadata,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if out.SourceID == "" && len(metadata) > 0 {
		var chatMeta map[string]string
		if json.Unmarshal(metadata, &chatMeta) == nil {
			out.SourceID = chatMeta["sourceId"]
		}
	}
	return &out, nil
}

