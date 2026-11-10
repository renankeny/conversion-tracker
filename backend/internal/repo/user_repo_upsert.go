package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/traffic"
)

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
		external_id, id_wpp, pixel_name, account_id,
		fbp, fbc, gclid, wbraid, gbraid, ttclid, ttp,
		client_ip, client_user_agent,
		country, state, city, postal_code,
		first_name, last_name, email, phone, url, first_url,
		first_utm_source, first_utm_medium, first_utm_campaign, first_utm_content, first_utm_term,
		last_utm_source,  last_utm_medium,  last_utm_campaign,  last_utm_content,  last_utm_term,
		ctwa_clid, wa_chat_id, wa_name, wa_ad_id, wa_source_app, wa_lid,
		ga4_client_id, ga4_session_id,
		pageview_count, checkout_count, last_seen_at,
		referrer, traffic_type, traffic_score, traffic_category, traffic_rules
	) VALUES (
		$1,$2,$3,$27,
		$4,$5,$6,$7,$8,$9,$10,
		$11,$12,
		$13,$14,$15,$16,
		$17,$18,$19,$20,$21,$43,
		$22,$23,$24,$25,$26,
		$22,$23,$24,$25,$26,
		$28,$29,$30,$31,$32,$33,
		$34,$35,
		$36,$37,NOW(),
		$38,$39,$40,$41,$42
	)
	ON CONFLICT (account_id, external_id) DO UPDATE SET
		id_wpp            = CASE WHEN tracked_users.id_wpp = '' THEN EXCLUDED.id_wpp ELSE tracked_users.id_wpp END,
		pixel_name        = COALESCE(NULLIF(EXCLUDED.pixel_name, ''), tracked_users.pixel_name),
		account_id        = CASE WHEN EXCLUDED.account_id > 0 THEN EXCLUDED.account_id ELSE tracked_users.account_id END,
		fbp               = COALESCE(NULLIF(EXCLUDED.fbp,    ''), tracked_users.fbp),
		fbc               = COALESCE(NULLIF(EXCLUDED.fbc,    ''), tracked_users.fbc),
		gclid             = COALESCE(NULLIF(EXCLUDED.gclid,  ''), tracked_users.gclid),
		wbraid            = COALESCE(NULLIF(EXCLUDED.wbraid, ''), tracked_users.wbraid),
		gbraid            = COALESCE(NULLIF(EXCLUDED.gbraid, ''), tracked_users.gbraid),
		ttclid            = COALESCE(NULLIF(EXCLUDED.ttclid, ''), tracked_users.ttclid),
		ttp               = COALESCE(NULLIF(EXCLUDED.ttp,    ''), tracked_users.ttp),
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
		first_url         = CASE WHEN tracked_users.first_url = '' THEN EXCLUDED.first_url ELSE tracked_users.first_url END,
		first_utm_source   = CASE WHEN tracked_users.first_utm_source = '' THEN EXCLUDED.first_utm_source ELSE tracked_users.first_utm_source END,
		first_utm_medium   = CASE WHEN tracked_users.first_utm_medium = '' THEN EXCLUDED.first_utm_medium ELSE tracked_users.first_utm_medium END,
		first_utm_campaign = CASE WHEN tracked_users.first_utm_campaign = '' THEN EXCLUDED.first_utm_campaign ELSE tracked_users.first_utm_campaign END,
		first_utm_content  = CASE WHEN tracked_users.first_utm_content = '' THEN EXCLUDED.first_utm_content ELSE tracked_users.first_utm_content END,
		first_utm_term     = CASE WHEN tracked_users.first_utm_term = '' THEN EXCLUDED.first_utm_term ELSE tracked_users.first_utm_term END,
		last_utm_source   = COALESCE(NULLIF(EXCLUDED.last_utm_source, ''),   tracked_users.last_utm_source),
		last_utm_medium   = COALESCE(NULLIF(EXCLUDED.last_utm_medium, ''),   tracked_users.last_utm_medium),
		last_utm_campaign = COALESCE(NULLIF(EXCLUDED.last_utm_campaign, ''), tracked_users.last_utm_campaign),
		last_utm_content  = COALESCE(NULLIF(EXCLUDED.last_utm_content, ''),  tracked_users.last_utm_content),
		last_utm_term     = COALESCE(NULLIF(EXCLUDED.last_utm_term, ''),     tracked_users.last_utm_term),
		ctwa_clid         = COALESCE(NULLIF(EXCLUDED.ctwa_clid, ''),     tracked_users.ctwa_clid),
		wa_chat_id        = COALESCE(NULLIF(EXCLUDED.wa_chat_id, ''),    tracked_users.wa_chat_id),
		wa_name           = COALESCE(NULLIF(EXCLUDED.wa_name, ''),      tracked_users.wa_name),
		wa_ad_id          = COALESCE(NULLIF(EXCLUDED.wa_ad_id, ''),     tracked_users.wa_ad_id),
		wa_source_app     = COALESCE(NULLIF(EXCLUDED.wa_source_app, ''), tracked_users.wa_source_app),
		wa_lid            = COALESCE(NULLIF(EXCLUDED.wa_lid, ''),       tracked_users.wa_lid),
		ga4_client_id     = COALESCE(NULLIF(EXCLUDED.ga4_client_id, ''),  tracked_users.ga4_client_id),
		ga4_session_id    = COALESCE(NULLIF(EXCLUDED.ga4_session_id, ''), tracked_users.ga4_session_id),
		pageview_count    = CASE WHEN EXCLUDED.pageview_count > 0 THEN tracked_users.pageview_count + EXCLUDED.pageview_count ELSE tracked_users.pageview_count END,
		checkout_count    = CASE WHEN EXCLUDED.checkout_count > 0 THEN tracked_users.checkout_count + EXCLUDED.checkout_count ELSE tracked_users.checkout_count END,
		referrer          = CASE WHEN tracked_users.referrer = '' THEN EXCLUDED.referrer ELSE tracked_users.referrer END,
		traffic_type      = CASE WHEN tracked_users.traffic_type = 'bot' THEN 'bot' ELSE COALESCE(NULLIF(EXCLUDED.traffic_type, ''), tracked_users.traffic_type) END,
		traffic_score     = CASE WHEN EXCLUDED.traffic_score != 0 THEN EXCLUDED.traffic_score ELSE tracked_users.traffic_score END,
		traffic_category  = COALESCE(NULLIF(EXCLUDED.traffic_category, ''), tracked_users.traffic_category),
		traffic_rules     = CASE WHEN EXCLUDED.traffic_rules IS NOT NULL AND jsonb_array_length(EXCLUDED.traffic_rules) > 0 THEN EXCLUDED.traffic_rules ELSE tracked_users.traffic_rules END,
		last_seen_at      = NOW(),
		updated_at        = NOW()
	RETURNING id
	`
	accountID := u.AccountID
	if accountID == 0 {
		accountID = 1
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
		u.ExternalID, u.IDWpp, u.PixelName,
		u.FBP, u.FBC, u.GCLID, u.WBraid, u.GBraid, u.TTClid, u.TTP,
		u.ClientIP, u.ClientUserAgent,
		u.Country, u.State, u.City, u.PostalCode,
		u.FirstName, u.LastName, u.Email, u.Phone, u.URL,
		u.FirstUTMSource, u.FirstUTMMedium, u.FirstUTMCampaign, u.FirstUTMContent, u.FirstUTMTerm,
		accountID,
		u.CtwaClid, u.WaChatID, u.WaName, u.WaAdID, u.WaSourceApp, u.WaLid,
		u.GA4ClientID, u.GA4SessionID,
		u.PageViewCount, u.CheckoutCount,
		u.Referrer, trafficType, trafficScore, trafficCategory, trafficRules,
		firstURL,
	).Scan(&u.ID)
}

// LinkedFields devolve, campo por campo, o valor mais recente não-vazio
// entre TODOS os tracked_users do mesmo GroupID do external_id informado.
func (r *UserRepo) LinkedFields(ctx context.Context, externalID string) (domain.LinkedTrackingFields, error) {
	return r.LinkedFieldsByAccount(ctx, 0, externalID)
}

// LinkedFieldsByAccount isola a busca de grupo por account_id quando accountID > 0.
func (r *UserRepo) LinkedFieldsByAccount(ctx context.Context, accountID int64, externalID string) (domain.LinkedTrackingFields, error) {
	var out domain.LinkedTrackingFields
	u, err := r.FindByExternalIDByAccount(ctx, externalID, accountID)
	if err != nil {
		return out, err
	}
	if u.GroupID == nil {
		return domain.LinkedTrackingFields{
			FBP: u.FBP, FBC: u.FBC, GCLID: u.GCLID, WBraid: u.WBraid, GBraid: u.GBraid,
			TTClid: u.TTClid, TTP: u.TTP, GA4ClientID: u.GA4ClientID, GA4SessionID: u.GA4SessionID,
		}, nil
	}

	rows, err := r.pool.Query(ctx, `SELECT `+userColumns+` FROM tracked_users WHERE group_id = $1 AND ($2 = 0 OR account_id = $2) ORDER BY updated_at DESC`, *u.GroupID, u.AccountID)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	fill := func(dst *string, v string) {
		if *dst == "" && v != "" {
			*dst = v
		}
	}
	for rows.Next() {
		m, err := scanUser(rows)
		if err != nil {
			return out, err
		}
		fill(&out.FBP, m.FBP)
		fill(&out.FBC, m.FBC)
		fill(&out.GCLID, m.GCLID)
		fill(&out.WBraid, m.WBraid)
		fill(&out.GBraid, m.GBraid)
		fill(&out.TTClid, m.TTClid)
		fill(&out.TTP, m.TTP)
		fill(&out.GA4ClientID, m.GA4ClientID)
		fill(&out.GA4SessionID, m.GA4SessionID)
	}
	return out, rows.Err()
}

func scanUser(row pgx.Row) (*domain.TrackedUser, error) {
	var u domain.TrackedUser
	var rulesJSON []byte
	err := row.Scan(
		&u.ID, &u.ExternalID, &u.IDWpp, &u.PixelName, &u.AccountID, &u.GroupID,
		&u.FBP, &u.FBC, &u.GCLID, &u.WBraid, &u.GBraid, &u.TTClid, &u.TTP,
		&u.ClientIP, &u.ClientUserAgent,
		&u.Country, &u.State, &u.City, &u.PostalCode,
		&u.FirstName, &u.LastName, &u.Email, &u.Phone, &u.URL, &u.FirstURL,
		&u.FirstUTMSource, &u.FirstUTMMedium, &u.FirstUTMCampaign, &u.FirstUTMContent, &u.FirstUTMTerm,
		&u.LastUTMSource, &u.LastUTMMedium, &u.LastUTMCampaign, &u.LastUTMContent, &u.LastUTMTerm,
		&u.CtwaClid, &u.WaChatID, &u.WaName, &u.WaAdID, &u.WaSourceApp,
		&u.GA4ClientID, &u.GA4SessionID, &u.WaLid,
		&u.CreatedAt, &u.UpdatedAt,
		&u.PageViewCount, &u.CheckoutCount, &u.LastSeenAt,
		&u.Referrer, &u.TrafficType, &u.TrafficScore, &u.TrafficCategory, &rulesJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(rulesJSON) > 0 && string(rulesJSON) != "[]" && string(rulesJSON) != "null" && u.TrafficType == "bot" {
		u.TrafficRules = json.RawMessage(rulesJSON)
	} else if u.ClientUserAgent != "" {
		res := traffic.Classify(u.ClientUserAgent, u.Referrer, false)
		if res.Type == "bot" || len(rulesJSON) == 0 || string(rulesJSON) == "[]" || string(rulesJSON) == "null" {
			u.TrafficType, u.TrafficScore, u.TrafficCategory = res.Type, res.Score, res.Category
			if rj, err := json.Marshal(res.Rules); err == nil {
				u.TrafficRules = json.RawMessage(rj)
			} else {
				u.TrafficRules = json.RawMessage("[]")
			}
		} else {
			u.TrafficRules = json.RawMessage(rulesJSON)
		}
	} else {
		u.TrafficRules = json.RawMessage("[]")
	}
	return &u, nil
}
