package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type PlatformTokenRepo struct {
	db *pgxpool.Pool
}

func NewPlatformTokenRepo(db *pgxpool.Pool) *PlatformTokenRepo {
	return &PlatformTokenRepo{db: db}
}

// FindByPlatform retorna o token descriptografado de uma plataforma para uma conta.
func (r *PlatformTokenRepo) FindByPlatform(ctx context.Context, accountID int64, platformName string) (*domain.PlatformToken, error) {
	q := `SELECT id, account_id, platform_name, token_secret, created_at, updated_at
	      FROM platform_tokens
	      WHERE account_id = $1 AND platform_name = $2`
	var t domain.PlatformToken
	err := r.db.QueryRow(ctx, q, accountID, strings.ToLower(strings.TrimSpace(platformName))).Scan(
		&t.ID, &t.AccountID, &t.PlatformName, &t.TokenSecret, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find platform token: %w", err)
	}
	t.TokenSecret = strings.TrimSpace(util.Decrypt(t.TokenSecret))
	return &t, nil
}

// ListByAccount retorna todos os tokens ativos de uma empresa indexados por platform_name.
func (r *PlatformTokenRepo) ListByAccount(ctx context.Context, accountID int64) (map[string]domain.PlatformToken, error) {
	out := make(map[string]domain.PlatformToken)
	if accountID <= 0 {
		return out, nil
	}
	q := `SELECT id, account_id, platform_name, token_secret, created_at, updated_at
	      FROM platform_tokens
	      WHERE account_id = $1 AND token_secret <> ''`
	rows, err := r.db.Query(ctx, q, accountID)
	if err != nil {
		return nil, fmt.Errorf("list platform tokens: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t domain.PlatformToken
		if err := rows.Scan(&t.ID, &t.AccountID, &t.PlatformName, &t.TokenSecret, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan platform token: %w", err)
		}
		t.TokenSecret = strings.TrimSpace(util.Decrypt(t.TokenSecret))
		if t.TokenSecret != "" {
			out[strings.ToLower(t.PlatformName)] = t
		}
	}
	return out, rows.Err()
}

// UpsertToken insere ou atualiza o token criptografado de uma plataforma (ou remove quando vazio).
func (r *PlatformTokenRepo) UpsertToken(ctx context.Context, t *domain.PlatformToken) error {
	platform := strings.ToLower(strings.TrimSpace(t.PlatformName))
	secret := strings.TrimSpace(t.TokenSecret)
	if secret == "" {
		_, err := r.db.Exec(ctx, `DELETE FROM platform_tokens WHERE account_id = $1 AND platform_name = $2`, t.AccountID, platform)
		if err != nil {
			return fmt.Errorf("delete platform token: %w", err)
		}
		return nil
	}

	encSecret, err := util.Encrypt(secret)
	if err != nil {
		return fmt.Errorf("encrypt platform token: %w", err)
	}

	q := `INSERT INTO platform_tokens (account_id, platform_name, token_secret, created_at, updated_at)
	      VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	      ON CONFLICT (account_id, platform_name)
	      DO UPDATE SET token_secret = EXCLUDED.token_secret, updated_at = CURRENT_TIMESTAMP
	      RETURNING id, created_at, updated_at`
	if err := r.db.QueryRow(ctx, q, t.AccountID, platform, encSecret).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return fmt.Errorf("upsert platform token: %w", err)
	}
	return nil
}
