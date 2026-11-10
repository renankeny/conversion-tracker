package repo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type PlatformTokenRepo struct {
	db *pgxpool.Pool
}

func NewPlatformTokenRepo(db *pgxpool.Pool) *PlatformTokenRepo {
	return &PlatformTokenRepo{db: db}
}

// FindByPlatform retorna o token de uma determinada plataforma para uma conta.
func (r *PlatformTokenRepo) FindByPlatform(ctx context.Context, accountID int64, platformName string) (*domain.PlatformToken, error) {
	q := `SELECT id, account_id, platform_name, token_secret, created_at, updated_at 
	      FROM platform_tokens 
	      WHERE account_id = $1 AND platform_name = $2`
	var t domain.PlatformToken
	err := r.db.QueryRow(ctx, q, accountID, platformName).Scan(
		&t.ID, &t.AccountID, &t.PlatformName, &t.TokenSecret, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// UpsertToken insere ou atualiza o token de uma plataforma.
func (r *PlatformTokenRepo) UpsertToken(ctx context.Context, t *domain.PlatformToken) error {
	q := `INSERT INTO platform_tokens (account_id, platform_name, token_secret, created_at, updated_at)
	      VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	      ON CONFLICT (account_id, platform_name) 
	      DO UPDATE SET token_secret = EXCLUDED.token_secret, updated_at = CURRENT_TIMESTAMP
	      RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, q, t.AccountID, t.PlatformName, t.TokenSecret).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}
