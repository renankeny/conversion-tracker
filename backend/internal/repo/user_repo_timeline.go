package repo

import (
	"context"
	"sort"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// groupExternalIDs devolve todos os external_id do mesmo grupo de u (ou só o
// próprio u.ExternalID, se GroupID for nil — nunca vinculado).
func (r *UserRepo) groupExternalIDs(ctx context.Context, u *domain.TrackedUser) []string {
	if u.GroupID != nil {
		rows, err := r.pool.Query(ctx, `SELECT external_id FROM tracked_users WHERE group_id = $1 AND ($2 = 0 OR account_id = $2)`, *u.GroupID, u.AccountID)
		if err == nil {
			defer rows.Close()
			ids := make([]string, 0, 4)
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err == nil {
					ids = append(ids, id)
				}
			}
			if len(ids) > 0 {
				return ids
			}
		}
	}
	// Fallback por e-mail ou telefone se group_id não estiver setado
	if (u.Email != "" || u.Phone != "") && u.AccountID > 0 {
		rows, err := r.pool.Query(ctx, `
			SELECT external_id FROM tracked_users 
			WHERE account_id = $1 AND ((email = $2 AND $2 <> '') OR (phone = $3 AND $3 <> ''))
		`, u.AccountID, u.Email, u.Phone)
		if err == nil {
			defer rows.Close()
			ids := make([]string, 0, 4)
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err == nil {
					ids = append(ids, id)
				}
			}
			if len(ids) > 0 {
				return ids
			}
		}
	}
	return []string{u.ExternalID}
}

// ResolveAndLink busca tracked_users existentes por qualquer combinação de
// external_id/phone/email informada (phone/email já normalizados pelo
// chamador — mesma convenção do resto do repo). Não para no primeiro match:
// se mais de um registro distinto bater, liga todos ao mesmo group_id (cria
// um novo se nenhum tiver, reaproveita se algum já tiver, funde dois grupos
// se cada um já tinha o seu). Nada em tracked_users além do group_id é
// alterado — sent_events/webhook_logs nunca são tocados.
//
// Devolve o registro que corresponde ao externalID pedido, se estiver entre
// os batidos; senão o mais recentemente atualizado dos que bateram (ex:
// telefone bateu mas o external_id do payload atual é novo). ErrNotFound se
// nenhum dos três identificadores encontrar nada — comportamento igual ao
// dos Find* de hoje, quem chama decide se cria um registro novo.
func (r *UserRepo) ResolveAndLink(ctx context.Context, accountID int64, externalID, phone, email string) (*domain.TrackedUser, error) {
	if externalID == "" && phone == "" && email == "" {
		return nil, ErrNotFound
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	args := []any{accountID}
	conds := make([]string, 0, 3)
	if externalID != "" {
		args = append(args, externalID)
		conds = append(conds, "external_id = $"+itoa(len(args)))
	}
	if phone != "" {
		args = append(args, phone)
		conds = append(conds, "(phone <> '' AND phone = $"+itoa(len(args))+")")
	}
	if email != "" {
		args = append(args, email)
		conds = append(conds, "(email <> '' AND email = $"+itoa(len(args))+")")
	}

	q := `SELECT ` + userColumns + ` FROM tracked_users
	      WHERE account_id = $1 AND (` + strings.Join(conds, " OR ") + `)
	      ORDER BY id ASC
	      FOR UPDATE`
	rows, err := tx.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	matches := make([]domain.TrackedUser, 0, 2)
	for rows.Next() {
		m, err := scanUser(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		matches = append(matches, *m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, ErrNotFound
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].UpdatedAt.After(matches[j].UpdatedAt)
	})

	pick := func() *domain.TrackedUser {
		for i := range matches {
			if externalID != "" && matches[i].ExternalID == externalID {
				return &matches[i]
			}
		}
		return &matches[0] // mais recentemente atualizado (ORDER BY updated_at DESC)
	}

	if len(matches) == 1 {
		// Só um bateu — nada pra linkar ainda.
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return pick(), nil
	}

	// Grupos já existentes entre os batidos.
	groupSet := map[int64]bool{}
	matchedUserIDs := make([]int64, 0, len(matches))
	for _, m := range matches {
		matchedUserIDs = append(matchedUserIDs, m.ID)
		if m.GroupID != nil {
			groupSet[*m.GroupID] = true
		}
	}

	var canonicalGroupID int64
	involvedGroups := make([]int64, 0, len(groupSet)+1)
	switch len(groupSet) {
	case 0:
		if err := tx.QueryRow(ctx,
			`INSERT INTO groups (account_id) VALUES ($1) RETURNING id`, accountID,
		).Scan(&canonicalGroupID); err != nil {
			return nil, err
		}
		involvedGroups = append(involvedGroups, canonicalGroupID)
	case 1:
		for gid := range groupSet {
			canonicalGroupID = gid
			involvedGroups = append(involvedGroups, gid)
		}
	default:
		ids := make([]int64, 0, len(groupSet))
		for gid := range groupSet {
			ids = append(ids, gid)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		canonicalGroupID = ids[0]
		involvedGroups = append(involvedGroups, ids...)
	}

	if err := consolidateGroupCRMLeads(ctx, tx, accountID, canonicalGroupID, matchedUserIDs, involvedGroups); err != nil {
		return nil, err
	}
	for _, gid := range involvedGroups {
		if gid == canonicalGroupID {
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE tracked_users SET group_id = $1 WHERE group_id = $2 AND account_id = $3`, canonicalGroupID, gid, accountID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `UPDATE crm_leads SET group_id = $1 WHERE group_id = $2 AND account_id = $3`, canonicalGroupID, gid, accountID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM groups WHERE id = $1 AND account_id = $2`, gid, accountID); err != nil {
			return nil, err
		}
	}

	needUpdateIDs := make([]int64, 0, len(matches))
	for _, m := range matches {
		if m.GroupID == nil || *m.GroupID != canonicalGroupID {
			needUpdateIDs = append(needUpdateIDs, m.ID)
		}
	}
	if len(needUpdateIDs) > 0 {
		if _, err := tx.Exec(ctx,
			`UPDATE tracked_users SET group_id = $1, updated_at = NOW() WHERE id = ANY($2::bigint[]) AND account_id = $3`,
			canonicalGroupID, needUpdateIDs, accountID,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	result := pick()
	result.GroupID = &canonicalGroupID
	enrichTrackedUserGroupFields(ctx, r.pool, result)
	return result, nil
}
