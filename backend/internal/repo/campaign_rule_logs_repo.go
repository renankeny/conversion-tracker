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
)

type CampaignRuleLogsRepo struct {
	db *pgxpool.Pool
}

func NewCampaignRuleLogsRepo(db *pgxpool.Pool) *CampaignRuleLogsRepo {
	return &CampaignRuleLogsRepo{db: db}
}

const selectCampaignRuleLogCols = `
	id, account_id, rule_id, rule_name, platform, level, entity_subtype,
	object_id, parent_id, object_name, pixel_name, account_ref_id,
	ad_account_id, login_customer_id, budget_id, currency,
	action_type, dry_run, status, previous_status, new_status,
	previous_budget, new_budget, metrics_snapshot, reason, error_message,
	undone_at, created_at
`

func scanCampaignRuleLog(row pgx.Row) (*domain.CampaignRuleLog, error) {
	var l domain.CampaignRuleLog
	var snapRaw []byte
	err := row.Scan(
		&l.ID, &l.AccountID, &l.RuleID, &l.RuleName, &l.Platform, &l.Level, &l.EntitySubtype,
		&l.ObjectID, &l.ParentID, &l.ObjectName, &l.PixelName, &l.AccountRefID,
		&l.AdAccountID, &l.LoginCustomerID, &l.BudgetID, &l.Currency,
		&l.ActionType, &l.DryRun, &l.Status, &l.PreviousStatus, &l.NewStatus,
		&l.PreviousBudget, &l.NewBudget, &snapRaw, &l.Reason, &l.ErrorMessage,
		&l.UndoneAt, &l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(snapRaw) > 0 {
		_ = json.Unmarshal(snapRaw, &l.MetricsSnapshot)
	}
	return &l, nil
}

func (r *CampaignRuleLogsRepo) InsertLog(ctx context.Context, log *domain.CampaignRuleLog) error {
	snapJSON, _ := json.Marshal(log.MetricsSnapshot)
	if strings.TrimSpace(log.Currency) == "" {
		log.Currency = "BRL"
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO campaign_rule_logs (
			account_id, rule_id, rule_name, platform, level, entity_subtype,
			object_id, parent_id, object_name, pixel_name, account_ref_id,
			ad_account_id, login_customer_id, budget_id, currency,
			action_type, dry_run, status, previous_status, new_status,
			previous_budget, new_budget, metrics_snapshot, reason, error_message
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,$11,$12,
			$13,$14,$15,$16,
			$17,$18,$19,$20,$21,
			$22,$23,$24,$25
		)
		RETURNING id, created_at
	`,
		log.AccountID, log.RuleID, log.RuleName, log.Platform, log.Level, log.EntitySubtype,
		log.ObjectID, log.ParentID, log.ObjectName, log.PixelName, log.AccountRefID,
		log.AdAccountID, log.LoginCustomerID, log.BudgetID, log.Currency,
		log.ActionType, log.DryRun, log.Status, log.PreviousStatus, log.NewStatus,
		log.PreviousBudget, log.NewBudget, snapJSON, log.Reason, log.ErrorMessage,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *CampaignRuleLogsRepo) ListLogs(ctx context.Context, accountID, ruleID int64, status string, limit int) ([]domain.CampaignRuleLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 60
	}
	where := []string{"account_id = $1"}
	args := []any{accountID}
	argN := 2

	if ruleID > 0 {
		where = append(where, fmt.Sprintf("rule_id = $%d", argN))
		args = append(args, ruleID)
		argN++
	}
	if s := strings.ToLower(strings.TrimSpace(status)); s != "" && s != "all" {
		where = append(where, fmt.Sprintf("status = $%d", argN))
		args = append(args, s)
		argN++
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT %s
		FROM campaign_rule_logs
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d
	`, selectCampaignRuleLogCols, strings.Join(where, " AND "), argN)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.CampaignRuleLog{}
	for rows.Next() {
		item, err := scanCampaignRuleLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *CampaignRuleLogsRepo) GetLogByID(ctx context.Context, accountID, logID int64) (*domain.CampaignRuleLog, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+selectCampaignRuleLogCols+`
		FROM campaign_rule_logs
		WHERE account_id = $1 AND id = $2
	`, accountID, logID)
	item, err := scanCampaignRuleLog(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *CampaignRuleLogsRepo) HasRecentExecution(ctx context.Context, ruleID int64, objectID string, since time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM campaign_rule_logs
			WHERE rule_id = $1
			  AND object_id = $2
			  AND status IN ('executed', 'simulated')
			  AND created_at >= $3
		)
	`, ruleID, objectID, since).Scan(&exists)
	return exists, err
}

// WasPausedByRuleToday verifica se um objeto foi pausado automaticamente por
// alguma regra de campanha hoje (usado para Reativação por Venda Atrasada).
func (r *CampaignRuleLogsRepo) WasPausedByRuleToday(ctx context.Context, accountID int64, platform, objectID string, startOfDaySP time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM campaign_rule_logs
			WHERE account_id = $1
			  AND platform = $2
			  AND object_id = $3
			  AND action_type = 'pause'
			  AND status = 'executed'
			  AND created_at >= $4
		)
	`, accountID, platform, objectID, startOfDaySP).Scan(&exists)
	return exists, err
}

func (r *CampaignRuleLogsRepo) MarkLogUndone(ctx context.Context, accountID, logID int64) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE campaign_rule_logs
		SET status = 'undone', undone_at = NOW()
		WHERE id = $1 AND account_id = $2 AND status = 'executed'
	`, logID, accountID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CampaignRuleLogsRepo) CountTodayExecutions(ctx context.Context, accountID int64, startOfDaySP time.Time) (executed, simulated int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'executed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'simulated' THEN 1 ELSE 0 END), 0)
		FROM campaign_rule_logs
		WHERE account_id = $1 AND created_at >= $2
	`, accountID, startOfDaySP).Scan(&executed, &simulated)
	return executed, simulated, err
}
