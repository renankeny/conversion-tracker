package repo

import (
	"context"
	"fmt"
)

func (r *AdminUserRepo) Create(ctx context.Context, u *AdminUserRecord) error {
	const q = `INSERT INTO admin_users (email, password_hash, name, phone, role, account_id, parent_id, permissions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, q,
		u.Email, u.PasswordHash, u.Name, u.Phone, u.Role, u.AccountID, u.ParentID, u.Permissions,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create admin_user: %w", err)
	}
	return nil
}

func (r *AdminUserRepo) Update(
	ctx context.Context,
	id int64,
	email, name, phone, role string,
	accountID *int64,
	perms []byte,
	passwordHash string,
) error {
	if passwordHash != "" {
		_, err := r.pool.Exec(ctx,
			`UPDATE admin_users
			 SET email = $1, name = $2, phone = $3, role = $4, account_id = $5, permissions = $6, password_hash = $7, updated_at = NOW()
			 WHERE id = $8`,
			email, name, phone, role, accountID, perms, passwordHash, id,
		)
		if err != nil {
			return fmt.Errorf("update admin_user with password: %w", err)
		}
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE admin_users
		 SET email = $1, name = $2, phone = $3, role = $4, account_id = $5, permissions = $6, updated_at = NOW()
		 WHERE id = $7`,
		email, name, phone, role, accountID, perms, id,
	)
	if err != nil {
		return fmt.Errorf("update admin_user: %w", err)
	}
	return nil
}

func (r *AdminUserRepo) Delete(ctx context.Context, id int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM admin_users WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete admin_user: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *AdminUserRepo) UpdateProfile(ctx context.Context, id int64, name, phone string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE admin_users SET name = $1, phone = $2, updated_at = NOW() WHERE id = $3`,
		name, phone, id,
	)
	if err != nil {
		return false, fmt.Errorf("update admin_user profile: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *AdminUserRepo) UpdatePasswordHash(ctx context.Context, id int64, newHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE admin_users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		newHash, id,
	)
	if err != nil {
		return fmt.Errorf("update admin_user password_hash: %w", err)
	}
	return nil
}

func (r *AdminUserRepo) UpdatePreferences(ctx context.Context, id int64, prefs []byte) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE admin_users SET preferences = $1, updated_at = NOW() WHERE id = $2`,
		prefs, id,
	)
	if err != nil {
		return fmt.Errorf("update admin_user preferences: %w", err)
	}
	return nil
}

func (r *AdminUserRepo) EnsureBootstrap(ctx context.Context, email, passwordHash string) error {
	n, err := r.Count(ctx)
	if err != nil || n > 0 {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO admin_users (email, password_hash, role, account_id) VALUES ($1, $2, 'superadmin', 1)`,
		email, passwordHash,
	)
	if err != nil {
		return fmt.Errorf("bootstrap admin_user: %w", err)
	}
	return nil
}

