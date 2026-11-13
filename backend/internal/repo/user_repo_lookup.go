package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// FindByExternalID busca um tracked user pelo ID gerado no client (UUID-like de 10 chars).
func (r *UserRepo) FindByExternalID(ctx context.Context, externalID string) (*domain.TrackedUser, error) {
	return r.FindByExternalIDByAccount(ctx, externalID, 0)
}

// FindByExternalIDByAccount busca um tracked user por external_id isolando por account_id se accountID > 0.
func (r *UserRepo) FindByExternalIDByAccount(ctx context.Context, externalID string, accountID int64) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users 
	      WHERE external_id = $1 AND ($2 = 0 OR account_id = $2)
	      ORDER BY updated_at DESC LIMIT 1`
	return scanUser(r.pool.QueryRow(ctx, q, externalID, accountID))
}

// FindByExternalIDAndPixel busca um tracked user pelo external_id + pixel_name.
// Útil para APIs externas que enviam eventos com context de um pixel específico.
func (r *UserRepo) FindByExternalIDAndPixel(ctx context.Context, externalID, pixelName string) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users 
	      WHERE external_id = $1 AND pixel_name = $2 
	      ORDER BY updated_at DESC LIMIT 1`
	return scanUser(r.pool.QueryRow(ctx, q, externalID, pixelName))
}

// FindByPhone busca um tracked user pelo telefone exato (já normalizado).
// Retorna o registro mais recente se houver múltiplos.
func (r *UserRepo) FindByPhone(ctx context.Context, phone string) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users
	      WHERE phone = $1 AND phone <> ''
	      ORDER BY updated_at DESC LIMIT 1`
	return scanUser(r.pool.QueryRow(ctx, q, phone))
}

// FindByPhoneVariants busca por qualquer um dos telefones informados (ex: com e sem 55).
// Retorna o registro mais recente.
func (r *UserRepo) FindByPhoneVariants(ctx context.Context, phones []string) (*domain.TrackedUser, error) {
	return r.FindByPhoneVariantsByAccount(ctx, phones, 0)
}

// FindByPhoneVariantsByAccount busca por telefones com filtro opcional por account_id.
func (r *UserRepo) FindByPhoneVariantsByAccount(ctx context.Context, phones []string, accountID int64) (*domain.TrackedUser, error) {
	if len(phones) == 0 {
		return nil, ErrNotFound
	}
	q := `SELECT ` + userColumns + ` FROM tracked_users
	      WHERE phone = ANY($1::text[]) AND phone <> ''
	        AND ($2 = 0 OR account_id = $2)
	      ORDER BY updated_at DESC LIMIT 1`
	return scanUser(r.pool.QueryRow(ctx, q, phones, accountID))
}

// FindByEmail busca um tracked user pelo email exato (lowercase).
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.TrackedUser, error) {
	return r.FindByEmailByAccount(ctx, email, 0)
}

// FindByEmailByAccount busca um tracked user pelo email com filtro opcional por account_id.
func (r *UserRepo) FindByEmailByAccount(ctx context.Context, email string, accountID int64) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users
	      WHERE email = $1 AND email <> ''
	        AND ($2 = 0 OR account_id = $2)
	      ORDER BY updated_at DESC LIMIT 1`
	return scanUser(r.pool.QueryRow(ctx, q, email, accountID))
}

// FindByIDWpp busca um tracked user pelo id_wpp (10 chars [A-Z0-9]).
func (r *UserRepo) FindByIDWpp(ctx context.Context, idWpp string) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users WHERE id_wpp = $1 AND id_wpp <> ''`
	return scanUser(r.pool.QueryRow(ctx, q, idWpp))
}

// FindByID busca pelo PK numérico (não pelo external_id). Usado pelo
// painel admin pra carregar os dados antes de editar.
func (r *UserRepo) FindByID(ctx context.Context, id int64) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users WHERE id = $1`
	u, err := scanUser(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		return nil, err
	}

	_ = r.EnrichLTV(ctx, u)

	return u, nil
}

