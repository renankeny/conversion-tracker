package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

const userColumns = `id, external_id, id_wpp, pixel_name, fbp, fbc, gclid,
		client_ip, client_user_agent,
		country, state, city, postal_code,
		first_name, last_name, email, phone, url,
		first_utm_source, first_utm_medium, first_utm_campaign, first_utm_content, first_utm_term,
		last_utm_source, last_utm_medium, last_utm_campaign, last_utm_content, last_utm_term,
		created_at, updated_at`

// FindByExternalID busca um tracked user pelo ID gerado no client (UUID-like de 10 chars).
func (r *UserRepo) FindByExternalID(ctx context.Context, externalID string) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users WHERE external_id = $1`
	return scanUser(r.pool.QueryRow(ctx, q, externalID))
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
	if len(phones) == 0 {
		return nil, ErrNotFound
	}
	q := `SELECT ` + userColumns + ` FROM tracked_users
	      WHERE phone = ANY($1::text[]) AND phone <> ''
	      ORDER BY updated_at DESC LIMIT 1`
	return scanUser(r.pool.QueryRow(ctx, q, phones))
}

// FindByEmail busca um tracked user pelo email exato (lowercase).
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users
	      WHERE email = $1 AND email <> ''
	      ORDER BY updated_at DESC LIMIT 1`
	return scanUser(r.pool.QueryRow(ctx, q, email))
}

// FindByIDWpp busca um tracked user pelo id_wpp (10 chars [A-Z0-9]).
func (r *UserRepo) FindByIDWpp(ctx context.Context, idWpp string) (*domain.TrackedUser, error) {
	q := `SELECT ` + userColumns + ` FROM tracked_users WHERE id_wpp = $1 AND id_wpp <> ''`
	return scanUser(r.pool.QueryRow(ctx, q, idWpp))
}

