package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type WhatsAppAIRepo struct {
	pool *pgxpool.Pool
}

func NewWhatsAppAIRepo(pool *pgxpool.Pool) *WhatsAppAIRepo {
	return &WhatsAppAIRepo{pool: pool}
}

func (r *WhatsAppAIRepo) GetSettings(ctx context.Context, accountID int64) (*domain.AccountAISettings, error) {
	s := &domain.AccountAISettings{
		AccountID:    accountID,
		Model:        "gpt-4o-mini",
		LearnedRules: []domain.AILearnedRule{},
	}
	var rulesRaw []byte
	err := r.pool.QueryRow(ctx, `
		SELECT enabled, COALESCE(NULLIF(model, ''), 'gpt-4o-mini'),
		       qualified_stage_id, sale_stage_id, base_prompt, learned_rules, last_run_at, updated_at
		FROM account_ai_settings
		WHERE account_id = $1
	`, accountID).Scan(
		&s.Enabled, &s.Model,
		&s.QualifiedStageID, &s.SaleStageID, &s.BasePrompt, &rulesRaw, &s.LastRunAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get account_ai_settings: %w", err)
	}
	if len(rulesRaw) > 0 {
		_ = json.Unmarshal(rulesRaw, &s.LearnedRules)
	}
	if s.LearnedRules == nil {
		s.LearnedRules = []domain.AILearnedRule{}
	}
	return s, nil
}

func (r *WhatsAppAIRepo) UpsertSettings(ctx context.Context, s *domain.AccountAISettings) error {
	model := strings.TrimSpace(s.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	rules := s.LearnedRules
	if rules == nil {
		rules = []domain.AILearnedRule{}
	}
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("marshal learned_rules: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO account_ai_settings (
			account_id, enabled, model,
			qualified_stage_id, sale_stage_id, base_prompt, learned_rules, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (account_id) DO UPDATE SET
			enabled            = EXCLUDED.enabled,
			model              = EXCLUDED.model,
			qualified_stage_id = EXCLUDED.qualified_stage_id,
			sale_stage_id      = EXCLUDED.sale_stage_id,
			base_prompt        = EXCLUDED.base_prompt,
			learned_rules      = EXCLUDED.learned_rules,
			updated_at         = NOW()
	`,
		s.AccountID, s.Enabled, model,
		s.QualifiedStageID, s.SaleStageID, s.BasePrompt, rulesJSON,
	)
	return err
}

// ResolveOpenAIKey busca a chave da OpenAI configurada em "Meu Perfil"
// (priorizando o usuário atual, depois o dono da conta, admins da conta e superadmins).
func (r *WhatsAppAIRepo) ResolveOpenAIKey(ctx context.Context, accountID, fallbackUserID int64) string {
	rows, err := r.pool.Query(ctx, `
		SELECT u.preferences
		FROM admin_users u
		LEFT JOIN accounts a ON a.id = $1
		WHERE (
			($2::bigint > 0 AND u.id = $2)
			OR (a.owner_id IS NOT NULL AND u.id = a.owner_id)
			OR (u.account_id = $1 AND COALESCE(u.role, 'admin') IN ('owner', 'superadmin', 'admin'))
			OR COALESCE(u.role, '') IN ('owner', 'superadmin')
		)
		ORDER BY
			CASE
				WHEN $2::bigint > 0 AND u.id = $2 THEN 0
				WHEN a.owner_id IS NOT NULL AND u.id = a.owner_id THEN 1
				WHEN u.account_id = $1 THEN 2
				ELSE 3
			END,
			u.id ASC
		LIMIT 10
	`, accountID, fallbackUserID)
	if err != nil {
		return ""
	}
	defer rows.Close()

	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil || len(raw) == 0 {
			continue
		}
		var prefs map[string]any
		if json.Unmarshal(raw, &prefs) != nil {
			continue
		}
		enc, _ := prefs["openai_api_key"].(string)
		if strings.TrimSpace(enc) == "" {
			continue
		}
		if dec := strings.TrimSpace(util.Decrypt(strings.TrimSpace(enc))); dec != "" {
			return dec
		}
	}
	return ""
}

func (r *WhatsAppAIRepo) ListEnabledAccounts(ctx context.Context) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT account_id FROM account_ai_settings WHERE enabled = TRUE ORDER BY account_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func (r *WhatsAppAIRepo) TouchLastRun(ctx context.Context, accountID int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE account_ai_settings SET last_run_at = NOW() WHERE account_id = $1`, accountID)
	return err
}

// RevokeDispatchedAIEvent estorna o evento da IA em sent_events (status = 'contested')
// e decrementa o bucket correspondente em event_counters para não inflar métricas da Dashboard.
func (r *WhatsAppAIRepo) RevokeDispatchedAIEvent(ctx context.Context, accountID int64, dispatchedEventID string) {
	evID := strings.TrimSpace(dispatchedEventID)
	if evID == "" {
		return
	}
	var pixelName, eventName string
	var createdAt time.Time
	var hadSent, hadFailed bool
	err := r.pool.QueryRow(ctx, `
		SELECT MAX(pixel_name), MAX(event_name), MIN(created_at),
		       BOOL_OR(status = 'sent'), BOOL_OR(status = 'failed')
		FROM sent_events
		WHERE account_id = $1 AND event_id = $2 AND status <> 'contested'
		HAVING COUNT(*) > 0
	`, accountID, evID).Scan(&pixelName, &eventName, &createdAt, &hadSent, &hadFailed)
	if err != nil {
		return
	}
	_, _ = r.pool.Exec(ctx, `
		UPDATE sent_events
		SET status = 'contested', updated_at = NOW()
		WHERE account_id = $1 AND event_id = $2 AND status <> 'contested'
	`, accountID, evID)

	var sentDec, failedDec int64
	if hadSent {
		sentDec = 1
	} else if hadFailed {
		failedDec = 1
	}
	bucket := createdAt.UTC().Truncate(time.Hour)
	_, _ = r.pool.Exec(ctx, `
		UPDATE event_counters
		SET count_total  = GREATEST(0, count_total - 1),
		    count_sent   = GREATEST(0, count_sent - $5),
		    count_failed = GREATEST(0, count_failed - $6),
		    updated_at   = NOW()
		WHERE account_id = $1 AND pixel_name = $2 AND event_name = $3 AND bucket_at = $4
	`, accountID, pixelName, eventName, bucket, sentDec, failedDec)
}
