package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type CampaignRulesRepo struct {
	db *pgxpool.Pool
}

func NewCampaignRulesRepo(db *pgxpool.Pool) *CampaignRulesRepo {
	return &CampaignRulesRepo{db: db}
}

const selectCampaignRuleCols = `
	id, account_id, name, description, template_key, enabled, dry_run,
	platform, pixel_name, level, attribution_mode, rule_kind, period_preset,
	conditions, action_type, action_value, min_budget, max_budget,
	name_contains, name_excludes, target_status, frequency_minutes, min_hour_sp,
	schedule_time, schedule_days, cooldown_minutes,
	notify_push, notify_whatsapp, whatsapp_phone,
	last_run_at, last_triggered_at, last_error, trigger_count,
	created_at, updated_at
`

func scanCampaignRule(row pgx.Row) (*domain.CampaignRule, error) {
	var r domain.CampaignRule
	var condRaw, daysRaw []byte
	err := row.Scan(
		&r.ID, &r.AccountID, &r.Name, &r.Description, &r.TemplateKey, &r.Enabled, &r.DryRun,
		&r.Platform, &r.PixelName, &r.Level, &r.AttributionMode, &r.RuleKind, &r.PeriodPreset,
		&condRaw, &r.ActionType, &r.ActionValue, &r.MinBudget, &r.MaxBudget,
		&r.NameContains, &r.NameExcludes, &r.TargetStatus, &r.FrequencyMinutes, &r.MinHourSP,
		&r.ScheduleTime, &daysRaw, &r.CooldownMinutes,
		&r.NotifyPush, &r.NotifyWhatsApp, &r.WhatsAppPhone,
		&r.LastRunAt, &r.LastTriggeredAt, &r.LastError, &r.TriggerCount,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(condRaw) > 0 {
		_ = json.Unmarshal(condRaw, &r.Conditions)
	}
	if r.Conditions == nil {
		r.Conditions = []domain.CampaignRuleCondition{}
	}
	if len(daysRaw) > 0 {
		_ = json.Unmarshal(daysRaw, &r.ScheduleDays)
	}
	if r.ScheduleDays == nil {
		r.ScheduleDays = []int{}
	}
	return &r, nil
}

func (r *CampaignRulesRepo) ListByAccount(ctx context.Context, accountID int64) ([]domain.CampaignRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+selectCampaignRuleCols+`
		FROM campaign_rules
		WHERE account_id = $1
		ORDER BY enabled DESC, id DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.CampaignRule{}
	for rows.Next() {
		item, err := scanCampaignRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *CampaignRulesRepo) ListAllEnabled(ctx context.Context) ([]domain.CampaignRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+selectCampaignRuleCols+`
		FROM campaign_rules
		WHERE enabled = TRUE
		ORDER BY account_id ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.CampaignRule{}
	for rows.Next() {
		item, err := scanCampaignRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *CampaignRulesRepo) GetByID(ctx context.Context, accountID, id int64) (*domain.CampaignRule, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+selectCampaignRuleCols+`
		FROM campaign_rules
		WHERE account_id = $1 AND id = $2
	`, accountID, id)
	item, err := scanCampaignRule(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *CampaignRulesRepo) Create(ctx context.Context, rule *domain.CampaignRule) error {
	if err := rule.NormalizeAndValidate(); err != nil {
		return err
	}
	condJSON, _ := json.Marshal(rule.Conditions)
	daysJSON, _ := json.Marshal(rule.ScheduleDays)

	return r.db.QueryRow(ctx, `
		INSERT INTO campaign_rules (
			account_id, name, description, template_key, enabled, dry_run,
			platform, pixel_name, level, attribution_mode, rule_kind, period_preset,
			conditions, action_type, action_value, min_budget, max_budget,
			name_contains, name_excludes, target_status, frequency_minutes, min_hour_sp,
			schedule_time, schedule_days, cooldown_minutes,
			notify_push, notify_whatsapp, whatsapp_phone
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9,$10,$11,$12,
			$13,$14,$15,$16,$17,
			$18,$19,$20,$21,$22,
			$23,$24,$25,
			$26,$27,$28
		)
		RETURNING id, created_at, updated_at
	`,
		rule.AccountID, rule.Name, rule.Description, rule.TemplateKey, rule.Enabled, rule.DryRun,
		rule.Platform, rule.PixelName, rule.Level, rule.AttributionMode, rule.RuleKind, rule.PeriodPreset,
		condJSON, rule.ActionType, rule.ActionValue, rule.MinBudget, rule.MaxBudget,
		rule.NameContains, rule.NameExcludes, rule.TargetStatus, rule.FrequencyMinutes, rule.MinHourSP,
		rule.ScheduleTime, daysJSON, rule.CooldownMinutes,
		rule.NotifyPush, rule.NotifyWhatsApp, rule.WhatsAppPhone,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *CampaignRulesRepo) Update(ctx context.Context, rule *domain.CampaignRule) error {
	if err := rule.NormalizeAndValidate(); err != nil {
		return err
	}
	condJSON, _ := json.Marshal(rule.Conditions)
	daysJSON, _ := json.Marshal(rule.ScheduleDays)

	err := r.db.QueryRow(ctx, `
		UPDATE campaign_rules SET
			name = $1, description = $2, template_key = $3, enabled = $4, dry_run = $5,
			platform = $6, pixel_name = $7, level = $8, attribution_mode = $9,
			rule_kind = $10, period_preset = $11, conditions = $12,
			action_type = $13, action_value = $14, min_budget = $15, max_budget = $16,
			name_contains = $17, name_excludes = $18, target_status = $19,
			frequency_minutes = $20, min_hour_sp = $21, schedule_time = $22,
			schedule_days = $23, cooldown_minutes = $24,
			notify_push = $25, notify_whatsapp = $26, whatsapp_phone = $27,
			updated_at = NOW()
		WHERE id = $28 AND account_id = $29
		RETURNING updated_at
	`,
		rule.Name, rule.Description, rule.TemplateKey, rule.Enabled, rule.DryRun,
		rule.Platform, rule.PixelName, rule.Level, rule.AttributionMode,
		rule.RuleKind, rule.PeriodPreset, condJSON,
		rule.ActionType, rule.ActionValue, rule.MinBudget, rule.MaxBudget,
		rule.NameContains, rule.NameExcludes, rule.TargetStatus,
		rule.FrequencyMinutes, rule.MinHourSP, rule.ScheduleTime,
		daysJSON, rule.CooldownMinutes,
		rule.NotifyPush, rule.NotifyWhatsApp, rule.WhatsAppPhone,
		rule.ID, rule.AccountID,
	).Scan(&rule.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *CampaignRulesRepo) ToggleEnabled(ctx context.Context, accountID, id int64, enabled bool) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE campaign_rules SET enabled = $1, updated_at = NOW()
		WHERE id = $2 AND account_id = $3
	`, enabled, id, accountID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CampaignRulesRepo) Delete(ctx context.Context, accountID, id int64) error {
	ct, err := r.db.Exec(ctx, `DELETE FROM campaign_rules WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CampaignRulesRepo) UpdateRunTelemetry(ctx context.Context, ruleID int64, triggeredCount int, lastErr string) error {
	if triggeredCount > 0 {
		_, err := r.db.Exec(ctx, `
			UPDATE campaign_rules
			SET last_run_at = NOW(),
			    last_triggered_at = NOW(),
			    last_error = $1,
			    trigger_count = trigger_count + $2,
			    updated_at = NOW()
			WHERE id = $3
		`, lastErr, triggeredCount, ruleID)
		return err
	}
	_, err := r.db.Exec(ctx, `
		UPDATE campaign_rules
		SET last_run_at = NOW(), last_error = $1, updated_at = NOW()
		WHERE id = $2
	`, lastErr, ruleID)
	return err
}
