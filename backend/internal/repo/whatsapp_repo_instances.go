package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const selectInstanceColumns = `
	i.id, i.account_id, i.pixel_id, COALESCE(p.name, ''), COALESCE(i.server_id, 0),
	COALESCE(i.provider, 'evolution_go'), COALESCE(i.external_api_url, ''), COALESCE(i.currency, 'BRL'),
	i.name, i.instance_name, i.evolution_uuid, i.instance_token_encrypted, i.webhook_secret,
	i.owner_jid, i.profile_name, i.profile_pic_url,
	i.share_token, i.connection_status,
	i.purchase_trigger, i.lead_trigger, i.custom_event_name, i.custom_event_trigger,
	i.auto_create_crm_lead, i.default_crm_stage_id, i.enabled,
	i.created_at, i.updated_at,
	COALESCE(s.url, ''), COALESCE(s.api_key, '')
`

func scanWhatsAppInstance(scanner pgx.Row, i *domain.WhatsAppInstance) error {
	var encInstToken, encServerKey string
	err := scanner.Scan(
		&i.ID, &i.AccountID, &i.PixelID, &i.PixelName, &i.ServerID,
		&i.Provider, &i.ExternalAPIURL, &i.Currency,
		&i.Name, &i.InstanceName, &i.EvolutionUUID, &encInstToken, &i.WebhookSecret,
		&i.OwnerJID, &i.ProfileName, &i.ProfilePicURL,
		&i.ShareToken, &i.ConnectionStatus,
		&i.PurchaseTrigger, &i.LeadTrigger, &i.CustomEventName, &i.CustomEventTrigger,
		&i.AutoCreateCRMLead, &i.DefaultCRMStageID, &i.Enabled,
		&i.CreatedAt, &i.UpdatedAt,
		&i.ServerURL, &encServerKey,
	)
	if err != nil {
		return err
	}
	i.Provider = domain.NormalizeWhatsAppProvider(i.Provider)
	i.Currency = domain.NormalizeWhatsAppCurrency(i.Currency)
	if encServerKey != "" {
		i.ServerAPIKey = util.Decrypt(encServerKey)
	}
	if encInstToken != "" {
		i.InstanceToken = util.Decrypt(encInstToken)
		i.HasInstanceToken = strings.TrimSpace(i.InstanceToken) != ""
	} else if !i.IsExternalUazapi() {
		i.InstanceToken = i.InstanceName
	}
	return nil
}

func (r *WhatsAppRepo) CreateInstance(ctx context.Context, inst *domain.WhatsAppInstance) error {
	encTok := ""
	if strings.TrimSpace(inst.InstanceToken) != "" {
		encTok = util.Encrypt(strings.TrimSpace(inst.InstanceToken))
		inst.HasInstanceToken = true
	}
	inst.Provider = domain.NormalizeWhatsAppProvider(inst.Provider)
	inst.Currency = domain.NormalizeWhatsAppCurrency(inst.Currency)
	var srvID *int64
	if inst.ServerID > 0 {
		srvID = &inst.ServerID
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO whatsapp_instances (
			account_id, pixel_id, server_id, provider, external_api_url, currency,
			name, instance_name, evolution_uuid, instance_token_encrypted, webhook_secret,
			share_token, connection_status,
			purchase_trigger, lead_trigger, custom_event_name, custom_event_trigger,
			auto_create_crm_lead, default_crm_stage_id, enabled
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		RETURNING id, created_at, updated_at
	`,
		inst.AccountID, inst.PixelID, srvID, inst.Provider, strings.TrimSpace(inst.ExternalAPIURL), inst.Currency,
		inst.Name, inst.InstanceName, inst.EvolutionUUID, encTok, inst.WebhookSecret,
		inst.ShareToken, inst.ConnectionStatus,
		inst.PurchaseTrigger, inst.LeadTrigger, inst.CustomEventName, inst.CustomEventTrigger,
		inst.AutoCreateCRMLead, inst.DefaultCRMStageID, inst.Enabled,
	).Scan(&inst.ID, &inst.CreatedAt, &inst.UpdatedAt)
}

func (r *WhatsAppRepo) UpdateInstance(ctx context.Context, inst *domain.WhatsAppInstance) error {
	inst.Provider = domain.NormalizeWhatsAppProvider(inst.Provider)
	inst.Currency = domain.NormalizeWhatsAppCurrency(inst.Currency)
	encTok := ""
	if strings.TrimSpace(inst.InstanceToken) != "" {
		encTok = util.Encrypt(strings.TrimSpace(inst.InstanceToken))
		inst.HasInstanceToken = true
	}
	_, err := r.db.Exec(ctx, `
		UPDATE whatsapp_instances
		SET name = $1, pixel_id = $2, provider = $3, external_api_url = $4, currency = $5,
		    instance_token_encrypted = CASE WHEN $6 <> '' THEN $6 ELSE instance_token_encrypted END,
		    purchase_trigger = $7, lead_trigger = $8,
		    custom_event_name = $9, custom_event_trigger = $10,
		    auto_create_crm_lead = $11, default_crm_stage_id = $12,
		    enabled = $13, updated_at = NOW()
		WHERE id = $14 AND account_id = $15
	`,
		inst.Name, inst.PixelID, inst.Provider, strings.TrimSpace(inst.ExternalAPIURL), inst.Currency, encTok,
		inst.PurchaseTrigger, inst.LeadTrigger,
		inst.CustomEventName, inst.CustomEventTrigger,
		inst.AutoCreateCRMLead, inst.DefaultCRMStageID,
		inst.Enabled, inst.ID, inst.AccountID,
	)
	return err
}

func (r *WhatsAppRepo) ListInstances(ctx context.Context, accountID int64) ([]domain.WhatsAppInstance, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+selectInstanceColumns+`
		FROM whatsapp_instances i
		LEFT JOIN whatsapp_servers s ON i.server_id = s.id
		LEFT JOIN pixels p ON i.pixel_id = p.id
		WHERE i.account_id = $1
		ORDER BY i.id DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.WhatsAppInstance, 0)
	for rows.Next() {
		var i domain.WhatsAppInstance
		if err := scanWhatsAppInstance(rows, &i); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *WhatsAppRepo) CountInstances(ctx context.Context, accountID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM whatsapp_instances WHERE account_id = $1", accountID).Scan(&count)
	return count, err
}

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

func (r *WhatsAppRepo) UpdateInstanceExternalCredentials(ctx context.Context, instanceID int64, externalURL, rawToken string) error {
	encTok := ""
	if strings.TrimSpace(rawToken) != "" {
		encTok = util.Encrypt(strings.TrimSpace(rawToken))
	}
	_, err := r.db.Exec(ctx, `
		UPDATE whatsapp_instances
		SET external_api_url         = COALESCE(NULLIF(external_api_url, ''), NULLIF($1, ''), external_api_url),
		    instance_token_encrypted = COALESCE(NULLIF(instance_token_encrypted, ''), NULLIF($2, ''), instance_token_encrypted),
		    updated_at               = NOW()
		WHERE id = $3
	`, strings.TrimRight(strings.TrimSpace(externalURL), "/"), encTok, instanceID)
	return err
}
