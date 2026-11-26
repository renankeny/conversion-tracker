package repo

import (
	"context"
	"time"

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
	Segment   string // identified | buyers | anonymous | bot (vazio = todos)
	Search    string // busca em external_id, id_wpp, email, phone, fbp e nome
	Limit     int    // default 50, max 500
	Offset    int
}

// List devolve tracked users filtrados + total count (para paginação no painel admin).
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

	where, args, idx := buildTrackedUserListWhere(f)

	var total int64
	countQ := "SELECT COUNT(*) FROM (SELECT DISTINCT account_id, COALESCE(group_id, -id) FROM tracked_users " + where + ") cnt"
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	q := `SELECT ` + userColumns + `
		FROM (
			SELECT DISTINCT ON (account_id, COALESCE(group_id, -id)) *
			FROM tracked_users ` + where + `
			ORDER BY account_id, COALESCE(group_id, -id), updated_at DESC, id DESC
		) grouped
		ORDER BY updated_at DESC, id DESC LIMIT $` + itoa(idx+1) + ` OFFSET $` + itoa(idx+2)

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
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(out) == 0 {
		return out, total, nil
	}

	enrichTrackedUsersSlice(ctx, r.pool, out)

	type accGKey struct {
		accountID int64
		gkey      int64
	}
	extIDs := make([]string, len(out))
	keyByIdx := make([]accGKey, len(out))
	for i := range out {
		extIDs[i] = out[i].ExternalID
		out[i].TimeToPurchase = "Sem compra"
		gkey := -out[i].ID
		if out[i].GroupID != nil {
			gkey = *out[i].GroupID
		}
		keyByIdx[i] = accGKey{accountID: out[i].AccountID, gkey: gkey}
	}

	ltvRows, err := r.pool.Query(ctx, `
		WITH keys AS (
			SELECT DISTINCT account_id, COALESCE(group_id, -id) AS gkey
			FROM tracked_users
			WHERE external_id = ANY($1::text[])
			  AND ($2::bigint = 0 OR account_id = $2)
		),
		members AS (
			SELECT t.id AS user_id, t.account_id, t.external_id, t.created_at, COALESCE(t.group_id, -t.id) AS gkey
			FROM tracked_users t
			JOIN keys k ON k.gkey = COALESCE(t.group_id, -t.id) AND k.account_id = t.account_id
		),
		refunds_by_gkey AS (
			SELECT sr.account_id, COALESCE(sr.group_id, -sr.tracked_user_id) AS gkey, COALESCE(SUM(sr.amount), 0.0) AS ref_amt
			FROM sale_refunds sr
			WHERE ($2::bigint = 0 OR sr.account_id = $2)
			  AND (sr.tracked_user_id IS NOT NULL OR sr.group_id IS NOT NULL)
			GROUP BY sr.account_id, COALESCE(sr.group_id, -sr.tracked_user_id)
		)
		SELECT m.account_id, m.gkey,
		       GREATEST(0.0, COALESCE(SUM(s.max_value), 0.0) - COALESCE(MAX(rf.ref_amt), 0.0)),
		       MIN(m.created_at), MIN(s.first_purchase_at)
		FROM members m
		LEFT JOIN (
			SELECT se.account_id, se.external_id, se.event_id,
			       MAX(se.purchase_value) AS max_value,
			       MIN(se.created_at) AS first_purchase_at
			FROM sent_events se
			WHERE se.event_name = 'Purchase'
			  AND se.status = 'sent'
			  AND ($2::bigint = 0 OR se.account_id = $2)
			  AND se.external_id IN (SELECT external_id FROM members)
			GROUP BY se.account_id, se.external_id, se.event_id
		) s ON s.external_id = m.external_id AND s.account_id = m.account_id
		LEFT JOIN refunds_by_gkey rf ON rf.account_id = m.account_id AND rf.gkey = m.gkey
		GROUP BY m.account_id, m.gkey
	`, extIDs, f.AccountID)
	if err == nil {
		defer ltvRows.Close()
		type ltvSummary struct {
			ltv float64
			ttp string
		}
		byKey := make(map[accGKey]ltvSummary, len(out))
		for ltvRows.Next() {
			var accID, gkey int64
			var ltvVal float64
			var firstSeen time.Time
			var firstPurchase *time.Time
			if err := ltvRows.Scan(&accID, &gkey, &ltvVal, &firstSeen, &firstPurchase); err == nil {
				ttp := "Sem compra"
				if firstPurchase != nil && !firstPurchase.IsZero() {
					diff := firstPurchase.Sub(firstSeen)
					if diff < 0 {
						diff = 0
					}
					ttp = formatDuration(diff)
				}
				byKey[accGKey{accountID: accID, gkey: gkey}] = ltvSummary{ltv: ltvVal, ttp: ttp}
			}
		}
		for i := range out {
			if sum, ok := byKey[keyByIdx[i]]; ok {
				out[i].LTV = sum.ltv
				out[i].TimeToPurchase = sum.ttp
			}
		}
	}

	return out, total, nil
}
