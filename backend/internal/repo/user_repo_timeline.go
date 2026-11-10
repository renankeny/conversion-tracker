package repo

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

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
	addCond := func(col, val string) {
		args = append(args, val)
		conds = append(conds, col+" = $"+itoa(len(args)))
	}
	if externalID != "" {
		addCond("external_id", externalID)
	}
	if phone != "" {
		addCond("phone", phone)
	}
	if email != "" {
		addCond("email", email)
	}

	q := `SELECT ` + userColumns + ` FROM tracked_users
	      WHERE account_id = $1 AND (` + strings.Join(conds, " OR ") + `)
	      ORDER BY updated_at DESC`
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
	for _, m := range matches {
		if m.GroupID != nil {
			groupSet[*m.GroupID] = true
		}
	}

	var canonicalGroupID int64
	switch len(groupSet) {
	case 0:
		if err := tx.QueryRow(ctx,
			`INSERT INTO groups (account_id) VALUES ($1) RETURNING id`, accountID,
		).Scan(&canonicalGroupID); err != nil {
			return nil, err
		}
	case 1:
		for gid := range groupSet {
			canonicalGroupID = gid
		}
	default:
		ids := make([]int64, 0, len(groupSet))
		for gid := range groupSet {
			ids = append(ids, gid)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		canonicalGroupID = ids[0]
		for _, losingGroupID := range ids[1:] {
			if err := mergeGroupCRMLeads(ctx, tx, accountID, canonicalGroupID, losingGroupID); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(ctx, `UPDATE tracked_users SET group_id = $1 WHERE group_id = $2`, canonicalGroupID, losingGroupID); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(ctx, `UPDATE crm_leads SET group_id = $1 WHERE group_id = $2`, canonicalGroupID, losingGroupID); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM groups WHERE id = $1`, losingGroupID); err != nil {
				return nil, err
			}
		}
	}

	for _, m := range matches {
		if m.GroupID == nil || *m.GroupID != canonicalGroupID {
			if _, err := tx.Exec(ctx,
				`UPDATE tracked_users SET group_id = $1, updated_at = NOW() WHERE id = $2`,
				canonicalGroupID, m.ID,
			); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	result := pick()
	result.GroupID = &canonicalGroupID
	return result, nil
}

// mergeGroupCRMLeads resolve a colisão rara de dois grupos que já tinham,
// cada um, seu próprio card no kanban antes de serem ligados: mantém o de
// stage mais avançado (maior order_index da coluna) e remove o outro. Sem
// colisão (só um dos dois grupos tem card, ou nenhum), não faz nada — o
// UPDATE de group_id feito depois por ResolveAndLink já resolve sozinho.
func mergeGroupCRMLeads(ctx context.Context, tx pgx.Tx, accountID, canonicalGroupID, losingGroupID int64) error {
	findLead := func(groupID int64) (id int64, order int, found bool, err error) {
		err = tx.QueryRow(ctx, `
			SELECT cl.id, COALESCE(cs.order_index, -1)
			FROM crm_leads cl LEFT JOIN crm_stages cs ON cs.id = cl.stage_id
			WHERE cl.account_id = $1 AND cl.group_id = $2
		`, accountID, groupID).Scan(&id, &order)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, false, nil
		}
		return id, order, err == nil, err
	}

	canonicalID, canonicalOrder, canonicalFound, err := findLead(canonicalGroupID)
	if err != nil {
		return err
	}
	losingID, losingOrder, losingFound, err := findLead(losingGroupID)
	if err != nil {
		return err
	}
	if !canonicalFound || !losingFound {
		return nil // sem colisão — no máximo um card entre os dois grupos
	}

	loserID := canonicalID
	if losingOrder > canonicalOrder {
		loserID = losingID
	}
	_, err = tx.Exec(ctx, `DELETE FROM crm_leads WHERE id = $1`, loserID)
	return err
}