// Upsert cria ou atualiza pelo external_id.
//
// Os campos UTM "first_*" só são preenchidos na criação (não são sobrescritos depois).
// Os campos PII e UTMs "last_*" são atualizados apenas quando o valor de entrada
// for não vazio (evita sobrescrever dado bom com vazio).
//
// id_wpp é gravado na criação; se já existir um id_wpp não vazio na linha, ele é
// preservado e não sobrescrito.
func (r *UserRepo) Upsert(ctx context.Context, u *domain.TrackedUser) error {
	const q = `
	INSERT INTO tracked_users (
		external_id, id_wpp, pixel_name,
		fbp, fbc, gclid, client_ip, client_user_agent,
		country, state, city, postal_code,
		first_name, last_name, email, phone, url,
		first_utm_source, first_utm_medium, first_utm_campaign, first_utm_content, first_utm_term,
		last_utm_source,  last_utm_medium,  last_utm_campaign,  last_utm_content,  last_utm_term
	) VALUES (
		$1,$2,$3,
		$4,$5,$6,$7,$8,
		$9,$10,$11,$12,
		$13,$14,$15,$16,$17,
		$18,$19,$20,$21,$22,
		$18,$19,$20,$21,$22
	)
	ON CONFLICT (external_id) DO UPDATE SET
		id_wpp            = CASE WHEN tracked_users.id_wpp = '' THEN EXCLUDED.id_wpp ELSE tracked_users.id_wpp END,
		pixel_name        = COALESCE(NULLIF(EXCLUDED.pixel_name, ''), tracked_users.pixel_name),
		fbp               = COALESCE(NULLIF(EXCLUDED.fbp,  ''), tracked_users.fbp),
		fbc               = COALESCE(NULLIF(EXCLUDED.fbc,  ''), tracked_users.fbc),
		gclid             = COALESCE(NULLIF(EXCLUDED.gclid,''), tracked_users.gclid),
		client_ip         = COALESCE(NULLIF(EXCLUDED.client_ip, ''), tracked_users.client_ip),
		client_user_agent = COALESCE(NULLIF(EXCLUDED.client_user_agent, ''), tracked_users.client_user_agent),
		country           = COALESCE(NULLIF(EXCLUDED.country, ''), tracked_users.country),
		state             = COALESCE(NULLIF(EXCLUDED.state, ''), tracked_users.state),
		city              = COALESCE(NULLIF(EXCLUDED.city, ''), tracked_users.city),
		postal_code       = COALESCE(NULLIF(EXCLUDED.postal_code, ''), tracked_users.postal_code),
		first_name        = COALESCE(NULLIF(EXCLUDED.first_name, ''), tracked_users.first_name),
		last_name         = COALESCE(NULLIF(EXCLUDED.last_name, ''), tracked_users.last_name),
		email             = COALESCE(NULLIF(EXCLUDED.email, ''), tracked_users.email),
		phone             = COALESCE(NULLIF(EXCLUDED.phone, ''), tracked_users.phone),
		url               = COALESCE(NULLIF(EXCLUDED.url, ''), tracked_users.url),
		last_utm_source   = COALESCE(NULLIF(EXCLUDED.last_utm_source, ''),   tracked_users.last_utm_source),
		last_utm_medium   = COALESCE(NULLIF(EXCLUDED.last_utm_medium, ''),   tracked_users.last_utm_medium),
		last_utm_campaign = COALESCE(NULLIF(EXCLUDED.last_utm_campaign, ''), tracked_users.last_utm_campaign),
		last_utm_content  = COALESCE(NULLIF(EXCLUDED.last_utm_content, ''),  tracked_users.last_utm_content),
		last_utm_term     = COALESCE(NULLIF(EXCLUDED.last_utm_term, ''),     tracked_users.last_utm_term),
		updated_at        = NOW()
	`
	_, err := r.pool.Exec(ctx, q,
		u.ExternalID, u.IDWpp, u.PixelName,
		u.FBP, u.FBC, u.GCLID, u.ClientIP, u.ClientUserAgent,
		u.Country, u.State, u.City, u.PostalCode,
		u.FirstName, u.LastName, u.Email, u.Phone, u.URL,
		u.FirstUTMSource, u.FirstUTMMedium, u.FirstUTMCampaign, u.FirstUTMContent, u.FirstUTMTerm,
	)
	return err
}

func scanUser(row pgx.Row) (*domain.TrackedUser, error) {
	var u domain.TrackedUser
	err := row.Scan(
		&u.ID, &u.ExternalID, &u.IDWpp, &u.PixelName, &u.FBP, &u.FBC, &u.GCLID,
		&u.ClientIP, &u.ClientUserAgent,
		&u.Country, &u.State, &u.City, &u.PostalCode,
		&u.FirstName, &u.LastName, &u.Email, &u.Phone, &u.URL,
		&u.FirstUTMSource, &u.FirstUTMMedium, &u.FirstUTMCampaign, &u.FirstUTMContent, &u.FirstUTMTerm,
		&u.LastUTMSource, &u.LastUTMMedium, &u.LastUTMCampaign, &u.LastUTMContent, &u.LastUTMTerm,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}


// TrackedUserFilter agrupa os filtros aceitos por List.
type TrackedUserFilter struct {
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
	conds := make([]string, 0, 2)
	idx := 0
	addArg := func(value any) string {
		args = append(args, value)
		idx++
		return "$" + itoa(idx)
	}
	if f.PixelName != "" {
		conds = append(conds, "pixel_name = "+addArg(f.PixelName))
	}
	if f.Search != "" {
		// busca insensível a caso em vários campos
		ph := "%" + f.Search + "%"
		ph2 := addArg(ph)
		conds = append(conds, "(external_id ILIKE "+ph2+" OR id_wpp ILIKE "+ph2+
			" OR email ILIKE "+ph2+" OR phone ILIKE "+ph2+")")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}

	var total int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM tracked_users "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	q := "SELECT " + userColumns + " FROM tracked_users " + where +
		" ORDER BY id DESC LIMIT $" + itoa(idx+1) + " OFFSET $" + itoa(idx+2)

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
