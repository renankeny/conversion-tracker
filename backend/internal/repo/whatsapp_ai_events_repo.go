package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type AIChatCandidate struct {
	ChatID                int64
	AccountID             int64
	InstanceID            int64
	PixelID               int64
	Currency              string
	TrackedUserID         int64
	CRMLeadID             *int64
	Phone                 string
	CtwaClid              string
	PageID                string
	AIQualified           bool
	AILastAnalyzedMsgID   int64
	NewestMsgID           int64
	HasRecentPendingMedia bool
}

func (r *WhatsAppAIRepo) ListCandidateChats(ctx context.Context, accountID int64, limit int) ([]AIChatCandidate, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT
			c.id, i.account_id, c.instance_id, i.pixel_id, COALESCE(i.currency, 'BRL'),
			c.tracked_user_id, c.crm_lead_id, c.phone, c.ctwa_clid, c.page_id,
			c.ai_qualified, c.ai_last_analyzed_msg_id,
			(SELECT COALESCE(MAX(m.id), 0) FROM whatsapp_messages m WHERE m.chat_id = c.id) AS newest_msg_id,
			EXISTS (
				SELECT 1 FROM whatsapp_messages pm
				WHERE pm.chat_id = c.id
				  AND pm.id > c.ai_last_analyzed_msg_id
				  AND pm.message_type IN ('audio', 'image', 'document')
				  AND pm.media_status = 'pending'
				  AND pm.created_at > NOW() - INTERVAL '5 minutes'
			) AS has_pending_media
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON c.instance_id = i.id
		WHERE i.account_id = $1
		  AND i.enabled = TRUE
		  AND c.tracked_user_id IS NOT NULL
		  AND EXISTS (
		    SELECT 1 FROM whatsapp_messages m2
		    WHERE m2.chat_id = c.id AND m2.id > c.ai_last_analyzed_msg_id
		  )
		ORDER BY c.last_message_time ASC
		LIMIT $2
	`, accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("list ai candidate chats: %w", err)
	}
	defer rows.Close()

	var out []AIChatCandidate
	for rows.Next() {
		var item AIChatCandidate
		if err := rows.Scan(
			&item.ChatID, &item.AccountID, &item.InstanceID, &item.PixelID, &item.Currency,
			&item.TrackedUserID, &item.CRMLeadID, &item.Phone, &item.CtwaClid, &item.PageID,
			&item.AIQualified, &item.AILastAnalyzedMsgID, &item.NewestMsgID, &item.HasRecentPendingMedia,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *WhatsAppAIRepo) ListChatMessagesForAI(ctx context.Context, chatID int64, limit int) ([]*domain.WhatsAppMessage, error) {
	if limit <= 0 || limit > 80 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT * FROM (
			SELECT `+selectMessageColumns+`
			FROM whatsapp_messages m
			WHERE m.chat_id = $1
			ORDER BY m.timestamp DESC, m.id DESC
			LIMIT $2
		) sub
		ORDER BY sub.timestamp ASC, sub.id ASC
	`, chatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.WhatsAppMessage
	for rows.Next() {
		var m domain.WhatsAppMessage
		if err := scanWhatsAppMessage(rows, &m); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (r *WhatsAppAIRepo) UpdateChatAICursor(ctx context.Context, chatID, lastMsgID int64, markQualified bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE whatsapp_chats
		SET ai_last_analyzed_msg_id = GREATEST(ai_last_analyzed_msg_id, $2),
		    ai_qualified = ai_qualified OR $3,
		    ai_last_analyzed_at = NOW()
		WHERE id = $1
	`, chatID, lastMsgID, markQualified)
	return err
}

func (r *WhatsAppAIRepo) SetChatAIQualified(ctx context.Context, chatID int64, qualified bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE whatsapp_chats SET ai_qualified = $2 WHERE id = $1`, chatID, qualified)
	return err
}

const selectAIEventCols = `
	id, account_id, chat_id, crm_lead_id, tracked_user_id, event_type,
	evidence_message_id, amount, currency, ai_reason,
	crm_prev_stage_id, crm_new_stage_id, crm_prev_value,
	dispatched_event_id, deduplicated, status,
	contest_reason, contested_by_user_id, contested_at, created_at
`

func scanAIEvent(row pgx.Row, e *domain.WhatsAppAIEvent) error {
	return row.Scan(
		&e.ID, &e.AccountID, &e.ChatID, &e.CRMLeadID, &e.TrackedUserID, &e.EventType,
		&e.EvidenceMessageID, &e.Amount, &e.Currency, &e.AIReason,
		&e.CRMPrevStageID, &e.CRMNewStageID, &e.CRMPrevValue,
		&e.DispatchedEventID, &e.Deduplicated, &e.Status,
		&e.ContestReason, &e.ContestedByUserID, &e.ContestedAt, &e.CreatedAt,
	)
}

func (r *WhatsAppAIRepo) CreateAIEvent(ctx context.Context, ev *domain.WhatsAppAIEvent) (bool, error) {
	if ev.Status == "" {
		ev.Status = domain.AIEventStatusActive
	}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO whatsapp_ai_events (
			account_id, chat_id, crm_lead_id, tracked_user_id, event_type,
			evidence_message_id, amount, currency, ai_reason,
			crm_prev_stage_id, crm_new_stage_id, crm_prev_value,
			dispatched_event_id, deduplicated, status
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (chat_id, event_type, evidence_message_id)
		WHERE status = 'active' AND evidence_message_id IS NOT NULL
		DO NOTHING
		RETURNING id, created_at
	`,
		ev.AccountID, ev.ChatID, ev.CRMLeadID, ev.TrackedUserID, ev.EventType,
		ev.EvidenceMessageID, ev.Amount, domain.NormalizeWhatsAppCurrency(ev.Currency), ev.AIReason,
		ev.CRMPrevStageID, ev.CRMNewStageID, ev.CRMPrevValue,
		ev.DispatchedEventID, ev.Deduplicated, ev.Status,
	).Scan(&ev.ID, &ev.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *WhatsAppAIRepo) ListAIEventsByChat(ctx context.Context, chatID int64) ([]*domain.WhatsAppAIEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+selectAIEventCols+`
		FROM whatsapp_ai_events
		WHERE chat_id = $1
		ORDER BY created_at ASC, id ASC
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.WhatsAppAIEvent, 0)
	for rows.Next() {
		var e domain.WhatsAppAIEvent
		if err := scanAIEvent(rows, &e); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *WhatsAppAIRepo) ContestAIEvent(ctx context.Context, accountID, id, userID int64, reason string, addRule bool, authorEmail string) (*domain.WhatsAppAIEvent, error) {
	var ev domain.WhatsAppAIEvent
	row := r.pool.QueryRow(ctx, `
		UPDATE whatsapp_ai_events
		SET status = 'contested', contest_reason = $3, contested_by_user_id = NULLIF($4::bigint, 0), contested_at = NOW()
		WHERE id = $1 AND account_id = $2 AND status = 'active'
		RETURNING `+selectAIEventCols,
		id, accountID, strings.TrimSpace(reason), userID,
	)
	if err := scanAIEvent(row, &ev); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if addRule && strings.TrimSpace(reason) != "" {
		s, _ := r.GetSettings(ctx, accountID)
		if s != nil {
			rule := domain.AILearnedRule{
				ID: fmt.Sprintf("r_%d", ev.ID), EventType: ev.EventType,
				Rule: strings.TrimSpace(reason), AuthorEmail: authorEmail, CreatedAt: ev.CreatedAt,
			}
			s.LearnedRules = append(s.LearnedRules, rule)
			if raw, merr := json.Marshal(s.LearnedRules); merr == nil {
				_, _ = r.pool.Exec(ctx, `
					INSERT INTO account_ai_settings (account_id, learned_rules, updated_at)
					VALUES ($1, $2, NOW())
					ON CONFLICT (account_id) DO UPDATE SET learned_rules = $2, updated_at = NOW()
				`, accountID, raw)
			}
		}
	}
	return &ev, nil
}

