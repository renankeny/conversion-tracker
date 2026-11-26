package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// ApplyRefundToLead abate o valor estornado no card do CRM Kanban, marca refund_status
// ('refunded' ou 'chargeback') e registra nota automática na timeline do lead.
func (r *CRMRepo) ApplyRefundToLead(
	ctx context.Context,
	accountID, trackedUserID int64,
	refundType string,
	amount float64,
	noteText string,
) (int64, bool) {
	if accountID <= 0 || trackedUserID <= 0 {
		return 0, false
	}
	groupID, _ := r.groupIDOf(ctx, trackedUserID)

	var leadID int64
	var err error
	if groupID != nil && *groupID > 0 {
		err = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_leads
			WHERE account_id = $1 AND (group_id = $2 OR tracked_user_id = $3)
			ORDER BY updated_at DESC LIMIT 1
		`, accountID, *groupID, trackedUserID).Scan(&leadID)
	} else {
		err = r.pool.QueryRow(ctx, `
			SELECT id FROM crm_leads
			WHERE account_id = $1 AND tracked_user_id = $2
			LIMIT 1
		`, accountID, trackedUserID).Scan(&leadID)
	}
	if errors.Is(err, pgx.ErrNoRows) || err != nil || leadID == 0 {
		return 0, false
	}

	nextStatus := domain.CRMRefundRefunded
	if strings.EqualFold(refundType, domain.RefundTypeChargeback) {
		nextStatus = domain.CRMRefundChargeback
	}
	if amount < 0 {
		amount = 0
	}

	tag, uerr := r.pool.Exec(ctx, `
		UPDATE crm_leads
		SET value = GREATEST(0, value - $3),
		    refund_status = CASE
		        WHEN refund_status = 'chargeback' OR $4 = 'chargeback' THEN 'chargeback'
		        ELSE 'refunded'
		    END,
		    refunded_amount = refunded_amount + $3,
		    updated_at = NOW()
		WHERE id = $1 AND account_id = $2
	`, leadID, accountID, amount, nextStatus)
	if uerr != nil || tag.RowsAffected() == 0 {
		return leadID, false
	}

	if strings.TrimSpace(noteText) != "" {
		_, _ = r.pool.Exec(ctx, `INSERT INTO crm_notes (crm_lead_id, content) VALUES ($1, $2)`, leadID, strings.TrimSpace(noteText))
	}
	return leadID, true
}
