package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ResolvePurchaseStageID localiza a etapa de Vendas ("Purchase" / "Venda Fechada") da conta,
// garantindo que as etapas padrão existam caso a empresa ainda não tenha aberto o CRM.
func (r *CRMRepo) ResolvePurchaseStageID(ctx context.Context, accountID int64) int64 {
	if accountID <= 0 {
		return 0
	}
	_ = r.EnsureDefaultStages(ctx, accountID)

	var stageID int64
	_ = r.pool.QueryRow(ctx, `
		SELECT id FROM crm_stages
		WHERE account_id = $1
		  AND (LOWER(trigger_event) = 'purchase' OR LOWER(trigger_event_ctwa) = 'purchase')
		ORDER BY order_index DESC, id DESC LIMIT 1
	`, accountID).Scan(&stageID)
	if stageID > 0 {
		return stageID
	}

	_ = r.pool.QueryRow(ctx, `
		SELECT id FROM crm_stages
		WHERE account_id = $1
		  AND (
		    LOWER(name) LIKE '%venda%'
		    OR LOWER(name) LIKE '%ganho%'
		    OR LOWER(name) LIKE '%fechad%'
		    OR LOWER(name) LIKE '%comprou%'
		  )
		ORDER BY order_index DESC, id DESC LIMIT 1
	`, accountID).Scan(&stageID)
	return stageID
}

// SyncOrCreateLeadOnPurchase move o card existente para a coluna de Vendas com o valor,
// ou cria automaticamente o card na coluna de Vendas caso o comprador ainda não estivesse no CRM.
// Atualiza direto no banco (sem disparar MoveLead HTTP), evitando qualquer reenvio duplicado à Meta.
func (r *CRMRepo) SyncOrCreateLeadOnPurchase(
	ctx context.Context,
	accountID, trackedUserID int64,
	saleValue float64,
	accumulateIfAlreadyWon bool,
) (int64, bool) {
	if accountID <= 0 || trackedUserID <= 0 {
		return 0, false
	}
	targetStageID := r.ResolvePurchaseStageID(ctx, accountID)
	if targetStageID == 0 {
		return 0, false
	}

	groupID, _ := r.groupIDOf(ctx, trackedUserID)

	var existingID int64
	var curStageID *int64
	var curValue float64
	var err error
	if groupID != nil && *groupID > 0 {
		err = r.pool.QueryRow(ctx, `
			SELECT id, stage_id, COALESCE(value, 0)
			FROM crm_leads
			WHERE account_id = $1 AND (group_id = $2 OR tracked_user_id = $3)
			ORDER BY updated_at DESC LIMIT 1
		`, accountID, *groupID, trackedUserID).Scan(&existingID, &curStageID, &curValue)
	} else {
		err = r.pool.QueryRow(ctx, `
			SELECT id, stage_id, COALESCE(value, 0)
			FROM crm_leads
			WHERE account_id = $1 AND tracked_user_id = $2
			LIMIT 1
		`, accountID, trackedUserID).Scan(&existingID, &curStageID, &curValue)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, false
	}

	val := 0.0
	if saleValue > 0 {
		val = saleValue
	}

	if existingID == 0 {
		var newID int64
		err = r.pool.QueryRow(ctx, `
			INSERT INTO crm_leads (account_id, tracked_user_id, stage_id, value, group_id)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (account_id, tracked_user_id)
			DO UPDATE SET
				stage_id   = EXCLUDED.stage_id,
				value      = CASE WHEN EXCLUDED.value > 0 THEN EXCLUDED.value ELSE crm_leads.value END,
				group_id   = COALESCE(EXCLUDED.group_id, crm_leads.group_id),
				updated_at = NOW()
			RETURNING id
		`, accountID, trackedUserID, targetStageID, val, groupID).Scan(&newID)
		if err != nil || newID == 0 {
			return 0, false
		}
		_, _ = r.pool.Exec(ctx, `
			UPDATE whatsapp_chats
			SET crm_lead_id = $1
			WHERE account_id = $2 AND tracked_user_id = $3 AND crm_lead_id IS NULL
		`, newID, accountID, trackedUserID)
		return newID, true
	}

	alreadyInTarget := curStageID != nil && *curStageID == targetStageID
	nextVal := curValue
	if !alreadyInTarget {
		if val > 0 {
			nextVal = val
		}
	} else if accumulateIfAlreadyWon && val > 0 {
		nextVal = curValue + val
	} else if curValue <= 0 && val > 0 {
		nextVal = val
	}

	if alreadyInTarget && nextVal == curValue {
		return existingID, false
	}

	tag, uerr := r.pool.Exec(ctx, `
		UPDATE crm_leads
		SET stage_id   = $3,
		    value      = $4,
		    group_id   = COALESCE($5, group_id),
		    updated_at = NOW()
		WHERE id = $1 AND account_id = $2
	`, existingID, accountID, targetStageID, nextVal, groupID)
	if uerr != nil || tag.RowsAffected() == 0 {
		return existingID, false
	}
	return existingID, true
}
