package repo

import (
	"context"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
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
			  AND channel <> 'internal'
			  AND status IN ('sent', 'queued')
		)
	`, accountID, pixelName, extIDs).Scan(&exists)
	return exists, err
}

// HasEventNameSent verifica se um evento específico (ex: QualifiedLead, Schedule, AddToCart)
// já foi enviado para esta identidade/grupo neste pixel, sem bloquear QualifiedLead por causa de LeadSubmitted.
func (r *WhatsAppAIRepo) HasEventNameSent(ctx context.Context, accountID, trackedUserID int64, pixelName, eventName string) (bool, error) {
	extIDs := r.groupExternalIDs(ctx, accountID, trackedUserID)
	evName := strings.TrimSpace(eventName)
	if len(extIDs) == 0 || pixelName == "" || evName == "" {
		return false, nil
	}
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sent_events
			WHERE account_id = $1
			  AND pixel_name = $2
			  AND external_id = ANY($3::text[])
			  AND LOWER(event_name) = LOWER($4)
			  AND channel <> 'internal'
			  AND status IN ('sent', 'queued')
		)
	`, accountID, pixelName, extIDs, evName).Scan(&exists)
	return exists, err
}

// HasMatchingPurchaseSent evita duplicar uma venda caso o gatilho do atendente,
// um webhook de gateway (Kiwify/Hotmart/Cakto), pixel web ou o Kanban já tenha disparado Purchase.
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
	winStart := aroundTime.Add(-72 * time.Hour)
	winEnd := aroundTime.Add(12 * time.Hour)

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
		// Entre duas vendas detectadas pela própria IA (ex: upsell/recompra em dias diferentes),
		// limitar deduplicação a 24h para não bloquear compras legítimas subsequentes.
		if strings.HasPrefix(strings.TrimSpace(evID), "ai_sale:") {
			diff := sentAt.Sub(aroundTime)
			if diff < 0 {
				diff = -diff
			}
			if diff > 24*time.Hour {
				continue
			}
		}
		if cache.MatchCrossSourcePurchase(evID, storedOrd, sentVal, sentAt, "", true, amount, aroundTime) {
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


