package repo

import (
	"context"
	"encoding/json"
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

const userColumns = `id, external_id, id_wpp, pixel_name, account_id, group_id,
		fbp, fbc, gclid, wbraid, gbraid, ttclid, ttp,
		client_ip, client_user_agent,
		country, state, city, postal_code,
		first_name, last_name, email, phone, url, first_url,
		first_utm_source, first_utm_medium, first_utm_campaign, first_utm_content, first_utm_term,
		last_utm_source, last_utm_medium, last_utm_campaign, last_utm_content, last_utm_term,
		ctwa_clid, wa_chat_id, wa_name, wa_ad_id, wa_source_app,
		ga4_client_id, ga4_session_id, wa_lid,
		created_at, updated_at,
		pageview_count, checkout_count, last_seen_at,
		referrer, traffic_type, traffic_score, traffic_category, traffic_rules`

// Create insere um novo tracked_user. Retorna erro se external_id já existir.
func (r *UserRepo) Create(ctx context.Context, u *domain.TrackedUser) error {
	if u.AccountID == 0 {
		u.AccountID = 1
	}
	const q = `INSERT INTO tracked_users (
		external_id, id_wpp, pixel_name, account_id,
		fbp, fbc, gclid, wbraid, gbraid, ttclid, ttp,
		client_ip, client_user_agent,
		country, state, city, postal_code,
		first_name, last_name, email, phone, url, first_url,
		first_utm_source, first_utm_medium, first_utm_campaign, first_utm_content, first_utm_term,
		last_utm_source, last_utm_medium, last_utm_campaign, last_utm_content, last_utm_term,
		ctwa_clid, wa_chat_id, wa_name, wa_ad_id, wa_source_app,
		ga4_client_id, ga4_session_id, wa_lid,
		pageview_count, checkout_count, last_seen_at,
		referrer, traffic_type, traffic_score, traffic_category, traffic_rules
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
		$21,$48,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,
		$41,$42,NOW(),
		$43,$44,$45,$46,$47
	) RETURNING id, created_at, updated_at`
	pageviewCount := u.PageViewCount
	if pageviewCount <= 0 {
		pageviewCount = 1
	}
	trafficType := u.TrafficType
	if trafficType == "" {
		trafficType = "human"
	}
	trafficScore := u.TrafficScore
	if trafficScore == 0 {
		trafficScore = 60
	}
	trafficCategory := u.TrafficCategory
	if trafficCategory == "" {
		trafficCategory = "none"
	}
	trafficRules := u.TrafficRules
	if len(trafficRules) == 0 {
		trafficRules = json.RawMessage("[]")
	}
	firstURL := u.FirstURL
	if firstURL == "" {
		firstURL = u.URL
	}
	return r.pool.QueryRow(ctx, q,
		u.ExternalID, u.IDWpp, u.PixelName, u.AccountID,
		u.FBP, u.FBC, u.GCLID, u.WBraid, u.GBraid, u.TTClid, u.TTP,
		u.ClientIP, u.ClientUserAgent,
		u.Country, u.State, u.City, u.PostalCode,
		u.FirstName, u.LastName, u.Email, u.Phone, u.URL,
		u.FirstUTMSource, u.FirstUTMMedium, u.FirstUTMCampaign, u.FirstUTMContent, u.FirstUTMTerm,
		u.LastUTMSource, u.LastUTMMedium, u.LastUTMCampaign, u.LastUTMContent, u.LastUTMTerm,
		u.CtwaClid, u.WaChatID, u.WaName, u.WaAdID, u.WaSourceApp,
		u.GA4ClientID, u.GA4SessionID, u.WaLid,
		pageviewCount, u.CheckoutCount,
		u.Referrer, trafficType, trafficScore, trafficCategory, trafficRules,
		firstURL,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

// Update atualiza um tracked_user existente pelo ID.
func (r *UserRepo) Update(ctx context.Context, u *domain.TrackedUser) error {
	const q = `
	UPDATE tracked_users SET
		external_id = $2,
		id_wpp = $3,
		pixel_name = $4,
		fbp = $5, fbc = $6, gclid = $7,
		wbraid = $8, gbraid = $9,
		ttclid = $10, ttp = $11,
		client_ip = $12, client_user_agent = $13,
		country = $14, state = $15, city = $16, postal_code = $17,
		first_name = $18, last_name = $19, email = $20, phone = $21,
		url = $22,
		first_url = CASE WHEN tracked_users.first_url = '' THEN $22 ELSE tracked_users.first_url END,
		first_utm_source = $23, first_utm_medium = $24,
		first_utm_campaign = $25, first_utm_content = $26, first_utm_term = $27,
		last_utm_source = $28, last_utm_medium = $29,
		last_utm_campaign = $30, last_utm_content = $31, last_utm_term = $32,
		ctwa_clid = $33, wa_chat_id = $34, wa_name = $35,
		wa_ad_id = $36, wa_source_app = $37,
		ga4_client_id = $38, ga4_session_id = $39,
		updated_at = NOW()
	WHERE id = $1
	RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, q,
		u.ID,
		u.ExternalID, u.IDWpp, u.PixelName,
		u.FBP, u.FBC, u.GCLID,
		u.WBraid, u.GBraid,
		u.TTClid, u.TTP,
		u.ClientIP, u.ClientUserAgent,
		u.Country, u.State, u.City, u.PostalCode,
		u.FirstName, u.LastName, u.Email, u.Phone,
		u.URL,
		u.FirstUTMSource, u.FirstUTMMedium,
		u.FirstUTMCampaign, u.FirstUTMContent, u.FirstUTMTerm,
		u.LastUTMSource, u.LastUTMMedium,
		u.LastUTMCampaign, u.LastUTMContent, u.LastUTMTerm,
		u.CtwaClid, u.WaChatID, u.WaName,
		u.WaAdID, u.WaSourceApp,
		u.GA4ClientID, u.GA4SessionID,
	).Scan(&u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// UpdateByID atualiza um tracked_user por ID (PK), permitindo editar
// qualquer campo editável a partir do painel admin. Diferente do Upsert,
// não usa external_id como chave de conflito e não preserva valores —
// sobrescreve com o que vier no payload (admin assume responsabilidade).
//
// Campos não-editáveis (preservados): id, account_id (uso por owner via
// outra rota), created_at; updated_at é sempre atualizado pra NOW().
func (r *UserRepo) UpdateByID(ctx context.Context, u *domain.TrackedUser) error {
	const q = `
	UPDATE tracked_users SET
		external_id = $2,
		id_wpp = $3,
		pixel_name = $4,
		fbp = $5, fbc = $6, gclid = $7,
		wbraid = $8, gbraid = $9,
		ttclid = $10, ttp = $11,
		client_ip = $12, client_user_agent = $13,
		country = $14, state = $15, city = $16, postal_code = $17,
		first_name = $18, last_name = $19, email = $20, phone = $21,
		url = $22,
		first_url = CASE WHEN tracked_users.first_url = '' THEN $22 ELSE tracked_users.first_url END,
		first_utm_source = $23, first_utm_medium = $24,
		first_utm_campaign = $25, first_utm_content = $26, first_utm_term = $27,
		last_utm_source = $28, last_utm_medium = $29,
		last_utm_campaign = $30, last_utm_content = $31, last_utm_term = $32,
		updated_at = NOW()
	WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, q,
		u.ID,
		u.ExternalID, u.IDWpp, u.PixelName,
		u.FBP, u.FBC, u.GCLID,
		u.WBraid, u.GBraid,
		u.TTClid, u.TTP,
		u.ClientIP, u.ClientUserAgent,
		u.Country, u.State, u.City, u.PostalCode,
		u.FirstName, u.LastName, u.Email, u.Phone,
		u.URL,
		u.FirstUTMSource, u.FirstUTMMedium,
		u.FirstUTMCampaign, u.FirstUTMContent, u.FirstUTMTerm,
		u.LastUTMSource, u.LastUTMMedium,
		u.LastUTMCampaign, u.LastUTMContent, u.LastUTMTerm,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
