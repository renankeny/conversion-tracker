package repo

import (
	"context"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// clearableTrackedUserColumns define a whitelist estrita de colunas que podem
// ser limpas em sessões irmãs quando um administrador apaga um valor no painel.
var clearableTrackedUserColumns = map[string]bool{
	"id_wpp":            true,
	"fbp":               true,
	"fbc":               true,
	"ctwa_clid":         true,
	"gclid":             true,
	"wbraid":            true,
	"gbraid":            true,
	"ttclid":            true,
	"ttp":               true,
	"ga4_client_id":     true,
	"ga4_session_id":    true,
	"client_ip":         true,
	"client_user_agent": true,
	"country":           true,
	"state":             true,
	"city":              true,
	"postal_code":       true,
	"first_name":        true,
	"last_name":         true,
	"email":             true,
	"phone":             true,
	"url":               true,
	"first_url":         true,
	"first_utm_source":   true,
	"first_utm_medium":   true,
	"first_utm_campaign": true,
	"first_utm_content":  true,
	"first_utm_term":     true,
	"last_utm_source":    true,
	"last_utm_medium":    true,
	"last_utm_campaign":  true,
	"last_utm_content":   true,
	"last_utm_term":      true,
}

// FindRawByID carrega apenas a linha individual de tracked_users pelo ID (PK),
// sem consolidar campos de grupo nem calcular LTV. Ideal para checagem de RBAC
// multi-tenant e carregamento prévio à edição/exclusão.
func (r *UserRepo) FindRawByID(ctx context.Context, id int64) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users WHERE id = $1`
	return scanUser(r.pool.QueryRow(ctx, q, id))
}

// ClearGroupColumns limpa (define como string vazia) as colunas informadas em
// todas as sessões irmãs do mesmo grupo ou contato dentro da mesma conta,
// evitando que FindByID ressuscite dados apagados pelo administrador.
func (r *UserRepo) ClearGroupColumns(ctx context.Context, accountID int64, groupID *int64, phone, email string, cols []string) error {
	if accountID <= 0 || len(cols) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(cols))
	setClauses := make([]string, 0, len(cols)+1)
	for _, col := range cols {
		if clearableTrackedUserColumns[col] && !seen[col] {
			seen[col] = true
			setClauses = append(setClauses, col+" = ''")
		}
	}
	if len(setClauses) == 0 {
		return nil
	}
	setClauses = append(setClauses, "updated_at = NOW()")

	args := []any{accountID}
	matchOrs := make([]string, 0, 3)
	if groupID != nil && *groupID > 0 {
		args = append(args, *groupID)
		matchOrs = append(matchOrs, "group_id = $"+itoa(len(args)))
	}
	if phone != "" {
		args = append(args, phone)
		matchOrs = append(matchOrs, "phone = $"+itoa(len(args)))
	}
	if email != "" {
		args = append(args, email)
		matchOrs = append(matchOrs, "email = $"+itoa(len(args)))
	}
	if len(matchOrs) == 0 {
		return nil
	}

	q := "UPDATE tracked_users SET " + strings.Join(setClauses, ", ") +
		" WHERE account_id = $1 AND (" + strings.Join(matchOrs, " OR ") + ")"
	_, err := r.pool.Exec(ctx, q, args...)
	return err
}

// DeleteByID exclui o tracked_user (e sessões irmãs do mesmo group_id na mesma conta,
// quando aplicável) garantindo escopo multi-tenant.
func (r *UserRepo) DeleteByID(ctx context.Context, accountID int64, id int64, groupID *int64) (int64, error) {
	if groupID != nil && *groupID > 0 {
		tag, err := r.pool.Exec(ctx, `
			DELETE FROM tracked_users
			WHERE (id = $1 OR (group_id = $3 AND ($4::bigint = 0 OR account_id = $4)))
			  AND ($2::bigint = 0 OR account_id = $2)
		`, id, accountID, *groupID, accountID)
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	}
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM tracked_users
		WHERE id = $1 AND ($2::bigint = 0 OR account_id = $2)
	`, id, accountID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
