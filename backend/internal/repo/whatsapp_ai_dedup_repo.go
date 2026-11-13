package repo

import (
	"context"
	"math"
	"time"
)

type PriorPurchaseRef struct {
	Source    string
	Amount    float64
	MessageID int64
	CreatedAt time.Time
}

func (r *WhatsAppAIRepo) groupExternalIDs(ctx context.Context, accountID, trackedUserID int64) []string {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT u.external_id
		FROM tracked_users u
		WHERE u.account_id = $1
		  AND u.external_id <> ''
		  AND (
		    u.id = $2
		    OR (
		      u.group_id IS NOT NULL
		      AND u.group_id = (SELECT group_id FROM tracked_users WHERE id = $2 AND account_id = $1)
		    )
		  )
	`, accountID, trackedUserID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var ext string
		if rows.Scan(&ext) == nil && ext != "" {
			ids = append(ids, ext)
		}
	}
	return ids
}

// HasLeadConversionSent verifica se Lead, LeadSubmitted ou QualifiedLead já foi enviado
// por qualquer fonte (gatilho de frase, CTWA, pixel ou CRM manual) para esta identidade.
func (r *WhatsAppAIRepo) HasLeadConversionSent(ctx context.Context, accountID, trackedUserID int64, pixelName string) (bool, error) {
	extIDs := r.groupExternalIDs(ctx, accountID, trackedUserID)
	if len(extIDs) == 0 || pixelName == "" {
		return false, nil
	}
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sent_events
			WHERE account_id = $1
			  AND pixel_name = $2
			  AND external_id = ANY($3::text[])
			  AND event_name IN ('Lead', 'LeadSubmitted', 'QualifiedLead')
			  AND status = 'sent'
		)
	`, accountID, pixelName, extIDs).Scan(&exists)
	return exists, err
}

// HasMatchingPurchaseSent evita duplicar uma venda caso o gatilho do atendente,
// um webhook de gateway (Kiwify/Hotmart/Cakto) ou o Kanban já tenha disparado Purchase.
func (r *WhatsAppAIRepo) HasMatchingPurchaseSent(
	ctx context.Context,
	accountID, trackedUserID int64,
	pixelName string,
	amount float64,
	aroundTime time.Time,
) (bool, error) {
	extIDs := r.groupExternalIDs(ctx, accountID, trackedUserID)
	if len(extIDs) == 0 || pixelName == "" {
		return false, nil
	}
	if aroundTime.IsZero() {
		aroundTime = time.Now()
	}
	winStart := aroundTime.Add(-24 * time.Hour)
	winEnd := aroundTime.Add(4 * time.Hour)

	rows, err := r.pool.Query(ctx, `
		SELECT purchase_value, created_at
		FROM sent_events
		WHERE account_id = $1
		  AND pixel_name = $2
		  AND external_id = ANY($3::text[])
		  AND event_name = 'Purchase'
		  AND status = 'sent'
		  AND created_at >= $4 AND created_at <= $5
	`, accountID, pixelName, extIDs, winStart, winEnd)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var sentVal float64
		var sentAt time.Time
		if err := rows.Scan(&sentVal, &sentAt); err != nil {
			continue
		}
		if amount > 0 && sentVal > 0 && math.Abs(sentVal-amount) <= 1.00 {
			return true, nil
		}
		diff := sentAt.Sub(aroundTime)
		if diff < 0 {
			diff = -diff
		}
		if (amount <= 0 || sentVal <= 0) && diff <= 2*time.Hour {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (r *WhatsAppAIRepo) ListPriorPurchasesSummary(ctx context.Context, accountID, chatID, trackedUserID int64) []PriorPurchaseRef {
	var out []PriorPurchaseRef
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(evidence_message_id, 0), amount, created_at
		FROM whatsapp_ai_events
		WHERE chat_id = $1 AND event_type = 'purchase' AND status = 'active'
		ORDER BY id ASC
	`, chatID)
	if err == nil {
		for rows.Next() {
			var p PriorPurchaseRef
			p.Source = "ia"
			if rows.Scan(&p.MessageID, &p.Amount, &p.CreatedAt) == nil {
				out = append(out, p)
			}
		}
		rows.Close()
	}

	extIDs := r.groupExternalIDs(ctx, accountID, trackedUserID)
	if len(extIDs) > 0 {
		sRows, serr := r.pool.Query(ctx, `
			SELECT DISTINCT ON (event_id) purchase_value, created_at
			FROM sent_events
			WHERE account_id = $1 AND external_id = ANY($2::text[])
			  AND event_name = 'Purchase' AND status = 'sent'
			ORDER BY event_id, created_at DESC
			LIMIT 20
		`, accountID, extIDs)
		if serr == nil {
			for sRows.Next() {
				var p PriorPurchaseRef
				p.Source = "sistema"
				if sRows.Scan(&p.Amount, &p.CreatedAt) == nil {
					out = append(out, p)
				}
			}
			sRows.Close()
		}
	}
	return out
}

func (r *WhatsAppAIRepo) ResolveTargetStages(ctx context.Context, accountID int64, cfgQualID, cfgSaleID *int64) (qualID, saleID int64) {
	if cfgQualID != nil && *cfgQualID > 0 {
		_ = r.pool.QueryRow(ctx, `SELECT id FROM crm_stages WHERE id = $1 AND account_id = $2`, *cfgQualID, accountID).Scan(&qualID)
	}
	if qualID == 0 {
		_ = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_stages
			WHERE account_id = $1
			  AND (
			    LOWER(trigger_event) IN ('lead', 'leadsubmitted')
			    OR LOWER(trigger_event_ctwa) IN ('lead', 'leadsubmitted')
			    OR name ILIKE '%qualific%'
			  )
			ORDER BY order_index ASC, id ASC LIMIT 1
		`, accountID).Scan(&qualID)
	}

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
			    OR name ILIKE '%venda%' OR name ILIKE '%fechad%' OR name ILIKE '%ganh%'
			  )
			ORDER BY order_index DESC, id DESC LIMIT 1
		`, accountID).Scan(&saleID)
	}
	return qualID, saleID
}

type CRMLeadSnap struct {
	LeadID     int64
	StageID    int64
	StageOrder int
	Value      float64
}

func (r *WhatsAppAIRepo) GetCRMLeadSnapshot(ctx context.Context, accountID, trackedUserID int64) CRMLeadSnap {
	var snap CRMLeadSnap
	_ = r.pool.QueryRow(ctx, `
		SELECT l.id, l.stage_id, COALESCE(s.order_index, 0), COALESCE(l.value, 0)
		FROM crm_leads l
		LEFT JOIN crm_stages s ON s.id = l.stage_id
		WHERE l.account_id = $1
		  AND (
		    l.tracked_user_id = $2
		    OR (l.group_id IS NOT NULL AND l.group_id = (SELECT group_id FROM tracked_users WHERE id = $2 AND account_id = $1))
		  )
		ORDER BY l.updated_at DESC LIMIT 1
	`, accountID, trackedUserID).Scan(&snap.LeadID, &snap.StageID, &snap.StageOrder, &snap.Value)
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

