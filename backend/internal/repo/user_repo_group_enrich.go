package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// enrichTrackedUserGroupFields consolida em memória os cookies de plataforma,
// click IDs e UTMs entre todas as sessões (tracked_users) da mesma pessoa
// (mesmo group_id ou mesmo telefone/e-mail na conta), sem sobrescrever as
// linhas individuais de sessão no banco.
//
// Regra de precedência:
//   - Click IDs, cookies e last_utm_*: vence o valor não-vazio mais recente (updated_at DESC).
//   - first_utm_* e first_url: vence o valor não-vazio mais antigo (created_at ASC).
//   - PII/Geo: preenche apenas campos vazios em u.
func enrichTrackedUserGroupFields(ctx context.Context, pool *pgxpool.Pool, u *domain.TrackedUser) {
	if pool == nil || u == nil {
		return
	}

	var query string
	var args []any

	if u.GroupID != nil && *u.GroupID > 0 {
		query = `SELECT ` + userColumns + ` FROM tracked_users
		         WHERE group_id = $1 AND ($2 = 0 OR account_id = $2)
		         ORDER BY updated_at DESC, id DESC`
		args = []any{*u.GroupID, u.AccountID}
	} else if (u.Phone != "" || u.Email != "") && u.AccountID > 0 {
		query = `SELECT ` + userColumns + ` FROM tracked_users
		         WHERE account_id = $1
		           AND ((phone = $2 AND $2 <> '') OR (email = $3 AND $3 <> ''))
		         ORDER BY updated_at DESC, id DESC`
		args = []any{u.AccountID, u.Phone, u.Email}
	} else {
		return
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	members := make([]*domain.TrackedUser, 0, 4)
	for rows.Next() {
		m, err := scanUser(rows)
		if err != nil {
			return
		}
		members = append(members, m)
	}
	if len(members) <= 1 {
		return
	}

	applyConsolidatedGroupMembers(u, members)
}

// applyConsolidatedGroupMembers aplica a consolidação assumindo members ordenados por updated_at DESC.
func applyConsolidatedGroupMembers(u *domain.TrackedUser, members []*domain.TrackedUser) {
	newestNonEmpty := func(getter func(*domain.TrackedUser) string) string {
		for _, m := range members {
			if v := getter(m); v != "" {
				return v
			}
		}
		return ""
	}
	oldestNonEmpty := func(getter func(*domain.TrackedUser) string) string {
		for i := len(members) - 1; i >= 0; i-- {
			if v := getter(members[i]); v != "" {
				return v
			}
		}
		return ""
	}
	setLatest := func(dst *string, getter func(*domain.TrackedUser) string) {
		if v := newestNonEmpty(getter); v != "" {
			*dst = v
		}
	}
	fillEmpty := func(dst *string, getter func(*domain.TrackedUser) string) {
		if *dst == "" {
			if v := newestNonEmpty(getter); v != "" {
				*dst = v
			}
		}
	}

	// Click IDs e cookies de plataforma: sempre reflete o toque/cookie mais recente da pessoa
	setLatest(&u.FBP, func(m *domain.TrackedUser) string { return m.FBP })
	setLatest(&u.FBC, func(m *domain.TrackedUser) string { return m.FBC })
	setLatest(&u.GCLID, func(m *domain.TrackedUser) string { return m.GCLID })
	setLatest(&u.WBraid, func(m *domain.TrackedUser) string { return m.WBraid })
	setLatest(&u.GBraid, func(m *domain.TrackedUser) string { return m.GBraid })
	setLatest(&u.TTClid, func(m *domain.TrackedUser) string { return m.TTClid })
	setLatest(&u.TTP, func(m *domain.TrackedUser) string { return m.TTP })
	setLatest(&u.GA4ClientID, func(m *domain.TrackedUser) string { return m.GA4ClientID })
	setLatest(&u.GA4SessionID, func(m *domain.TrackedUser) string { return m.GA4SessionID })
	setLatest(&u.CtwaClid, func(m *domain.TrackedUser) string { return m.CtwaClid })
	setLatest(&u.WaAdID, func(m *domain.TrackedUser) string { return m.WaAdID })
	setLatest(&u.WaSourceApp, func(m *domain.TrackedUser) string { return m.WaSourceApp })

	// UTMs da última visita (mais recente; ignora "direto" quando houver origem real)
	if v := newestNonEmpty(func(m *domain.TrackedUser) string {
		if m.LastUTMSource == "direto" {
			return ""
		}
		return m.LastUTMSource
	}); v != "" {
		u.LastUTMSource = v
	} else {
		setLatest(&u.LastUTMSource, func(m *domain.TrackedUser) string { return m.LastUTMSource })
	}
	if u.LastUTMSource == "" || u.LastUTMSource == "direto" {
		if u.GCLID != "" || u.GBraid != "" || u.WBraid != "" {
			u.LastUTMSource = "google"
		} else if u.FBC != "" || u.CtwaClid != "" {
			u.LastUTMSource = "facebook"
		} else if u.TTClid != "" {
			u.LastUTMSource = "tiktok"
		}
	}
	setLatest(&u.LastUTMMedium, func(m *domain.TrackedUser) string { return m.LastUTMMedium })
	setLatest(&u.LastUTMCampaign, func(m *domain.TrackedUser) string { return m.LastUTMCampaign })
	setLatest(&u.LastUTMContent, func(m *domain.TrackedUser) string { return m.LastUTMContent })
	setLatest(&u.LastUTMTerm, func(m *domain.TrackedUser) string { return m.LastUTMTerm })
	setLatest(&u.URL, func(m *domain.TrackedUser) string { return m.URL })

	// UTMs da primeira visita (mais antiga) — preserva se u já tiver, ou busca o 1º toque do grupo
	if u.FirstUTMSource == "" {
		u.FirstUTMSource = oldestNonEmpty(func(m *domain.TrackedUser) string { return m.FirstUTMSource })
		u.FirstUTMMedium = oldestNonEmpty(func(m *domain.TrackedUser) string { return m.FirstUTMMedium })
		u.FirstUTMCampaign = oldestNonEmpty(func(m *domain.TrackedUser) string { return m.FirstUTMCampaign })
		u.FirstUTMContent = oldestNonEmpty(func(m *domain.TrackedUser) string { return m.FirstUTMContent })
		u.FirstUTMTerm = oldestNonEmpty(func(m *domain.TrackedUser) string { return m.FirstUTMTerm })
	}
	if u.FirstURL == "" {
		u.FirstURL = oldestNonEmpty(func(m *domain.TrackedUser) string { return m.FirstURL })
	}

	// Contato e localização de fallback
	fillEmpty(&u.FirstName, func(m *domain.TrackedUser) string { return m.FirstName })
	fillEmpty(&u.LastName, func(m *domain.TrackedUser) string { return m.LastName })
	fillEmpty(&u.Email, func(m *domain.TrackedUser) string { return m.Email })
	fillEmpty(&u.Phone, func(m *domain.TrackedUser) string { return m.Phone })
	fillEmpty(&u.ClientIP, func(m *domain.TrackedUser) string { return m.ClientIP })
	fillEmpty(&u.ClientUserAgent, func(m *domain.TrackedUser) string { return m.ClientUserAgent })
	fillEmpty(&u.Country, func(m *domain.TrackedUser) string { return m.Country })
	fillEmpty(&u.State, func(m *domain.TrackedUser) string { return m.State })
	fillEmpty(&u.City, func(m *domain.TrackedUser) string { return m.City })
	fillEmpty(&u.PostalCode, func(m *domain.TrackedUser) string { return m.PostalCode })
}

// enrichCRMLeadsGroupFields consolida em lote (1 query única) os click IDs,
// cookies e UTMs de todos os cards do Kanban que possuem grupo ou telefone/e-mail.
func enrichCRMLeadsGroupFields(ctx context.Context, pool *pgxpool.Pool, accountID int64, leads []domain.CRMLead) {
	if pool == nil || accountID <= 0 || len(leads) == 0 {
		return
	}

	groupIDs := make([]int64, 0, len(leads))
	phones := make([]string, 0, len(leads))
	emails := make([]string, 0, len(leads))
	seenG := map[int64]bool{}
	seenP := map[string]bool{}
	seenE := map[string]bool{}

	for i := range leads {
		tu := leads[i].TrackedUser
		if tu == nil {
			continue
		}
		if tu.GroupID != nil && *tu.GroupID > 0 && !seenG[*tu.GroupID] {
			seenG[*tu.GroupID] = true
			groupIDs = append(groupIDs, *tu.GroupID)
		}
		if tu.Phone != "" && !seenP[tu.Phone] {
			seenP[tu.Phone] = true
			phones = append(phones, tu.Phone)
		}
		if tu.Email != "" && !seenE[tu.Email] {
			seenE[tu.Email] = true
			emails = append(emails, tu.Email)
		}
	}
	if len(groupIDs) == 0 && len(phones) == 0 && len(emails) == 0 {
		return
	}

	rows, err := pool.Query(ctx, `
		SELECT `+userColumns+` FROM tracked_users
		WHERE account_id = $1
		  AND (
		    (group_id IS NOT NULL AND group_id = ANY($2::bigint[]))
		    OR (phone <> '' AND phone = ANY($3::text[]))
		    OR (email <> '' AND email = ANY($4::text[]))
		  )
		ORDER BY updated_at DESC, id DESC
	`, accountID, groupIDs, phones, emails)
	if err != nil {
		return
	}
	defer rows.Close()

	byGroup := map[int64][]*domain.TrackedUser{}
	byPhone := map[string][]*domain.TrackedUser{}
	byEmail := map[string][]*domain.TrackedUser{}
	for rows.Next() {
		m, err := scanUser(rows)
		if err != nil {
			return
		}
		if m.GroupID != nil && *m.GroupID > 0 {
			byGroup[*m.GroupID] = append(byGroup[*m.GroupID], m)
		}
		if m.Phone != "" {
			byPhone[m.Phone] = append(byPhone[m.Phone], m)
		}
		if m.Email != "" {
			byEmail[m.Email] = append(byEmail[m.Email], m)
		}
	}

	for i := range leads {
		tu := leads[i].TrackedUser
		if tu == nil {
			continue
		}
		var members []*domain.TrackedUser
		if tu.GroupID != nil && len(byGroup[*tu.GroupID]) > 1 {
			members = byGroup[*tu.GroupID]
		} else if tu.Phone != "" && len(byPhone[tu.Phone]) > 1 {
			members = byPhone[tu.Phone]
		} else if tu.Email != "" && len(byEmail[tu.Email]) > 1 {
			members = byEmail[tu.Email]
		}
		if len(members) > 1 {
			applyConsolidatedGroupMembers(tu, members)
		}
	}
}

