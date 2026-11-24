package repo

import (
	"context"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
)

func (r *WhatsAppAIRepo) HasActiveChatAIQualification(ctx context.Context, chatID int64) bool {
	if chatID <= 0 {
		return false
	}
	var exists bool
	_ = r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM whatsapp_ai_events
			WHERE chat_id = $1 AND event_type = 'qualified' AND status = 'active'
		)
	`, chatID).Scan(&exists)
	return exists
}

func (r *WhatsAppAIRepo) HasRecentActiveChatAISale(ctx context.Context, chatID int64, amount float64, aroundTime time.Time) bool {
	if chatID <= 0 {
		return false
	}
	if aroundTime.IsZero() {
		aroundTime = time.Now()
	}
	rows, err := r.pool.Query(ctx, `
		SELECT amount, created_at
		FROM whatsapp_ai_events
		WHERE chat_id = $1
		  AND event_type = 'purchase'
		  AND status = 'active'
		  AND created_at >= $2 AND created_at <= $3
	`, chatID, aroundTime.Add(-48*time.Hour), aroundTime.Add(12*time.Hour))
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var prevAmt float64
		var prevAt time.Time
		if rows.Scan(&prevAmt, &prevAt) != nil {
			continue
		}
		if amount > 0 && prevAmt > 0 && cache.ValuesMatchCrossSource(prevAmt, amount) {
			return true
		}
		diff := prevAt.Sub(aroundTime)
		if diff < 0 {
			diff = -diff
		}
		if (amount <= 0 || prevAmt <= 0) && diff <= 4*time.Hour {
			return true
		}
	}
	return false
}
