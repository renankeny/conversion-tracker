package repo

import (
	"context"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
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

// HasLeadConversionSent verifica se Lead, LeadSubmitted ou QualifiedLead já foi enviado ou enfileirado
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
			  AND status IN ('sent', 'queued')
		)
	`, accountID, pixelName, extIDs).Scan(&exists)
	return exists, err
}

// IsNonOrderPurchaseEventID identifica vendas sem order_id real de gateway.
func IsNonOrderPurchaseEventID(eventID string) bool {
	return cache.IsNonOrderPurchaseEventID(eventID)
}

// HasMatchingPurchaseSent evita duplicar uma venda disparada por fontes sem order_id (CRM, IA, Gatilho, Manual).
func (r *SentEventRepo) HasMatchingPurchaseSent(
	ctx context.Context,
	accountID, trackedUserID int64,
	externalID, pixelName string,
	amount float64,
	aroundTime time.Time,
) (bool, error) {
	return r.HasDuplicateExternalPurchase(ctx, accountID, trackedUserID, externalID, pixelName, "", amount, aroundTime)
}

// HasDuplicateExternalPurchase verifica se uma venda recebida via Webhook, API, Tracker Web, CRM ou IA
// já foi enviada/enfileirada anteriormente para o mesmo comprador/grupo, preservando compras distintas
// quando ambos os eventos possuem order_ids reais de checkout diferentes.
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
	winStart := aroundTime.Add(-72 * time.Hour)
	winEnd := aroundTime.Add(12 * time.Hour)
	orderNorm := strings.TrimSpace(incomingOrderID)
	incomingIsNonOrder := orderNorm == "" || IsNonOrderPurchaseEventID(orderNorm)

	rows, err := r.pool.Query(ctx, `
		SELECT event_id, COALESCE(input_json->>'order_id', ''), purchase_value, created_at
		FROM sent_events
		WHERE account_id = $1
		  AND pixel_name = $2
		  AND external_id = ANY($3::text[])
		  AND event_name = 'Purchase'
		  AND channel <> 'internal'
		  AND status IN ('sent', 'queued')
		  AND created_at >= $4 AND created_at <= $5
	`, accountID, pixelName, extIDs, winStart, winEnd)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var evID, storedOrd string
		var sentVal float64
		var sentAt time.Time
		if err := rows.Scan(&evID, &storedOrd, &sentVal, &sentAt); err != nil {
			continue
		}
		if cache.MatchCrossSourcePurchase(evID, storedOrd, sentVal, sentAt, orderNorm, incomingIsNonOrder, amount, aroundTime) {
			return true, nil
		}
	}
	return false, rows.Err()
}
