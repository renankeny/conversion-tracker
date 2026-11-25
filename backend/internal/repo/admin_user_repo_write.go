package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func syncUserAccountsTx(ctx context.Context, tx pgx.Tx, userID int64, accountIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM admin_user_accounts WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clear admin_user_accounts: %w", err)
	}
	if len(accountIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO admin_user_accounts (user_id, account_id)
		SELECT $1, UNNEST($2::bigint[])
		ON CONFLICT (user_id, account_id) DO NOTHING
	`, userID, accountIDs)
	if err != nil {
		return fmt.Errorf("insert admin_user_accounts: %w", err)
	}
	return nil
}

func primaryAccountPtr(accountIDs []int64, fallback *int64) *int64 {
	if len(accountIDs) > 0 && accountIDs[0] > 0 {
		v := accountIDs[0]
		return &v
	}
	return fallback
}

func (r *AdminUserRepo) Create(ctx context.Context, u *AdminUserRecord) error {
	u.AccountID = primaryAccountPtr(u.AccountIDs, u.AccountID)
	if len(u.AccountIDs) == 0 && u.AccountID != nil && *u.AccountID > 0 {
		u.AccountIDs = []int64{*u.AccountID}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create admin_user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `INSERT INTO admin_users (email, password_hash, name, phone, role, account_id, parent_id, permissions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	if err := tx.QueryRow(ctx, q,
		u.Email, u.PasswordHash, u.Name, u.Phone, u.Role, u.AccountID, u.ParentID, u.Permissions,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return fmt.Errorf("create admin_user: %w", err)
	}
	if err := syncUserAccountsTx(ctx, tx, u.ID, u.AccountIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AdminUserRepo) Update(
	ctx context.Context,
	id int64,
	email, name, phone, role string,
	accountIDs []int64,
	perms []byte,
	passwordHash string,
) error {
	primaryAcc := primaryAccountPtr(accountIDs, nil)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update admin_user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if passwordHash != "" {
		_, err = tx.Exec(ctx,
			`UPDATE admin_users
			 SET email = $1, name = $2, phone = $3, role = $4, account_id = $5, permissions = $6, password_hash = $7, updated_at = NOW()
			 WHERE id = $8`,
			email, name, phone, role, primaryAcc, perms, passwordHash, id,
		)
	} else {
		_, err = tx.Exec(ctx,
			`UPDATE admin_users
			 SET email = $1, name = $2, phone = $3, role = $4, account_id = $5, permissions = $6, updated_at = NOW()
			 WHERE id = $7`,
			email, name, phone, role, primaryAcc, perms, id,
		)
	}
	if err != nil {
		return fmt.Errorf("update admin_user: %w", err)
	}
	if err := syncUserAccountsTx(ctx, tx, id, accountIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
	var newID int64
	err = r.pool.QueryRow(ctx,
		`INSERT INTO admin_users (email, password_hash, role, account_id) VALUES ($1, $2, 'superadmin', 1) RETURNING id`,
		email, passwordHash,
	).Scan(&newID)
	if err != nil {
		return fmt.Errorf("bootstrap admin_user: %w", err)
	}
	_, _ = r.pool.Exec(ctx,
		`INSERT INTO admin_user_accounts (user_id, account_id) VALUES ($1, 1) ON CONFLICT DO NOTHING`,
		newID,
	)
	return nil
}
