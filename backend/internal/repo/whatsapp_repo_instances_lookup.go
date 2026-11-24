package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func (r *WhatsAppRepo) getInstanceWhere(ctx context.Context, whereSQL string, args ...any) (*domain.WhatsAppInstance, error) {
	var i domain.WhatsAppInstance
	row := r.db.QueryRow(ctx, `
		SELECT `+selectInstanceColumns+`
		FROM whatsapp_instances i
		LEFT JOIN whatsapp_servers s ON i.server_id = s.id
		LEFT JOIN pixels p ON i.pixel_id = p.id
		WHERE `+whereSQL, args...)
	if err := scanWhatsAppInstance(row, &i); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &i, nil
}

func (r *WhatsAppRepo) GetInstance(ctx context.Context, accountID, instanceID int64) (*domain.WhatsAppInstance, error) {
	return r.getInstanceWhere(ctx, "i.account_id = $1 AND i.id = $2", accountID, instanceID)
}

func (r *WhatsAppRepo) GetInstanceByIDUnscoped(ctx context.Context, instanceID int64) (*domain.WhatsAppInstance, error) {
	return r.getInstanceWhere(ctx, "i.id = $1", instanceID)
}

func (r *WhatsAppRepo) GetInstanceByWebhookSecret(ctx context.Context, secret string) (*domain.WhatsAppInstance, error) {
	if secret == "" {
		return nil, nil
	}
	return r.getInstanceWhere(ctx, "i.webhook_secret = $1", secret)
}

func (r *WhatsAppRepo) GetInstanceByName(ctx context.Context, instanceName string) (*domain.WhatsAppInstance, error) {
	return r.getInstanceWhere(ctx, "i.instance_name = $1", instanceName)
}

func (r *WhatsAppRepo) GetInstanceByShareToken(ctx context.Context, token string) (*domain.WhatsAppInstance, error) {
	return r.getInstanceWhere(ctx, "i.share_token = $1", token)
}

func (r *WhatsAppRepo) DeleteInstance(ctx context.Context, accountID, instanceID int64) error {
	_, err := r.db.Exec(ctx, "DELETE FROM whatsapp_instances WHERE id = $1 AND account_id = $2", instanceID, accountID)
	return err
}

func (r *WhatsAppRepo) UpdateInstanceStatus(ctx context.Context, instanceID int64, status string) error {
	_, err := r.db.Exec(ctx, "UPDATE whatsapp_instances SET connection_status = $1, updated_at = NOW() WHERE id = $2", status, instanceID)
	return err
}

func (r *WhatsAppRepo) ResetInstanceSession(ctx context.Context, instanceID int64, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE whatsapp_instances
		SET connection_status = $1, owner_jid = '', profile_name = '', profile_pic_url = '', updated_at = NOW()
		WHERE id = $2
	`, status, instanceID)
	return err
}

func (r *WhatsAppRepo) EnsureInstanceShareToken(ctx context.Context, instanceID int64, shareToken string) error {
	_, err := r.db.Exec(ctx, "UPDATE whatsapp_instances SET share_token = $1, updated_at = NOW() WHERE id = $2 AND (share_token IS NULL OR share_token = '')", shareToken, instanceID)
	return err
}

func (r *WhatsAppRepo) UpdateInstanceOwnerAndProfile(ctx context.Context, instanceID int64, status, ownerJID, profileName, profilePicURL string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE whatsapp_instances
		SET connection_status = COALESCE(NULLIF($1, ''), connection_status),
		    owner_jid         = COALESCE(NULLIF($2, ''), owner_jid),
		    profile_name      = COALESCE(NULLIF($3, ''), profile_name),
		    profile_pic_url   = COALESCE(NULLIF($4, ''), profile_pic_url),
		    updated_at        = NOW()
		WHERE id = $5
	`, status, ownerJID, profileName, profilePicURL, instanceID)
	return err
}

func (r *WhatsAppRepo) GetChatInstanceAverageTicket(ctx context.Context, chatID int64) float64 {
	if chatID <= 0 {
		return 0
	}
	var avg float64
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(i.average_ticket, 0)
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON i.id = c.instance_id
		WHERE c.id = $1
	`, chatID).Scan(&avg)
	return avg
}

func (r *WhatsAppRepo) UpdateInstanceExternalCredentials(ctx context.Context, instanceID int64, externalURL, rawToken string) error {
	encTok, err := util.Encrypt(strings.TrimSpace(rawToken))
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE whatsapp_instances
		SET external_api_url         = COALESCE(NULLIF(external_api_url, ''), NULLIF($1, ''), external_api_url),
		    instance_token_encrypted = COALESCE(NULLIF(instance_token_encrypted, ''), NULLIF($2, ''), instance_token_encrypted),
		    updated_at               = NOW()
		WHERE id = $3
	`, strings.TrimRight(strings.TrimSpace(externalURL), "/"), encTok, instanceID)
	return err
}
