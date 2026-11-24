package repo

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// consolidateGroupCRMLeads unifica eventuais cards duplicados no CRM pertencentes
// aos tracked_users ou grupos mesclados em ResolveAndLink, preservando o card de
// etapa mais avançada (ou venda ganha), migrando notas, tags, chats WhatsApp e eventos de IA.
func consolidateGroupCRMLeads(
	ctx context.Context,
	tx pgx.Tx,
	accountID, canonicalGroupID int64,
	matchedUserIDs, involvedGroupIDs []int64,
) error {
	if canonicalGroupID <= 0 || (len(matchedUserIDs) == 0 && len(involvedGroupIDs) == 0) {
		return nil
	}

	rows, err := tx.Query(ctx, `
		SELECT cl.id, COALESCE(cl.value, 0)
		FROM crm_leads cl
		LEFT JOIN crm_stages cs ON cs.id = cl.stage_id
		WHERE cl.account_id = $1
		  AND (
		    (cl.group_id IS NOT NULL AND cl.group_id = ANY($2::bigint[]))
		    OR cl.tracked_user_id = ANY($3::bigint[])
		  )
		ORDER BY
			(COALESCE(LOWER(cs.trigger_event), '') = 'purchase' OR COALESCE(LOWER(cs.trigger_event_ctwa), '') = 'purchase') DESC,
			COALESCE(cs.order_index, -1) DESC,
			COALESCE(cl.value, 0) DESC,
			cl.updated_at DESC,
			cl.id ASC
	`, accountID, involvedGroupIDs, matchedUserIDs)
	if err != nil {
		return err
	}

	type leadCandidate struct {
		id  int64
		val float64
	}
	var leads []leadCandidate
	var maxVal float64
	for rows.Next() {
		var lc leadCandidate
		if err := rows.Scan(&lc.id, &lc.val); err != nil {
			rows.Close()
			return err
		}
		if lc.val > maxVal {
			maxVal = lc.val
		}
		leads = append(leads, lc)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(leads) == 0 {
		return nil
	}

	winnerID := leads[0].id
	for _, loser := range leads[1:] {
		if _, err := tx.Exec(ctx, `UPDATE crm_notes SET crm_lead_id = $1 WHERE crm_lead_id = $2`, winnerID, loser.id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO crm_lead_tags (crm_lead_id, crm_tag_id)
			SELECT $1, crm_tag_id FROM crm_lead_tags WHERE crm_lead_id = $2
			ON CONFLICT DO NOTHING
		`, winnerID, loser.id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM crm_lead_tags WHERE crm_lead_id = $1`, loser.id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE whatsapp_chats SET crm_lead_id = $1 WHERE crm_lead_id = $2`, winnerID, loser.id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE whatsapp_ai_events SET crm_lead_id = $1 WHERE crm_lead_id = $2`, winnerID, loser.id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM crm_leads WHERE id = $1`, loser.id); err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE crm_leads
		SET group_id = $1,
		    value = GREATEST(COALESCE(value, 0), $2)
		WHERE id = $3
	`, canonicalGroupID, maxVal, winnerID)
	return err
}
