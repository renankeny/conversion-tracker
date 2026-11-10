package repo

import (
	"context"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// GetGroupSessions retorna todas as sessões (tracked_users) da mesma pessoa (mesmo group_id
// ou mesmo externalID), ordenadas cronologicamente pelo primeiro acesso (created_at ASC).
func (r *UserRepo) GetGroupSessions(ctx context.Context, accountID int64, externalID string) ([]domain.TrackedUser, error) {
	u, err := r.FindByExternalIDByAccount(ctx, externalID, accountID)
	if err != nil {
		return nil, err
	}
	if u.GroupID == nil {
		if (u.Email != "" || u.Phone != "") && (accountID > 0 || u.AccountID > 0) {
			effAccID := accountID
			if effAccID == 0 {
				effAccID = u.AccountID
			}
			qFallback := `SELECT ` + userColumns + ` FROM tracked_users 
			              WHERE (account_id = $1 OR $1 = 0) AND ((email = $2 AND $2 <> '') OR (phone = $3 AND $3 <> ''))
			              ORDER BY created_at ASC`
			rows, err := r.pool.Query(ctx, qFallback, effAccID, u.Email, u.Phone)
			if err == nil {
				defer rows.Close()
				var sessions []domain.TrackedUser
				for rows.Next() {
					if s, err := scanUser(rows); err == nil {
						sessions = append(sessions, *s)
					}
				}
				if len(sessions) > 0 {
					return sessions, nil
				}
			}
		}
		return []domain.TrackedUser{*u}, nil
	}

	q := `SELECT ` + userColumns + ` FROM tracked_users 
	      WHERE group_id = $1 AND (account_id = $2 OR $2 = 0) 
	      ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, q, *u.GroupID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []domain.TrackedUser
	for rows.Next() {
		s, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *s)
	}
	if len(sessions) == 0 {
		return []domain.TrackedUser{*u}, nil
	}
	return sessions, nil
}

// TrackedUserFilter agrupa os filtros aceitos por List.
type TrackedUserFilter struct {
	AccountID int64  // 0 = todos (owner); >0 = filtra por conta (member)
	PixelName string // exato (vazio = todos)
	Search    string // busca em external_id/id_wpp/email/phone (LIKE %s%)
	Limit     int    // default 50, max 500
	Offset    int
}

// List devolve tracked users filtrados + total count (pra paginação).
// Usado pelo painel admin.
func (r *UserRepo) List(ctx context.Context, f TrackedUserFilter) ([]domain.TrackedUser, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 500 {
		f.Limit = 500
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	args := make([]any, 0, 4)
	conds := make([]string, 0, 3)
	idx := 0
	addArg := func(value any) string {
		args = append(args, value)
		idx++
		return "$" + itoa(idx)
	}
	if f.AccountID != 0 {
		conds = append(conds, "account_id = "+addArg(f.AccountID))
	}
	if f.PixelName != "" {
		conds = append(conds, "pixel_name = "+addArg(f.PixelName))
	}
	if f.Search != "" {
		ph2 := addArg("%" + f.Search + "%")
		conds = append(conds, "(external_id ILIKE "+ph2+" OR id_wpp ILIKE "+ph2+
			" OR email ILIKE "+ph2+" OR phone ILIKE "+ph2+")")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}

	var total int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(DISTINCT COALESCE(group_id, id)) FROM tracked_users "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	q := `SELECT ` + userColumns + `
		FROM (
			SELECT DISTINCT ON (COALESCE(group_id, id)) *
			FROM tracked_users ` + where + `
			ORDER BY COALESCE(group_id, id), updated_at DESC
		) grouped
		ORDER BY id DESC LIMIT $` + itoa(idx+1) + ` OFFSET $` + itoa(idx+2)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.TrackedUser, 0, f.Limit)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *u)
	}
	if len(out) == 0 {
		return out, total, nil
	}

	extIDs := make([]string, len(out))
	gkeyByExtID := make(map[string]int64, len(out))
	for i := range out {
		extIDs[i] = out[i].ExternalID
		gkey := out[i].ID
		if out[i].GroupID != nil {
			gkey = *out[i].GroupID
		}
		gkeyByExtID[out[i].ExternalID] = gkey
	}

	ltvRows, err := r.pool.Query(ctx, `
		WITH keys AS (
			SELECT DISTINCT account_id, COALESCE(group_id, id) AS gkey
			FROM tracked_users
			WHERE external_id = ANY($1::text[])
			  AND ($2::bigint = 0 OR account_id = $2)
		),
		members AS (
			SELECT t.account_id, t.external_id, COALESCE(t.group_id, t.id) AS gkey
			FROM tracked_users t
			JOIN keys k ON k.gkey = COALESCE(t.group_id, t.id) AND k.account_id = t.account_id
		)
		SELECT m.gkey, COALESCE(SUM(s.max_value), 0.0)
		FROM members m
		LEFT JOIN (
			SELECT se.account_id, se.external_id, se.event_id, MAX(se.purchase_value) AS max_value
			FROM sent_events se
			WHERE se.event_name = 'Purchase'
			  AND se.status = 'sent'
			  AND ($2::bigint = 0 OR se.account_id = $2)
			  AND se.external_id IN (SELECT external_id FROM members)
			GROUP BY se.account_id, se.external_id, se.event_id
		) s ON s.external_id = m.external_id AND s.account_id = m.account_id
		GROUP BY m.gkey
	`, extIDs, f.AccountID)
	if err == nil {
		defer ltvRows.Close()
		ltvByGKey := make(map[int64]float64, len(out))
		for ltvRows.Next() {
			var gkey int64
			var ltvVal float64
			if err := ltvRows.Scan(&gkey, &ltvVal); err == nil {
				ltvByGKey[gkey] = ltvVal
			}
		}
		for i := range out {
			out[i].LTV = ltvByGKey[gkeyByExtID[out[i].ExternalID]]
		}
	}

	return out, total, rows.Err()
}

func joinAnd(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += " AND " + parts[i]
	}
	return out
}
