package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

const selectChatColumns = `
	c.id, COALESCE(c.account_id, i.account_id), c.instance_id, i.name,
	c.remote_jid, c.tracked_user_id, c.crm_lead_id,
	c.phone, c.lid_jid, c.push_name, c.contact_name, c.profile_pic,
	c.origin_type, c.ctwa_clid, c.source_id, c.page_id, c.source_app,
	c.ad_name, c.adset_id, c.adset_name, c.campaign_id, c.campaign_name, c.ad_thumbnail_url,
	c.unread_count, c.last_message_time, c.metadata, c.created_at, c.updated_at
`

func scanWhatsAppChat(scanner pgx.Row, c *domain.WhatsAppChat) error {
	if err := scanner.Scan(
		&c.ID, &c.AccountID, &c.InstanceID, &c.InstanceName,
		&c.RemoteJid, &c.TrackedUserID, &c.CRMLeadID,
		&c.Phone, &c.LidJid, &c.PushName, &c.ContactName, &c.ProfilePic,
		&c.OriginType, &c.CtwaClid, &c.SourceID, &c.PageID, &c.SourceApp,
		&c.AdName, &c.AdsetID, &c.AdsetName, &c.CampaignID, &c.CampaignName, &c.AdThumbnailURL,
		&c.UnreadCount, &c.LastMessageTime, &c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return err
	}
	c.Source = c.OriginType
	return nil
}

// FindExistingChatForContact localiza uma conversa ativa já rastreada nesta instância
// por remote_jid, telefone normalizado ou lid_jid.
func (r *WhatsAppRepo) FindExistingChatForContact(ctx context.Context, instanceID int64, remoteJid, phone, lidJid string) (*domain.WhatsAppChat, error) {
	var c domain.WhatsAppChat
	row := r.db.QueryRow(ctx, `
		SELECT `+selectChatColumns+`
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		WHERE c.instance_id = $1
		  AND (
		    c.remote_jid = $2
		    OR ($3 <> '' AND c.phone = $3)
		    OR ($4 <> '' AND c.lid_jid = $4)
		  )
		ORDER BY c.last_message_time DESC
		LIMIT 1
	`, instanceID, remoteJid, phone, lidJid)
	if err := scanWhatsAppChat(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *WhatsAppRepo) UpsertActiveChat(ctx context.Context, c *domain.WhatsAppChat, incrementUnread bool) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO whatsapp_chats (
			account_id, instance_id, remote_jid, tracked_user_id, crm_lead_id,
			phone, lid_jid, push_name, contact_name, origin_type,
			ctwa_clid, source_id, page_id, source_app, ad_thumbnail_url,
			last_message_time, unread_count
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16, CASE WHEN $17::boolean THEN 1 ELSE 0 END)
		ON CONFLICT (instance_id, remote_jid) DO UPDATE SET
			account_id        = COALESCE(EXCLUDED.account_id, whatsapp_chats.account_id),
			tracked_user_id   = COALESCE(EXCLUDED.tracked_user_id, whatsapp_chats.tracked_user_id),
			crm_lead_id       = COALESCE(EXCLUDED.crm_lead_id, whatsapp_chats.crm_lead_id),
			phone             = COALESCE(NULLIF(EXCLUDED.phone, ''), whatsapp_chats.phone),
			lid_jid           = COALESCE(NULLIF(EXCLUDED.lid_jid, ''), whatsapp_chats.lid_jid),
			push_name         = COALESCE(NULLIF(EXCLUDED.push_name, ''), whatsapp_chats.push_name),
			contact_name      = COALESCE(EXCLUDED.contact_name, whatsapp_chats.contact_name),
			origin_type       = CASE WHEN EXCLUDED.origin_type NOT IN ('', 'unknown') THEN EXCLUDED.origin_type ELSE whatsapp_chats.origin_type END,
			ctwa_clid         = COALESCE(NULLIF(EXCLUDED.ctwa_clid, ''), whatsapp_chats.ctwa_clid),
			source_id         = COALESCE(NULLIF(EXCLUDED.source_id, ''), whatsapp_chats.source_id),
			page_id           = COALESCE(NULLIF(EXCLUDED.page_id, ''), whatsapp_chats.page_id),
			source_app        = COALESCE(NULLIF(EXCLUDED.source_app, ''), whatsapp_chats.source_app),
			ad_thumbnail_url  = COALESCE(NULLIF(EXCLUDED.ad_thumbnail_url, ''), whatsapp_chats.ad_thumbnail_url),
			last_message_time = EXCLUDED.last_message_time,
			unread_count      = CASE WHEN $17::boolean THEN whatsapp_chats.unread_count + 1 ELSE whatsapp_chats.unread_count END,
			updated_at        = NOW()
		RETURNING id
	`,
		c.AccountID, c.InstanceID, c.RemoteJid, c.TrackedUserID, c.CRMLeadID,
		c.Phone, c.LidJid, c.PushName, c.ContactName, c.OriginType,
		c.CtwaClid, c.SourceID, c.PageID, c.SourceApp, c.AdThumbnailURL,
		c.LastMessageTime, incrementUnread,
	).Scan(&id)
	return id, err
}

func (r *WhatsAppRepo) UpdateChatAdEnrichment(ctx context.Context, chatID int64, campaignID, campaignName, adsetID, adsetName, adName, pageID, thumbURL string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE whatsapp_chats
		SET campaign_id      = COALESCE(NULLIF($1, ''), campaign_id),
		    campaign_name    = COALESCE(NULLIF($2, ''), campaign_name),
		    adset_id         = COALESCE(NULLIF($3, ''), adset_id),
		    adset_name       = COALESCE(NULLIF($4, ''), adset_name),
		    ad_name          = COALESCE(NULLIF($5, ''), ad_name),
		    page_id          = COALESCE(NULLIF($6, ''), page_id),
		    ad_thumbnail_url = COALESCE(NULLIF($7, ''), ad_thumbnail_url),
		    updated_at       = NOW()
		WHERE id = $8
	`, campaignID, campaignName, adsetID, adsetName, adName, pageID, thumbURL, chatID)
	return err
}

func (r *WhatsAppRepo) ListChats(ctx context.Context, accountID int64) ([]*domain.WhatsAppChat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+selectChatColumns+`
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		WHERE i.account_id = $1 AND c.tracked_user_id IS NOT NULL
		ORDER BY c.last_message_time DESC
		LIMIT 200
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*domain.WhatsAppChat, 0)
	for rows.Next() {
		var c domain.WhatsAppChat
		if err := scanWhatsAppChat(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}
