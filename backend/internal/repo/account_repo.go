package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type AccountRepo struct {
	pool *pgxpool.Pool
}

func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool}
}

const accountColumns = `id, name, slug, enabled, owner_id,
	(SELECT COUNT(*) FROM pixels p WHERE p.account_id = accounts.id) AS pixels_count,
	(SELECT COUNT(*) FROM admin_users u WHERE u.account_id = accounts.id) AS users_count,
	sent_events_retention_days, event_counters_retention_days,
	event_label_overrides, events_filter, created_at, updated_at`

func (r *AccountRepo) FindByID(ctx context.Context, id int64) (*domain.Account, error) {
	q := `SELECT ` + accountColumns + ` FROM accounts WHERE id = $1`
	return scanAccount(r.pool.QueryRow(ctx, q, id))
}

func (r *AccountRepo) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *AccountRepo) FindBySlug(ctx context.Context, slug string) (*domain.Account, error) {
	q := `SELECT ` + accountColumns + ` FROM accounts WHERE slug = $1`
	return scanAccount(r.pool.QueryRow(ctx, q, strings.ToLower(strings.TrimSpace(slug))))
}

func (r *AccountRepo) List(ctx context.Context) ([]domain.Account, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+accountColumns+` FROM accounts ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectAccounts(rows)
}

func (r *AccountRepo) ListByHierarchy(ctx context.Context, c *auth.Claims) ([]domain.Account, error) {
	if c == nil {
		return []domain.Account{}, nil
	}
	if c.IsOwner() {
		return r.List(ctx)
	}

	var rows pgx.Rows
	var err error
	if c.IsAPIKey() || c.Role == "employee" || c.Role == "member" {
		if c.AccountID <= 0 {
			return []domain.Account{}, nil
		}
		q := `SELECT ` + accountColumns + ` FROM accounts WHERE id = $1 ORDER BY id ASC`
		rows, err = r.pool.Query(ctx, q, c.AccountID)
	} else {
		q := `SELECT ` + accountColumns + ` FROM accounts
		      WHERE owner_id = $1
		         OR owner_id IN (SELECT id FROM admin_users WHERE parent_id = $1)
		         OR id = $2
		         OR id = (SELECT COALESCE(account_id, 0) FROM admin_users WHERE id = $1)
		      ORDER BY id ASC`
		rows, err = r.pool.Query(ctx, q, c.UserID, c.AccountID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectAccounts(rows)
}

func (r *AccountRepo) CanAccess(ctx context.Context, c *auth.Claims, accountID int64) bool {
	if c == nil || accountID <= 0 {
		return false
	}
	if c.IsAPIKey() {
		return c.AccountID == accountID
	}
	if c.IsOwner() || c.AccountID == accountID {
		return true
	}
	if c.Role == "employee" || c.Role == "member" {
		return false
	}
	var ok bool
	const q = `SELECT EXISTS(
		SELECT 1 FROM accounts
		WHERE id = $1 AND (
			owner_id = $2
			OR owner_id IN (SELECT id FROM admin_users WHERE parent_id = $2)
			OR id = (SELECT COALESCE(account_id, 0) FROM admin_users WHERE id = $2)
		)
	)`
	_ = r.pool.QueryRow(ctx, q, accountID, c.UserID).Scan(&ok)
	return ok
}

func collectAccounts(rows pgx.Rows) ([]domain.Account, error) {
	out := make([]domain.Account, 0)
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
	const q = `INSERT INTO accounts (name, slug, enabled, owner_id, sent_events_retention_days, event_counters_retention_days)
	           VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, q,
		strings.TrimSpace(a.Name), strings.ToLower(strings.TrimSpace(a.Slug)),
		a.Enabled, a.OwnerID, a.SentEventsRetentionDays, a.EventCountersRetentionDays,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AccountRepo) Update(ctx context.Context, a *domain.Account) error {
	const q = `UPDATE accounts
	           SET name = $1, slug = $2, enabled = $3,
	               sent_events_retention_days = $4, event_counters_retention_days = $5,
	               updated_at = NOW()
	           WHERE id = $6 RETURNING updated_at`
	return r.pool.QueryRow(ctx, q,
		strings.TrimSpace(a.Name), strings.ToLower(strings.TrimSpace(a.Slug)), a.Enabled,
		a.SentEventsRetentionDays, a.EventCountersRetentionDays, a.ID,
	).Scan(&a.UpdatedAt)
}

func (r *AccountRepo) Delete(ctx context.Context, id int64) error {
	if id == 1 {
		return errors.New("a empresa principal (ID #1) não pode ser removida")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cleanupQueries := []string{
		`DELETE FROM sent_events WHERE account_id = $1`,
		`DELETE FROM event_counters WHERE account_id = $1`,
		`DELETE FROM url_mappings WHERE account_id = $1`,
		`DELETE FROM whatsapp_instances WHERE account_id = $1`,
		`DELETE FROM tracked_users WHERE account_id = $1`,
		`DELETE FROM pixels WHERE account_id = $1`,
		`DELETE FROM google_ads_accounts WHERE account_id = $1`,
	}
	for _, q := range cleanupQueries {
		if _, err := tx.Exec(ctx, q, id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func scanAccount(row rowScanner) (*domain.Account, error) {
	var a domain.Account
	var labelOverrides, eventsFilter []byte
	err := row.Scan(
		&a.ID, &a.Name, &a.Slug, &a.Enabled, &a.OwnerID,
		&a.PixelsCount, &a.UsersCount,
		&a.SentEventsRetentionDays, &a.EventCountersRetentionDays,
		&labelOverrides, &eventsFilter, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.EventLabelOverrides = map[string]string{}
	if len(labelOverrides) > 0 {
		_ = json.Unmarshal(labelOverrides, &a.EventLabelOverrides)
	}
	a.EventsFilter = map[string][]string{}
	if len(eventsFilter) > 0 {
		_ = json.Unmarshal(eventsFilter, &a.EventsFilter)
	}
	return &a, nil
}
