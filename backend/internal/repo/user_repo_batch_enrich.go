package repo

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func buildTrackedUserListWhere(f TrackedUserFilter) (string, []any, int) {
	args := make([]any, 0, 6)
	conds := make([]string, 0, 4)
	idx := 0
	addArg := func(value any) string {
		args = append(args, value)
		idx++
		return "$" + itoa(idx)
	}

	if f.AccountID != 0 {
		conds = append(conds, "account_id = "+addArg(f.AccountID))
	}
	if px := strings.TrimSpace(f.PixelName); px != "" {
		conds = append(conds, "pixel_name = "+addArg(px))
	}

	switch strings.ToLower(strings.TrimSpace(f.Segment)) {
	case "identified":
		conds = append(conds, `(
			email <> '' OR phone <> '' OR first_name <> '' OR last_name <> ''
			OR (group_id IS NOT NULL AND EXISTS (
				SELECT 1 FROM tracked_users t2
				WHERE t2.account_id = tracked_users.account_id
				  AND t2.group_id = tracked_users.group_id
				  AND (t2.email <> '' OR t2.phone <> '' OR t2.first_name <> '' OR t2.last_name <> '')
			))
		)`)
	case "buyers":
		conds = append(conds, `(
			EXISTS (
				SELECT 1 FROM sent_events se
				WHERE se.external_id = tracked_users.external_id
				  AND (tracked_users.account_id = 0 OR se.account_id = tracked_users.account_id)
				  AND se.event_name = 'Purchase' AND se.status = 'sent'
			)
			OR (group_id IS NOT NULL AND EXISTS (
				SELECT 1 FROM tracked_users t2
				JOIN sent_events se ON se.external_id = t2.external_id AND se.account_id = t2.account_id
				WHERE t2.account_id = tracked_users.account_id
				  AND COALESCE(t2.group_id, -t2.id) = COALESCE(tracked_users.group_id, -tracked_users.id)
				  AND se.event_name = 'Purchase' AND se.status = 'sent'
			))
		)`)
	case "anonymous":
		conds = append(conds, `(
			email = '' AND phone = '' AND first_name = '' AND last_name = ''
			AND (group_id IS NULL OR NOT EXISTS (
				SELECT 1 FROM tracked_users t2
				WHERE t2.account_id = tracked_users.account_id
				  AND t2.group_id = tracked_users.group_id
				  AND (t2.email <> '' OR t2.phone <> '' OR t2.first_name <> '' OR t2.last_name <> '')
			))
		)`)
	case "bot":
		conds = append(conds, "traffic_type = 'bot'")
	}

	if search := strings.TrimSpace(f.Search); search != "" {
		ph := addArg("%" + search + "%")
		ors := []string{
			"external_id ILIKE " + ph,
			"(id_wpp <> '' AND id_wpp ILIKE " + ph + ")",
			"(email <> '' AND email ILIKE " + ph + ")",
			"(phone <> '' AND phone ILIKE " + ph + ")",
			"(fbp <> '' AND fbp ILIKE " + ph + ")",
			"(first_name <> '' AND first_name ILIKE " + ph + ")",
			"(last_name <> '' AND last_name ILIKE " + ph + ")",
			"((first_name <> '' OR last_name <> '') AND (first_name || ' ' || last_name) ILIKE " + ph + ")",
		}
		digits := extractDigits(search)
		if len(digits) >= 4 && digits != search {
			phDigits := addArg("%" + digits + "%")
			ors = append(ors, "(phone <> '' AND phone ILIKE "+phDigits+")")
		}
		if len(digits) >= 8 {
			variants := util.PhoneVariants(util.NormalizePhone(search))
			if len(variants) > 0 {
				phVar := addArg(variants)
				ors = append(ors, "(phone <> '' AND phone = ANY("+phVar+"::text[]))")
			}
		}
		conds = append(conds, "("+strings.Join(ors, " OR ")+")")
	}

	if len(conds) == 0 {
		return "", args, idx
	}
	return "WHERE " + strings.Join(conds, " AND "), args, idx
}

func extractDigits(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// enrichTrackedUsersSlice consolida em uma única query os dados de grupo
// (nome, email, telefone, geo, UTMs e click IDs) para a página de usuários.
func enrichTrackedUsersSlice(ctx context.Context, pool *pgxpool.Pool, users []domain.TrackedUser) {
	if pool == nil || len(users) == 0 {
		return
	}

	accountIDs := make([]int64, 0, len(users))
	groupIDs := make([]int64, 0, len(users))
	phones := make([]string, 0, len(users))
	emails := make([]string, 0, len(users))
	seenA := map[int64]bool{}
	seenG := map[int64]bool{}
	seenP := map[string]bool{}
	seenE := map[string]bool{}

	for i := range users {
		tu := &users[i]
		if tu.AccountID > 0 && !seenA[tu.AccountID] {
			seenA[tu.AccountID] = true
			accountIDs = append(accountIDs, tu.AccountID)
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
	if len(accountIDs) == 0 || (len(groupIDs) == 0 && len(phones) == 0 && len(emails) == 0) {
		return
	}

	rows, err := pool.Query(ctx, `
		SELECT `+userColumns+` FROM tracked_users
		WHERE account_id = ANY($1::bigint[])
		  AND (
		    (group_id IS NOT NULL AND group_id = ANY($2::bigint[]))
		    OR (phone <> '' AND phone = ANY($3::text[]))
		    OR (email <> '' AND email = ANY($4::text[]))
		  )
		ORDER BY updated_at DESC, id DESC
	`, accountIDs, groupIDs, phones, emails)
	if err != nil {
		return
	}
	defer rows.Close()

	type accGroup struct {
		accID   int64
		groupID int64
	}
	type accStr struct {
		accID int64
		val   string
	}
	byGroup := map[accGroup][]*domain.TrackedUser{}
	byPhone := map[accStr][]*domain.TrackedUser{}
	byEmail := map[accStr][]*domain.TrackedUser{}

	for rows.Next() {
		m, err := scanUser(rows)
		if err != nil {
			return
		}
		if m.GroupID != nil && *m.GroupID > 0 {
			k := accGroup{m.AccountID, *m.GroupID}
			byGroup[k] = append(byGroup[k], m)
		}
		if m.Phone != "" {
			k := accStr{m.AccountID, m.Phone}
			byPhone[k] = append(byPhone[k], m)
		}
		if m.Email != "" {
			k := accStr{m.AccountID, m.Email}
			byEmail[k] = append(byEmail[k], m)
		}
	}

	for i := range users {
		tu := &users[i]
		var members []*domain.TrackedUser
		if tu.GroupID != nil && len(byGroup[accGroup{tu.AccountID, *tu.GroupID}]) > 1 {
			members = byGroup[accGroup{tu.AccountID, *tu.GroupID}]
		} else if tu.Phone != "" && len(byPhone[accStr{tu.AccountID, tu.Phone}]) > 1 {
			members = byPhone[accStr{tu.AccountID, tu.Phone}]
		} else if tu.Email != "" && len(byEmail[accStr{tu.AccountID, tu.Email}]) > 1 {
			members = byEmail[accStr{tu.AccountID, tu.Email}]
		}
		if len(members) > 1 {
			applyConsolidatedGroupMembers(tu, members)
		}
		if tu.FirstURL == "" && tu.URL != "" {
			tu.FirstURL = tu.URL
		}
	}
}
