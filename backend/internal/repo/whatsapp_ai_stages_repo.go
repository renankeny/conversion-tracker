package repo

import "context"

// ResolveTargetStages resolve as etapas de Qualificado e Venda no Kanban da conta.
// Primeiro resolve a etapa de Venda; depois resolve a etapa de Qualificado com fallback
// inteligente para a 1ª coluna intermediária (ex: "Em Contato", "Interessado"), evitando
// que contas com funil padrão fiquem com qualID = 0 sem mover cards.
func (r *WhatsAppAIRepo) ResolveTargetStages(ctx context.Context, accountID int64, cfgQualID, cfgSaleID *int64) (qualID, saleID int64) {
	if cfgSaleID != nil && *cfgSaleID > 0 {
		_ = r.pool.QueryRow(ctx, `SELECT id FROM crm_stages WHERE id = $1 AND account_id = $2`, *cfgSaleID, accountID).Scan(&saleID)
	}
	if saleID == 0 {
		_ = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_stages
			WHERE account_id = $1
			  AND (
			    LOWER(trigger_event) = 'purchase'
			    OR LOWER(trigger_event_ctwa) = 'purchase'
			    OR name ILIKE '%venda%' OR name ILIKE '%fechad%' OR name ILIKE '%ganh%' OR name ILIKE '%comprou%'
			  )
			ORDER BY order_index DESC, id DESC LIMIT 1
		`, accountID).Scan(&saleID)
	}

	if cfgQualID != nil && *cfgQualID > 0 {
		_ = r.pool.QueryRow(ctx, `SELECT id FROM crm_stages WHERE id = $1 AND account_id = $2`, *cfgQualID, accountID).Scan(&qualID)
	}
	if qualID == 0 {
		_ = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_stages
			WHERE account_id = $1
			  AND id <> $2
			  AND order_index > 0
			  AND LOWER(COALESCE(trigger_event, '')) <> 'purchase'
			  AND LOWER(COALESCE(trigger_event_ctwa, '')) <> 'purchase'
			  AND (
			    LOWER(trigger_event) IN ('qualifiedlead', 'lead', 'leadsubmitted')
			    OR LOWER(trigger_event_ctwa) IN ('qualifiedlead', 'lead', 'leadsubmitted')
			    OR name ILIKE '%qualific%'
			    OR name ILIKE '%interess%'
			    OR name ILIKE '%atendiment%'
			    OR name ILIKE '%em contato%'
			    OR name ILIKE '%negoci%'
			  )
			ORDER BY
				CASE
					WHEN name ILIKE '%qualific%' OR LOWER(trigger_event) = 'qualifiedlead' OR LOWER(trigger_event_ctwa) = 'qualifiedlead' THEN 0
					ELSE 1
				END ASC,
				order_index ASC, id ASC
			LIMIT 1
		`, accountID, saleID).Scan(&qualID)
	}
	if qualID == 0 {
		_ = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_stages
			WHERE account_id = $1
			  AND id <> $2
			  AND order_index > 0
			  AND LOWER(COALESCE(trigger_event, '')) <> 'purchase'
			  AND LOWER(COALESCE(trigger_event_ctwa, '')) <> 'purchase'
			  AND name NOT ILIKE '%venda%'
			  AND name NOT ILIKE '%fechad%'
			  AND name NOT ILIKE '%ganh%'
			  AND name NOT ILIKE '%perdid%'
			  AND name NOT ILIKE '%cancel%'
			  AND name NOT ILIKE '%desqualif%'
			ORDER BY order_index ASC, id ASC
			LIMIT 1
		`, accountID, saleID).Scan(&qualID)
	}
	return qualID, saleID
}

type CRMLeadSnap struct {
	LeadID     int64
	StageID    int64
	StageOrder int
	IsWonStage bool
	Value      float64
}

func (r *WhatsAppAIRepo) GetCRMLeadSnapshot(ctx context.Context, accountID, trackedUserID int64) CRMLeadSnap {
	var snap CRMLeadSnap
	_ = r.pool.QueryRow(ctx, `
		SELECT l.id, COALESCE(l.stage_id, 0), COALESCE(s.order_index, 0),
		       (
		         LOWER(COALESCE(s.trigger_event, '')) = 'purchase'
		         OR LOWER(COALESCE(s.trigger_event_ctwa, '')) = 'purchase'
		         OR COALESCE(s.name, '') ILIKE '%venda%'
		         OR COALESCE(s.name, '') ILIKE '%fechad%'
		         OR COALESCE(s.name, '') ILIKE '%ganh%'
		       ),
		       COALESCE(l.value, 0)
		FROM crm_leads l
		LEFT JOIN crm_stages s ON s.id = l.stage_id
		WHERE l.account_id = $1
		  AND (
		    l.tracked_user_id = $2
		    OR (l.group_id IS NOT NULL AND l.group_id = (SELECT group_id FROM tracked_users WHERE id = $2 AND account_id = $1))
		  )
		ORDER BY l.updated_at DESC LIMIT 1
	`, accountID, trackedUserID).Scan(&snap.LeadID, &snap.StageID, &snap.StageOrder, &snap.IsWonStage, &snap.Value)
	return snap
}

func (r *WhatsAppAIRepo) GetStageOrder(ctx context.Context, accountID, stageID int64) int {
	var ord int
	_ = r.pool.QueryRow(ctx, `SELECT COALESCE(order_index, 0) FROM crm_stages WHERE id = $1 AND account_id = $2`, stageID, accountID).Scan(&ord)
	return ord
}

func (r *WhatsAppAIRepo) RevertCRMLeadStage(ctx context.Context, accountID, leadID int64, currStageID, prevStageID *int64, prevValue float64) bool {
	if leadID <= 0 || prevStageID == nil || *prevStageID <= 0 {
		return false
	}
	if currStageID != nil && *currStageID > 0 {
		tag, err := r.pool.Exec(ctx, `
			UPDATE crm_leads SET stage_id = $3, value = $4, updated_at = NOW()
			WHERE id = $1 AND account_id = $2 AND stage_id = $5
		`, leadID, accountID, *prevStageID, prevValue, *currStageID)
		return err == nil && tag.RowsAffected() > 0
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE crm_leads SET stage_id = $3, value = $4, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
	`, leadID, accountID, *prevStageID, prevValue)
	return err == nil && tag.RowsAffected() > 0
}
