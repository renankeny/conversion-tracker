package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type AccountRepo struct {
	pool *pgxpool.Pool
}

func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool}
}

const accountColumns = `id, name, slug, enabled, created_at, updated_at`

func (r *AccountRepo) FindByID(ctx context.Context, id int64) (*domain.Account, error) {
	q := `SELECT ` + accountColumns + ` FROM accounts WHERE id = $1`
	return scanAccount(r.pool.QueryRow(ctx, q, id))
}

func (r *AccountRepo) FindBySlug(ctx context.Context, slug string) (*domain.Account, error) {
	q := `SELECT ` + accountColumns + ` FROM accounts WHERE slug = $1`
	return scanAccount(r.pool.QueryRow(ctx, q, strings.ToLower(strings.TrimSpace(slug))))
}

func (r *AccountRepo) List(ctx context.Context) ([]domain.Account, error) {
	q := `SELECT ` + accountColumns + ` FROM accounts ORDER BY id ASC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *AccountRepo) Create(ctx context.Context, a *domain.Account) error {
	const q = `INSERT INTO accounts (name, slug, enabled)
	           VALUES ($1, $2, $3)
	           RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, q,
		strings.TrimSpace(a.Name),
		strings.ToLower(strings.TrimSpace(a.Slug)),
		a.Enabled,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AccountRepo) Update(ctx context.Context, a *domain.Account) error {
	const q = `UPDATE accounts
	           SET name = $1, slug = $2, enabled = $3, updated_at = NOW()
	           WHERE id = $4
	           RETURNING updated_at`
	return r.pool.QueryRow(ctx, q,
		strings.TrimSpace(a.Name),
		strings.ToLower(strings.TrimSpace(a.Slug)),
		a.Enabled,
		a.ID,
	).Scan(&a.UpdatedAt)
}

// Delete só funciona se não houver pixels/admins associados.
// Se houver, retorna erro do FK (ON DELETE RESTRICT).
func (r *AccountRepo) Delete(ctx context.Context, id int64) error {
	if id == 1 {
		return errors.New("conta default não pode ser removida")
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, id)
	return err
}

func scanAccount(row rowScanner) (*domain.Account, error) {
	var a domain.Account
	err := row.Scan(&a.ID, &a.Name, &a.Slug, &a.Enabled, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
