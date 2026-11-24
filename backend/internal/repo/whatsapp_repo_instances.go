package repo

import (
	"context"
	"encoding/json"
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
	COALESCE(i.event_triggers, '[]'::jsonb), COALESCE(i.average_ticket, 0),
	i.auto_create_crm_lead, i.default_crm_stage_id, COALESCE(i.ctwa_min_inbound_messages, 1), i.enabled,
	i.created_at, i.updated_at,
	COALESCE(s.url, ''), COALESCE(s.api_key, '')
`

func scanWhatsAppInstance(scanner pgx.Row, i *domain.WhatsAppInstance) error {
	var encInstToken, encServerKey string
	var rawTriggers []byte
	err := scanner.Scan(
		&i.ID, &i.AccountID, &i.PixelID, &i.PixelName, &i.ServerID,
		&i.Provider, &i.ExternalAPIURL, &i.Currency,
		&i.Name, &i.InstanceName, &i.EvolutionUUID, &encInstToken, &i.WebhookSecret,
		&i.OwnerJID, &i.ProfileName, &i.ProfilePicURL,
		&i.ShareToken, &i.ConnectionStatus,
		&i.PurchaseTrigger, &i.LeadTrigger, &i.CustomEventName, &i.CustomEventTrigger,
		&rawTriggers, &i.AverageTicket,
		&i.AutoCreateCRMLead, &i.DefaultCRMStageID, &i.CtwaMinInboundMessages, &i.Enabled,
		&i.CreatedAt, &i.UpdatedAt,
		&i.ServerURL, &encServerKey,
	)
	if err != nil {
		return err
	}
	i.Provider = domain.NormalizeWhatsAppProvider(i.Provider)
	i.Currency = domain.NormalizeWhatsAppCurrency(i.Currency)
	i.CtwaMinInboundMessages = domain.NormalizeCtwaMinMessages(i.CtwaMinInboundMessages)
	if len(rawTriggers) > 0 {
		_ = json.Unmarshal(rawTriggers, &i.EventTriggers)
	}
	i.EventTriggers = i.EffectiveEventTriggers()
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

func marshalInstanceTriggers(inst *domain.WhatsAppInstance) ([]byte, error) {
	if inst.EventTriggers == nil {
		inst.EventTriggers = inst.EffectiveEventTriggers()
	} else {
		inst.EventTriggers = domain.NormalizeWhatsAppEventTriggers(inst.EventTriggers)
	}
	return json.Marshal(inst.EventTriggers)
}

func (r *WhatsAppRepo) CreateInstance(ctx context.Context, inst *domain.WhatsAppInstance) error {
	encTok, err := util.Encrypt(strings.TrimSpace(inst.InstanceToken))
	if err != nil {
		return err
	}
	if encTok != "" {
		inst.HasInstanceToken = true
	}
	inst.Provider = domain.NormalizeWhatsAppProvider(inst.Provider)
	inst.Currency = domain.NormalizeWhatsAppCurrency(inst.Currency)
	inst.CtwaMinInboundMessages = domain.NormalizeCtwaMinMessages(inst.CtwaMinInboundMessages)
	if inst.AverageTicket < 0 {
		inst.AverageTicket = 0
	}
	triggersJSON, err := marshalInstanceTriggers(inst)
	if err != nil {
		return err
	}
	var srvID *int64
	if inst.ServerID > 0 {
		srvID = &inst.ServerID
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO whatsapp_instances (
			account_id, pixel_id, server_id, provider, external_api_url, currency,
			name, instance_name, evolution_uuid, instance_token_encrypted, webhook_secret,
			share_token, connection_status,
			purchase_trigger, lead_trigger, custom_event_name, custom_event_trigger, event_triggers, average_ticket,
			auto_create_crm_lead, default_crm_stage_id, ctwa_min_inbound_messages, enabled
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		RETURNING id, created_at, updated_at
	`,
		inst.AccountID, inst.PixelID, srvID, inst.Provider, strings.TrimSpace(inst.ExternalAPIURL), inst.Currency,
		inst.Name, inst.InstanceName, inst.EvolutionUUID, encTok, inst.WebhookSecret,
		inst.ShareToken, inst.ConnectionStatus,
		inst.PurchaseTrigger, inst.LeadTrigger, inst.CustomEventName, inst.CustomEventTrigger, triggersJSON, inst.AverageTicket,
		inst.AutoCreateCRMLead, inst.DefaultCRMStageID, inst.CtwaMinInboundMessages, inst.Enabled,
	).Scan(&inst.ID, &inst.CreatedAt, &inst.UpdatedAt)
}

func (r *WhatsAppRepo) UpdateInstance(ctx context.Context, inst *domain.WhatsAppInstance) error {
	inst.Provider = domain.NormalizeWhatsAppProvider(inst.Provider)
	inst.Currency = domain.NormalizeWhatsAppCurrency(inst.Currency)
	inst.CtwaMinInboundMessages = domain.NormalizeCtwaMinMessages(inst.CtwaMinInboundMessages)
	if inst.AverageTicket < 0 {
		inst.AverageTicket = 0
	}
	encTok, err := util.Encrypt(strings.TrimSpace(inst.InstanceToken))
	if err != nil {
		return err
	}
	if encTok != "" {
		inst.HasInstanceToken = true
	}
	triggersJSON, err := marshalInstanceTriggers(inst)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE whatsapp_instances
		SET name = $1, pixel_id = $2, provider = $3, external_api_url = $4, currency = $5,
		    instance_token_encrypted = CASE WHEN $6 <> '' THEN $6 ELSE instance_token_encrypted END,
		    purchase_trigger = $7, lead_trigger = $8,
		    custom_event_name = $9, custom_event_trigger = $10, event_triggers = $11, average_ticket = $12,
		    auto_create_crm_lead = $13, default_crm_stage_id = $14,
		    ctwa_min_inbound_messages = $15, enabled = $16, updated_at = NOW()
		WHERE id = $17 AND account_id = $18
	`,
		inst.Name, inst.PixelID, inst.Provider, strings.TrimSpace(inst.ExternalAPIURL), inst.Currency, encTok,
		inst.PurchaseTrigger, inst.LeadTrigger,
		inst.CustomEventName, inst.CustomEventTrigger, triggersJSON, inst.AverageTicket,
		inst.AutoCreateCRMLead, inst.DefaultCRMStageID,
		inst.CtwaMinInboundMessages, inst.Enabled, inst.ID, inst.AccountID,
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
