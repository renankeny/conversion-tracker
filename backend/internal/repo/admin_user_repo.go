package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminUserRecord representa uma linha da tabela admin_users para autenticação e gestão.
type AdminUserRecord struct {
	ID           int64
	Email        string
	PasswordHash string
	Name         string
	Phone        string
	Role         string
	AccountID    *int64
	ParentID     *int64
	Permissions  []byte
	Preferences  []byte
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AdminUserRepo centraliza o acesso SQL à tabela admin_users.
type AdminUserRepo struct {
	pool *pgxpool.Pool
}

func NewAdminUserRepo(pool *pgxpool.Pool) *AdminUserRepo {
	return &AdminUserRepo{pool: pool}
}

func (r *AdminUserRepo) ListSubordinates(ctx context.Context, isOwner bool, userID int64) ([]AdminUserRecord, error) {
	const q = `SELECT id, email, COALESCE(name, ''), COALESCE(phone, ''),
		COALESCE(role, 'admin'), account_id, parent_id, permissions, created_at, updated_at
		FROM admin_users
		WHERE $1::boolean OR parent_id = $2 OR id = $2
		ORDER BY id ASC`

	rows, err := r.pool.Query(ctx, q, isOwner, userID)
	if err != nil {
		return nil, fmt.Errorf("list admin_users: %w", err)
	}
	defer rows.Close()

	var out []AdminUserRecord
	for rows.Next() {
		var u AdminUserRecord
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Name, &u.Phone, &u.Role,
			&u.AccountID, &u.ParentID, &u.Permissions, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan admin_users: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *AdminUserRepo) FindByID(ctx context.Context, id int64) (*AdminUserRecord, error) {
	const q = `SELECT id, email, COALESCE(name, ''), COALESCE(phone, ''),
		COALESCE(role, 'admin'), account_id, parent_id, permissions, created_at, updated_at
		FROM admin_users WHERE id = $1`
	var u AdminUserRecord
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.Name, &u.Phone, &u.Role,
		&u.AccountID, &u.ParentID, &u.Permissions, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find admin_user by id: %w", err)
	}
	return &u, nil
}

func (r *AdminUserRepo) FindByEmailForAuth(ctx context.Context, email string) (*AdminUserRecord, error) {
	const q = `SELECT id, password_hash, COALESCE(role, 'superadmin'), account_id,
		parent_id, permissions, COALESCE(name, ''), COALESCE(phone, ''), preferences
		FROM admin_users WHERE email = $1`
	var u AdminUserRecord
	u.Email = email
	err := r.pool.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.PasswordHash, &u.Role, &u.AccountID,
		&u.ParentID, &u.Permissions, &u.Name, &u.Phone, &u.Preferences,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find admin_user by email: %w", err)
	}
	return &u, nil
}

func (r *AdminUserRepo) GetRoleAndParent(ctx context.Context, id int64) (string, *int64, error) {
	var role string
	var parentID *int64
	err := r.pool.QueryRow(ctx, `SELECT role, parent_id FROM admin_users WHERE id = $1`, id).Scan(&role, &parentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("get admin_user role: %w", err)
	}
	return role, parentID, nil
}

func (r *AdminUserRepo) GetPasswordHash(ctx context.Context, id int64) (string, error) {
	var hash string
	err := r.pool.QueryRow(ctx, `SELECT password_hash FROM admin_users WHERE id = $1`, id).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get admin_user password_hash: %w", err)
	}
	return hash, nil
}

func (r *AdminUserRepo) GetProfileAndPrefs(ctx context.Context, id int64) (string, string, []byte, error) {
	var name, phone string
	var prefs []byte
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(name, ''), COALESCE(phone, ''), preferences FROM admin_users WHERE id = $1`, id,
	).Scan(&name, &phone, &prefs)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil, ErrNotFound
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("get admin_user profile: %w", err)
	}
	return name, phone, prefs, nil
}

func (r *AdminUserRepo) GetPreferences(ctx context.Context, id int64) ([]byte, error) {
	if r == nil || r.pool == nil || id <= 0 {
		return nil, nil
	}
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT preferences FROM admin_users WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get admin_user preferences: %w", err)
	}
	return raw, nil
}

func (r *AdminUserRepo) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count admin_users: %w", err)
	}
	return n, nil
}
