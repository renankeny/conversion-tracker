package repo

import (
	"context"
	"math"
	"strings"
	"time"
)

func (r *SentEventRepo) groupExternalIDsForUser(ctx context.Context, accountID, trackedUserID int64, fallbackExtID string) []string {
	var ids []string
	seen := map[string]bool{}
	add := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			ids = append(ids, s)
		}
	}
	add(fallbackExtID)
	if trackedUserID <= 0 {
		return ids
	}
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
		return ids
	}
	defer rows.Close()
	for rows.Next() {
		var ext string
		if rows.Scan(&ext) == nil {
			add(ext)
		}
	}
	return ids
}

// HasLeadConversionSent verifica se Lead, LeadSubmitted ou QualifiedLead já foi enviado
// por qualquer fonte (CTWA 1ª/3ª msg, frase-gatilho, IA, pixel web ou CRM) para a pessoa.
func (r *SentEventRepo) HasLeadConversionSent(ctx context.Context, accountID, trackedUserID int64, externalID, pixelName string) (bool, error) {
	extIDs := r.groupExternalIDsForUser(ctx, accountID, trackedUserID, externalID)
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
			  AND channel <> 'internal'
			  AND status = 'sent'
		)
	`, accountID, pixelName, extIDs).Scan(&exists)
	return exists, err
}

// HasMatchingPurchaseSent evita duplicar uma venda caso webhook, gatilho, IA ou Kanban já tenha enviado Purchase.
func (r *SentEventRepo) HasMatchingPurchaseSent(
	ctx context.Context,
	accountID, trackedUserID int64,
	externalID, pixelName string,
	amount float64,
	aroundTime time.Time,
) (bool, error) {
	extIDs := r.groupExternalIDsForUser(ctx, accountID, trackedUserID, externalID)
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
		  AND channel <> 'internal'
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

// IsNonOrderPurchaseEventID identifica vendas disparadas pelo CRM, IA, gatilho de frase ou confirmação manual,
// permitindo impedir reenvio duplicado caso o Webhook/API chegue logo depois para a mesma venda.
func IsNonOrderPurchaseEventID(eventID string) bool {
	id := strings.TrimSpace(eventID)
	return strings.HasPrefix(id, "crm_pur:") ||
		strings.HasPrefix(id, "crm:") ||
		strings.HasPrefix(id, "ai_sale:") ||
		strings.HasPrefix(id, "wpp_manual_") ||
		strings.HasPrefix(id, "pur_") ||
		strings.HasSuffix(id, ":Purchase")
}

// HasDuplicateExternalPurchase verifica se uma venda recebida via Webhook, API ou Tracker Web já foi
// enviada anteriormente (pelo mesmo order_id, sem order_id após outra venda igual, ou após baixa manual no CRM/IA/WhatsApp),
// preservando compras reais distintas quando houver dois order_ids de checkout diferentes.
func (r *SentEventRepo) HasDuplicateExternalPurchase(
	ctx context.Context,
	accountID, trackedUserID int64,
	externalID, pixelName, incomingOrderID string,
	amount float64,
	aroundTime time.Time,
) (bool, error) {
	extIDs := r.groupExternalIDsForUser(ctx, accountID, trackedUserID, externalID)
	if len(extIDs) == 0 || pixelName == "" {
		return false, nil
	}
	if aroundTime.IsZero() {
		aroundTime = time.Now()
	}
	winStart := aroundTime.Add(-24 * time.Hour)
	winEnd := aroundTime.Add(4 * time.Hour)
	orderNorm := strings.TrimSpace(incomingOrderID)

	rows, err := r.pool.Query(ctx, `
		SELECT event_id, purchase_value
		FROM sent_events
		WHERE account_id = $1
		  AND pixel_name = $2
		  AND external_id = ANY($3::text[])
		  AND event_name = 'Purchase'
		  AND channel <> 'internal'
		  AND status = 'sent'
		  AND created_at >= $4 AND created_at <= $5
	`, accountID, pixelName, extIDs, winStart, winEnd)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var evID string
		var sentVal float64
		if err := rows.Scan(&evID, &sentVal); err != nil {
			continue
		}
		if orderNorm != "" && strings.EqualFold(strings.TrimSpace(evID), orderNorm) {
			return true, nil
		}
		sameVal := amount > 0 && sentVal > 0 && math.Abs(sentVal-amount) <= 1.00
		if !sameVal {
			continue
		}
		if orderNorm == "" || IsNonOrderPurchaseEventID(evID) {
			return true, nil
		}
	}
	return false, rows.Err()
}
