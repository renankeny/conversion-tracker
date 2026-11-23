package repo

import "context"

// LeadBelongsToPixel valida estritamente se um lead do CRM pertence ao Pixel
// compartilhado no Portal Público do Cliente (/cliente/{token}), seja por
// tracked_users.pixel_name ou por vínculo com instância WhatsApp desse pixel.
func (r *CRMRepo) LeadBelongsToPixel(ctx context.Context, leadID, accountID, pixelID int64, pixelName string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM crm_leads c
			JOIN tracked_users t ON t.id = c.tracked_user_id
			WHERE c.id = $1
			  AND c.account_id = $2
			  AND (
			    t.pixel_name = $4
			    OR EXISTS (
			      SELECT 1
			      FROM whatsapp_chats wc
			      JOIN whatsapp_instances wi ON wi.id = wc.instance_id
			      WHERE (wc.crm_lead_id = c.id OR wc.tracked_user_id = c.tracked_user_id)
			        AND wi.account_id = c.account_id
			        AND wi.pixel_id = $3
			    )
			  )
		)
	`, leadID, accountID, pixelID, pixelName).Scan(&ok)
	return ok, err
}
