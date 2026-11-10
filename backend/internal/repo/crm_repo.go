package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

var ErrCannotDeleteLastStage = errors.New("cannot delete last stage")

type CRMRepo struct {
	pool *pgxpool.Pool
}

func NewCRMRepo(pool *pgxpool.Pool) *CRMRepo {
	return &CRMRepo{pool: pool}
}

// EnsureDefaultStages cria etapas iniciais se não existirem para a conta e repara leads órfãos legados.
func (r *CRMRepo) EnsureDefaultStages(ctx context.Context, accountID int64) error {
	var count int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM crm_stages WHERE account_id = $1", accountID).Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		stages := []domain.CRMStage{
			{Name: "Novo Lead", Color: "#3B82F6", OrderIndex: 0, TriggerEvent: ""},
			{Name: "Em Contato", Color: "#F59E0B", OrderIndex: 1, TriggerEvent: ""},
			{Name: "Venda Fechada", Color: "#10B981", OrderIndex: 2, TriggerEvent: "Purchase"},
		}
		for _, s := range stages {
			if _, err := r.pool.Exec(ctx, `
				INSERT INTO crm_stages (account_id, name, color, order_index, trigger_event)
				VALUES ($1, $2, $3, $4, $5)
			`, accountID, s.Name, s.Color, s.OrderIndex, s.TriggerEvent); err != nil {
				return err
			}
		}
	}

	_, _ = r.pool.Exec(ctx, `
		UPDATE crm_leads
		SET stage_id = (
			SELECT id FROM crm_stages
			WHERE account_id = $1
			ORDER BY order_index ASC, id ASC
			LIMIT 1
		)
		WHERE account_id = $1 AND stage_id IS NULL
	`, accountID)

	return nil
}

func (r *CRMRepo) GetStages(ctx context.Context, accountID int64) ([]domain.CRMStage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, account_id, name, color, order_index, trigger_event, google_ads_account_id, google_ads_action_id, created_at, updated_at
		FROM crm_stages
		WHERE account_id = $1
		ORDER BY order_index ASC, id ASC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []domain.CRMStage
	for rows.Next() {
		var s domain.CRMStage
		if err := rows.Scan(&s.ID, &s.AccountID, &s.Name, &s.Color, &s.OrderIndex, &s.TriggerEvent, &s.GoogleAdsAccountID, &s.GoogleAdsActionID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		stages = append(stages, s)
	}
	return stages, rows.Err()
}

func (r *CRMRepo) GetFirstStage(ctx context.Context, accountID int64) (*domain.CRMStage, error) {
	var s domain.CRMStage
	err := r.pool.QueryRow(ctx, `
		SELECT id, account_id, name, color, order_index, trigger_event, google_ads_account_id, google_ads_action_id, created_at, updated_at
		FROM crm_stages
		WHERE account_id = $1
		ORDER BY order_index ASC, id ASC LIMIT 1
	`, accountID).Scan(&s.ID, &s.AccountID, &s.Name, &s.Color, &s.OrderIndex, &s.TriggerEvent, &s.GoogleAdsAccountID, &s.GoogleAdsActionID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CRMRepo) GetStageByID(ctx context.Context, id, accountID int64) (*domain.CRMStage, error) {
	var s domain.CRMStage
	err := r.pool.QueryRow(ctx, `
		SELECT id, account_id, name, color, order_index, trigger_event, google_ads_account_id, google_ads_action_id, created_at, updated_at
		FROM crm_stages
		WHERE id = $1 AND account_id = $2
	`, id, accountID).Scan(&s.ID, &s.AccountID, &s.Name, &s.Color, &s.OrderIndex, &s.TriggerEvent, &s.GoogleAdsAccountID, &s.GoogleAdsActionID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CRMRepo) CreateStage(ctx context.Context, accountID int64, name, color, triggerEvent string, googleAdsAccountID *int64, googleAdsActionID string) error {
	var maxOrder int
	if err := r.pool.QueryRow(ctx, "SELECT COALESCE(MAX(order_index), -1) FROM crm_stages WHERE account_id = $1", accountID).Scan(&maxOrder); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO crm_stages (account_id, name, color, order_index, trigger_event, google_ads_account_id, google_ads_action_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, accountID, name, color, maxOrder+1, triggerEvent, googleAdsAccountID, googleAdsActionID)
	return err
}

func (r *CRMRepo) UpdateStage(ctx context.Context, id, accountID int64, name, color, triggerEvent string, googleAdsAccountID *int64, googleAdsActionID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE crm_stages
		SET name = $3, color = $4, trigger_event = $5, google_ads_account_id = $6, google_ads_action_id = $7, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
	`, id, accountID, name, color, triggerEvent, googleAdsAccountID, googleAdsActionID)
	return err
}

// DeleteStage transfere os leads da coluna para a primeira etapa remanescente antes de excluir.
func (r *CRMRepo) DeleteStage(ctx context.Context, id, accountID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var fallbackStageID int64
	err = tx.QueryRow(ctx, `
		SELECT id FROM crm_stages
		WHERE account_id = $1 AND id <> $2
		ORDER BY order_index ASC, id ASC
		LIMIT 1
	`, accountID, id).Scan(&fallbackStageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCannotDeleteLastStage
	}
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE crm_leads
		SET stage_id = $3, updated_at = NOW()
		WHERE account_id = $1 AND (stage_id = $2 OR stage_id IS NULL)
	`, accountID, id, fallbackStageID); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM crm_stages WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *CRMRepo) UpdateStageOrder(ctx context.Context, accountID int64, stageOrders map[int64]int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for id, order := range stageOrders {
		if _, err := tx.Exec(ctx, "UPDATE crm_stages SET order_index = $1 WHERE id = $2 AND account_id = $3", order, id, accountID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *CRMRepo) groupIDOf(ctx context.Context, trackedUserID int64) (*int64, error) {
	var groupID *int64
	err := r.pool.QueryRow(ctx, `SELECT group_id FROM tracked_users WHERE id = $1`, trackedUserID).Scan(&groupID)
	return groupID, err
}