// EnrichLTV preenche LTV, TimeToPurchase e consolida cookies/click IDs/UTMs do grupo.
func (r *UserRepo) EnrichLTV(ctx context.Context, u *domain.TrackedUser) error {
	enrichTrackedUserGroupFields(ctx, r.pool, u)
	extIDs := r.groupExternalIDs(ctx, u)
	accID := u.AccountID

	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(max_value), 0.0)
		FROM (
			SELECT event_id, MAX(purchase_value) AS max_value
			FROM sent_events
			WHERE external_id = ANY($1::text[]) AND ($2 = 0 OR account_id = $2)
			  AND event_name = 'Purchase' AND status = 'sent'
			GROUP BY event_id
		) t
	`, extIDs, accID).Scan(&u.LTV)

	var earliestCreatedAt time.Time
	var groupFirstSource, groupFirstMedium, groupFirstCampaign, groupFirstContent, groupFirstTerm string
	_ = r.pool.QueryRow(ctx, `
		SELECT created_at, first_utm_source, first_utm_medium, first_utm_campaign, first_utm_content, first_utm_term
		FROM tracked_users
		WHERE external_id = ANY($1::text[]) AND ($2 = 0 OR account_id = $2)
		ORDER BY created_at ASC
		LIMIT 1
	`, extIDs, accID).Scan(&earliestCreatedAt, &groupFirstSource, &groupFirstMedium, &groupFirstCampaign, &groupFirstContent, &groupFirstTerm)

	if earliestCreatedAt.IsZero() {
		earliestCreatedAt = u.CreatedAt
	}

	if groupFirstSource == "" {
		_ = r.pool.QueryRow(ctx, `
			SELECT first_utm_source, first_utm_medium, first_utm_campaign, first_utm_content, first_utm_term
			FROM tracked_users
			WHERE external_id = ANY($1::text[]) AND ($2 = 0 OR account_id = $2) AND first_utm_source <> ''
			ORDER BY created_at ASC
			LIMIT 1
		`, extIDs, accID).Scan(&groupFirstSource, &groupFirstMedium, &groupFirstCampaign, &groupFirstContent, &groupFirstTerm)
	}

	if u.FirstUTMSource == "" && groupFirstSource != "" {
		u.FirstUTMSource = groupFirstSource
		u.FirstUTMMedium = groupFirstMedium
		u.FirstUTMCampaign = groupFirstCampaign
		u.FirstUTMContent = groupFirstContent
		u.FirstUTMTerm = groupFirstTerm
	}

	if u.LastUTMSource == "" {
		_ = r.pool.QueryRow(ctx, `
			SELECT last_utm_source, last_utm_medium, last_utm_campaign, last_utm_content, last_utm_term
			FROM tracked_users
			WHERE external_id = ANY($1::text[]) AND ($2 = 0 OR account_id = $2) AND last_utm_source <> ''
			ORDER BY updated_at DESC
			LIMIT 1
		`, extIDs, accID).Scan(&u.LastUTMSource, &u.LastUTMMedium, &u.LastUTMCampaign, &u.LastUTMContent, &u.LastUTMTerm)
	}

	if u.FirstURL == "" {
		_ = r.pool.QueryRow(ctx, `
			SELECT first_url
			FROM tracked_users
			WHERE external_id = ANY($1::text[]) AND ($2 = 0 OR account_id = $2) AND first_url <> ''
			ORDER BY created_at ASC
			LIMIT 1
		`, extIDs, accID).Scan(&u.FirstURL)
	}
	if u.FirstURL == "" && u.URL != "" {
		u.FirstURL = u.URL
	}

	var firstPurchaseTime time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT MIN(created_at)
		FROM sent_events
		WHERE external_id = ANY($1::text[]) AND ($2 = 0 OR account_id = $2)
		  AND event_name = 'Purchase' AND status = 'sent'
	`, extIDs, accID).Scan(&firstPurchaseTime)
	if err == nil && !firstPurchaseTime.IsZero() {
		diff := firstPurchaseTime.Sub(earliestCreatedAt)
		if diff < 0 {
			diff = 0
		}
		u.TimeToPurchase = formatDuration(diff)
	} else {
		u.TimeToPurchase = "Nenhuma compra"
	}
	return nil
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	var parts []string
	if days > 0 {
		if days == 1 {
			parts = append(parts, "1 dia")
		} else {
			parts = append(parts, fmt.Sprintf("%d dias", days))
		}
	}
	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 hora")
		} else {
			parts = append(parts, fmt.Sprintf("%d horas", hours))
		}
	}
	if minutes > 0 {
		if minutes == 1 {
			parts = append(parts, "1 minuto")
		} else {
			parts = append(parts, fmt.Sprintf("%d minutos", minutes))
		}
	}
	if len(parts) == 0 {
		return "menos de 1 minuto"
	}
	return strings.Join(parts, ", ")
}

// SoleAccountID retorna o ID da conta quando a instalação possui exatamente uma conta.
// Usado por webhooks legados sem ?acc= / ?pixel= para evitar busca cross-account.
func (r *UserRepo) SoleAccountID(ctx context.Context) (int64, bool) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM accounts ORDER BY id LIMIT 2`)
	if err != nil {
		return 0, false
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) != 1 {
		return 0, false
	}
	return ids[0], true
}
