package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomWebhookTemplate struct {
	ID            string          `json:"id"`
	AccountID     *int64          `json:"account_id"`
	Name          string          `json:"name"`
	MappingConfig json.RawMessage `json:"mapping_config"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type CustomWebhookRepo struct {
	pool *pgxpool.Pool
}

func NewCustomWebhookRepo(pool *pgxpool.Pool) *CustomWebhookRepo {
	return &CustomWebhookRepo{pool: pool}
}

// ListByAccount lista templates visíveis para a conta (globais + próprios)
func (r *CustomWebhookRepo) ListByAccount(ctx context.Context, accountID int64) ([]CustomWebhookTemplate, error) {
	query := `
		SELECT id, account_id, name, mapping_config, created_at, updated_at
		FROM custom_webhook_templates
		WHERE account_id IS NULL OR account_id = $1
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []CustomWebhookTemplate
	for rows.Next() {
		var t CustomWebhookTemplate
		err := rows.Scan(&t.ID, &t.AccountID, &t.Name, &t.MappingConfig, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}

// GetByAccount busca um template garantindo acesso (global ou próprio)
func (r *CustomWebhookRepo) GetByAccount(ctx context.Context, id string, accountID int64) (*CustomWebhookTemplate, error) {
	query := `
		SELECT id, account_id, name, mapping_config, created_at, updated_at
		FROM custom_webhook_templates
		WHERE id = $1 AND (account_id IS NULL OR account_id = $2)
	`
	var t CustomWebhookTemplate
	err := r.pool.QueryRow(ctx, query, id, accountID).Scan(&t.ID, &t.AccountID, &t.Name, &t.MappingConfig, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Get busca um template pelo ID diretamente (usado pelo handler público do webhook)
func (r *CustomWebhookRepo) Get(ctx context.Context, id string) (*CustomWebhookTemplate, error) {
	query := `
		SELECT id, account_id, name, mapping_config, created_at, updated_at
		FROM custom_webhook_templates
		WHERE id = $1
	`
	var t CustomWebhookTemplate
	err := r.pool.QueryRow(ctx, query, id).Scan(&t.ID, &t.AccountID, &t.Name, &t.MappingConfig, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *CustomWebhookRepo) Create(ctx context.Context, t *CustomWebhookTemplate) error {
	query := `
		INSERT INTO custom_webhook_templates (account_id, name, mapping_config)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, t.AccountID, t.Name, t.MappingConfig).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *CustomWebhookRepo) Update(ctx context.Context, t *CustomWebhookTemplate) error {
	query := `
		UPDATE custom_webhook_templates
		SET name = $1, mapping_config = $2, updated_at = NOW()
		WHERE id = $3 AND (account_id IS NULL OR account_id = $4)
		RETURNING updated_at
	`
	// Fallback for global templates where account_id doesn't match the caller?
	// Wait: If a superadmin updates a global template (account_id = NULL), 
	// the WHERE clause (account_id IS NULL OR account_id = $4) will match!
	// So passing accountID=0 or whatever will still match `account_id IS NULL`.
	var dummyAcc int64
	if t.AccountID != nil {
		dummyAcc = *t.AccountID
	}
	return r.pool.QueryRow(ctx, query, t.Name, t.MappingConfig, t.ID, dummyAcc).Scan(&t.UpdatedAt)
}

func (r *CustomWebhookRepo) Delete(ctx context.Context, id string, accountID int64) error {
	query := `
		DELETE FROM custom_webhook_templates
		WHERE id = $1 AND (account_id IS NULL OR account_id = $2)
	`
	_, err := r.pool.Exec(ctx, query, id, accountID)
	return err
}
