package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// EnsureLead cria o card do kanban pra esse tracked_user, se ainda não existir na pessoa (group_id) ou no próprio ID.
func (r *CRMRepo) EnsureLead(ctx context.Context, accountID, trackedUserID, stageID int64) error {
	_, err := r.EnsureLeadReturningID(ctx, accountID, trackedUserID, stageID)
	return err
}

func (r *CRMRepo) EnsureLeadReturningID(ctx context.Context, accountID, trackedUserID, stageID int64) (int64, error) {
	groupID, err := r.groupIDOf(ctx, trackedUserID)
	if err != nil {
		return 0, err
	}
	if groupID != nil {
		var existing int64
		err := r.pool.QueryRow(ctx,
			`SELECT id FROM crm_leads WHERE account_id = $1 AND group_id = $2 LIMIT 1`,
			accountID, *groupID,
		).Scan(&existing)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, err
		}
	}

	var id int64
	err = r.pool.QueryRow(ctx, `
		INSERT INTO crm_leads (account_id, tracked_user_id, stage_id, group_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (account_id, tracked_user_id) DO UPDATE SET updated_at = crm_leads.updated_at
		RETURNING id
	`, accountID, trackedUserID, stageID, groupID).Scan(&id)
	return id, err
}

func (r *CRMRepo) GetLeads(ctx context.Context, accountID int64, pixel string) ([]domain.CRMLead, error) {
	query := `
		SELECT 
			c.id, c.account_id, c.tracked_user_id, c.stage_id, c.value, c.created_at, c.updated_at,
			COALESCE(t.first_name, ''), COALESCE(t.last_name, ''), COALESCE(t.email, ''), COALESCE(t.phone, ''),
			EXISTS(SELECT 1 FROM crm_notes n WHERE n.crm_lead_id = c.id) AS has_notes,
			(
				SELECT COALESCE(json_agg(json_build_object('id', tg.id, 'name', tg.name, 'color', tg.color) ORDER BY tg.name ASC), '[]')
				FROM crm_lead_tags lt
				JOIN crm_tags tg ON tg.id = lt.crm_tag_id
				WHERE lt.crm_lead_id = c.id
			) AS tags_json,
			COALESCE(t.external_id, ''), COALESCE(t.pixel_name, ''),
			COALESCE(t.first_utm_source, ''), COALESCE(t.last_utm_source, ''), COALESCE(t.first_utm_campaign, ''),
			COALESCE(t.city, ''), COALESCE(t.state, ''), COALESCE(t.traffic_type, ''),
			COALESCE(t.fbc, ''), COALESCE(t.gclid, ''), COALESCE(t.wbraid, ''), COALESCE(t.gbraid, ''),
			COALESCE(t.ttclid, ''), COALESCE(t.ctwa_clid, ''),
			COALESCE(c.group_id, t.group_id)
		FROM crm_leads c
		JOIN tracked_users t ON t.id = c.tracked_user_id
		WHERE c.account_id = $1
	`
	args := []any{accountID}

	if pixel != "" {
		query += ` AND t.pixel_name = $2 `
		args = append(args, pixel)
	}

	query += ` ORDER BY c.updated_at DESC `

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leads []domain.CRMLead
	for rows.Next() {
		var l domain.CRMLead
		var tu domain.TrackedUser
		var tagsJSON []byte
		if err := rows.Scan(
			&l.ID, &l.AccountID, &l.TrackedUserID, &l.StageID, &l.Value, &l.CreatedAt, &l.UpdatedAt,
			&tu.FirstName, &tu.LastName, &tu.Email, &tu.Phone,
			&l.HasNotes,
			&tagsJSON,
			&tu.ExternalID, &tu.PixelName,
			&tu.FirstUTMSource, &tu.LastUTMSource, &tu.FirstUTMCampaign,
			&tu.City, &tu.State, &tu.TrafficType,
			&tu.FBC, &tu.GCLID, &tu.WBraid, &tu.GBraid,
			&tu.TTClid, &tu.CtwaClid,
			&tu.GroupID,
		); err != nil {
			return nil, err
		}
		tu.ID = l.TrackedUserID
		tu.AccountID = l.AccountID
		tu.FirstUTMSource = util.StripUserIDFromUTM(tu.FirstUTMSource)
		tu.LastUTMSource = util.StripUserIDFromUTM(tu.LastUTMSource)
		l.TrackedUser = &tu
		if len(tagsJSON) > 0 {
			_ = json.Unmarshal(tagsJSON, &l.Tags)
		}
		leads = append(leads, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	enrichCRMLeadsGroupFields(ctx, r.pool, accountID, leads)
	return leads, nil
}

func (r *CRMRepo) GetLeadByID(ctx context.Context, id, accountID int64) (*domain.CRMLead, error) {
	var l domain.CRMLead
	var tu domain.TrackedUser
	err := r.pool.QueryRow(ctx, `
		SELECT 
			c.id, c.account_id, c.tracked_user_id, c.stage_id, c.value, c.created_at, c.updated_at,
			COALESCE(t.external_id, ''), COALESCE(t.first_name, ''), COALESCE(t.last_name, ''),
			COALESCE(t.email, ''), COALESCE(t.phone, ''), COALESCE(t.pixel_name, ''),
			COALESCE(t.fbp, ''), COALESCE(t.fbc, ''), COALESCE(t.gclid, ''), COALESCE(t.wbraid, ''), COALESCE(t.gbraid, ''),
			COALESCE(t.ttclid, ''), COALESCE(t.ttp, ''), COALESCE(t.ctwa_clid, ''), COALESCE(t.id_wpp, ''),
			COALESCE(t.client_ip, ''), COALESCE(t.client_user_agent, ''),
			COALESCE(t.ga4_client_id, ''), COALESCE(t.ga4_session_id, ''),
			COALESCE(t.country, ''), COALESCE(t.state, ''), COALESCE(t.city, ''), COALESCE(t.postal_code, ''),
			COALESCE(t.url, ''), COALESCE(t.first_url, ''),
			COALESCE(t.first_utm_source, ''), COALESCE(t.first_utm_medium, ''), COALESCE(t.first_utm_campaign, ''),
			COALESCE(t.first_utm_content, ''), COALESCE(t.first_utm_term, ''),
			COALESCE(t.last_utm_source, ''), COALESCE(t.last_utm_medium, ''), COALESCE(t.last_utm_campaign, ''),
			COALESCE(t.last_utm_content, ''), COALESCE(t.last_utm_term, ''),
			COALESCE(t.traffic_type, ''), COALESCE(c.group_id, t.group_id)
		FROM crm_leads c
		JOIN tracked_users t ON t.id = c.tracked_user_id
		WHERE c.id = $1 AND c.account_id = $2
	`, id, accountID).Scan(
		&l.ID, &l.AccountID, &l.TrackedUserID, &l.StageID, &l.Value, &l.CreatedAt, &l.UpdatedAt,
		&tu.ExternalID, &tu.FirstName, &tu.LastName,
		&tu.Email, &tu.Phone, &tu.PixelName,
		&tu.FBP, &tu.FBC, &tu.GCLID, &tu.WBraid, &tu.GBraid,
		&tu.TTClid, &tu.TTP, &tu.CtwaClid, &tu.IDWpp,
		&tu.ClientIP, &tu.ClientUserAgent,
		&tu.GA4ClientID, &tu.GA4SessionID,
		&tu.Country, &tu.State, &tu.City, &tu.PostalCode,
		&tu.URL, &tu.FirstURL,
		&tu.FirstUTMSource, &tu.FirstUTMMedium, &tu.FirstUTMCampaign,
		&tu.FirstUTMContent, &tu.FirstUTMTerm,
		&tu.LastUTMSource, &tu.LastUTMMedium, &tu.LastUTMCampaign,
		&tu.LastUTMContent, &tu.LastUTMTerm,
		&tu.TrafficType, &tu.GroupID,
	)
	if err != nil {
		return nil, err
	}
	tu.ID = l.TrackedUserID
	tu.AccountID = l.AccountID
	tu.FirstUTMSource = util.StripUserIDFromUTM(tu.FirstUTMSource)
	tu.LastUTMSource = util.StripUserIDFromUTM(tu.LastUTMSource)
	enrichTrackedUserGroupFields(ctx, r.pool, &tu)
	l.TrackedUser = &tu

	if nRows, err := r.pool.Query(ctx, "SELECT id, content, created_at FROM crm_notes WHERE crm_lead_id = $1 ORDER BY created_at DESC", id); err == nil {
		defer nRows.Close()
		for nRows.Next() {
			var n domain.CRMNote
			if nRows.Scan(&n.ID, &n.Content, &n.CreatedAt) == nil {
				n.CRMLeadID = id
				l.Notes = append(l.Notes, n)
			}
		}
	}

	if tRows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name, t.color, t.created_at
		FROM crm_tags t
		JOIN crm_lead_tags lt ON lt.crm_tag_id = t.id
		WHERE lt.crm_lead_id = $1 AND t.account_id = $2
		ORDER BY t.name ASC
	`, id, accountID); err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var t domain.CRMTag
			if tRows.Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt) == nil {
				t.AccountID = accountID
				l.Tags = append(l.Tags, t)
			}
		}
	}

	return &l, nil
}
