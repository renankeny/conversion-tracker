package repo

import (
	"context"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// FindByProtocolByAccount localiza um visitante pelo código de protocolo do tracker.js
// casando tanto external_id (exato ou uppercase) quanto id_wpp dentro da conta.
func (r *UserRepo) FindByProtocolByAccount(ctx context.Context, code string, accountID int64) (*domain.TrackedUser, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrNotFound
	}
	upper := strings.ToUpper(code)
	q := `SELECT ` + userColumns + ` FROM tracked_users
	      WHERE account_id = $1
	        AND (external_id = $2 OR external_id = $3 OR id_wpp = $3)
	      ORDER BY updated_at DESC LIMIT 1`
	return scanUser(r.pool.QueryRow(ctx, q, accountID, code, upper))
}

// FindByWaLIDByAccount localiza um visitante pelo identificador @lid dentro da conta.
func (r *UserRepo) FindByWaLIDByAccount(ctx context.Context, lid string, accountID int64) (*domain.TrackedUser, error) {
	lid = strings.TrimSpace(lid)
	if lid == "" {
		return nil, ErrNotFound
	}
	q := `SELECT ` + userColumns + ` FROM tracked_users
	      WHERE account_id = $1 AND wa_lid = $2 AND wa_lid <> ''
	      ORDER BY updated_at DESC LIMIT 1`
	return scanUser(r.pool.QueryRow(ctx, q, accountID, lid))
}

