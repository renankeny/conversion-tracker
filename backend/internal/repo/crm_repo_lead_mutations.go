package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// CreateOrUpdateLeadAPI evita duplicar cards para o mesmo cliente (group_id) e retorna a etapa anterior (nil se recém-criado).
func (r *CRMRepo) CreateOrUpdateLeadAPI(ctx context.Context, accountID, trackedUserID, stageID int64, value float64) (int64, *int64, error) {
	groupID, err := r.groupIDOf(ctx, trackedUserID)
	if err != nil {
		return 0, nil, err
	}

	if groupID != nil {
		var existingID int64
		var prevStageID *int64
		err := r.pool.QueryRow(ctx, `
			SELECT id, stage_id
			FROM crm_leads
			WHERE account_id = $1 AND group_id = $2
			LIMIT 1
		`, accountID, *groupID).Scan(&existingID, &prevStageID)
		if err == nil {
			_, err = r.pool.Exec(ctx, `
				UPDATE crm_leads
				SET stage_id = $3, value = $4, group_id = $5, updated_at = NOW()
				WHERE id = $1 AND account_id = $2
			`, existingID, accountID, stageID, value, groupID)
			return existingID, prevStageID, err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, nil, err
		}
	}

	var prevStageID *int64
	var existingByUser int64
	if err := r.pool.QueryRow(ctx, `
		SELECT id, stage_id
		FROM crm_leads
		WHERE account_id = $1 AND tracked_user_id = $2
	`, accountID, trackedUserID).Scan(&existingByUser, &prevStageID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, err
	}

	var id int64
	err = r.pool.QueryRow(ctx, `
		INSERT INTO crm_leads (account_id, tracked_user_id, stage_id, value, group_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (account_id, tracked_user_id)
		DO UPDATE SET
			stage_id = EXCLUDED.stage_id,
			value = EXCLUDED.value,
			group_id = EXCLUDED.group_id,
			updated_at = NOW()
		RETURNING id
	`, accountID, trackedUserID, stageID, value, groupID).Scan(&id)
	return id, prevStageID, err
}

func (r *CRMRepo) UpdateLeadStageAndValue(ctx context.Context, leadID, accountID, stageID int64, value float64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE crm_leads
		SET stage_id = $3, value = $4, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
	`, leadID, accountID, stageID, value)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateLeadPIIAndValue atualiza valor e contato preservando capitalização natural e normalizando telefone E.164 (DDI 55).
func (r *CRMRepo) UpdateLeadPIIAndValue(
	ctx context.Context,
	leadID, accountID int64,
	firstName, lastName, email, phone string,
	value float64,
) (externalID, normPhone, normEmail string, err error) {
	normEmail = util.NormalizeEmail(email)
	normPhone = util.NormalizePhone(phone)
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)

	var trackedUserID int64
	err = r.pool.QueryRow(ctx, `
		UPDATE crm_leads
		SET value = $3, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING tracked_user_id
	`, leadID, accountID, value).Scan(&trackedUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", ErrNotFound
	}
	if err != nil {
		return "", "", "", err
	}

	err = r.pool.QueryRow(ctx, `
		UPDATE tracked_users
		SET first_name = $2, last_name = $3, email = $4, phone = $5, updated_at = NOW()
		WHERE id = $1 AND account_id = $6
		RETURNING COALESCE(external_id, '')
	`, trackedUserID, firstName, lastName, normEmail, normPhone, accountID).Scan(&externalID)
	return externalID, normPhone, normEmail, err
}

func (r *CRMRepo) DeleteLead(ctx context.Context, id, accountID int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM crm_leads WHERE id = $1 AND account_id = $2", id, accountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SyncLeadFromEventTrigger sincroniza automaticamente a etapa e o valor do card no CRM
// quando um gatilho de conversão (ex: Purchase, Lead) é acionado via WhatsApp.
func (r *CRMRepo) SyncLeadFromEventTrigger(
	ctx context.Context,
	accountID, trackedUserID int64,
	eventName string,
	value *float64,
) (int64, bool) {
	if accountID <= 0 || trackedUserID <= 0 {
		return 0, false
	}

	var leadID int64
	groupID, _ := r.groupIDOf(ctx, trackedUserID)
	if groupID != nil && *groupID > 0 {
		_ = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_leads
			WHERE account_id = $1 AND (group_id = $2 OR tracked_user_id = $3)
			ORDER BY updated_at DESC LIMIT 1
		`, accountID, *groupID, trackedUserID).Scan(&leadID)
	} else {
		_ = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_leads
			WHERE account_id = $1 AND tracked_user_id = $2
			LIMIT 1
		`, accountID, trackedUserID).Scan(&leadID)
	}
	if leadID == 0 {
		return 0, false
	}

	var targetStageID int64
	evNorm := strings.TrimSpace(eventName)
	if strings.EqualFold(evNorm, "Lead") || strings.EqualFold(evNorm, "LeadSubmitted") {
		_ = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_stages
			WHERE account_id = $1 AND LOWER(trigger_event) IN ('lead', 'leadsubmitted')
			ORDER BY order_index ASC, id ASC LIMIT 1
		`, accountID).Scan(&targetStageID)
	} else if evNorm != "" {
		_ = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_stages
			WHERE account_id = $1 AND LOWER(trigger_event) = LOWER($2)
			ORDER BY order_index ASC, id ASC LIMIT 1
		`, accountID, evNorm).Scan(&targetStageID)
	}

	val := 0.0
	if value != nil && *value > 0 {
		val = *value
	}
	if targetStageID == 0 && val <= 0 {
		return leadID, false
	}

	tag, err := r.pool.Exec(ctx, `
		UPDATE crm_leads
		SET stage_id   = CASE WHEN $3 > 0 THEN $3 ELSE stage_id END,
		    value      = CASE WHEN $4 > 0 THEN $4 ELSE value END,
		    updated_at = NOW()
		WHERE id = $1 AND account_id = $2
	`, leadID, accountID, targetStageID, val)
	if err != nil || tag.RowsAffected() == 0 {
		return leadID, false
	}
	return leadID, true
}


