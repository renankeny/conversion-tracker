package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const auditEnrichedFromSQL = `
	FROM whatsapp_chats c
	JOIN whatsapp_instances i ON i.id = c.instance_id
	LEFT JOIN pixels px ON px.id = i.pixel_id
	JOIN tracked_users tu ON tu.id = c.tracked_user_id
	LEFT JOIN crm_leads cl ON cl.id = COALESCE(
		c.crm_lead_id,
		(SELECT l2.id FROM crm_leads l2 WHERE l2.account_id = i.account_id AND (l2.tracked_user_id = tu.id OR (tu.group_id IS NOT NULL AND l2.group_id = tu.group_id)) ORDER BY l2.updated_at DESC LIMIT 1)
	)
	LEFT JOIN crm_stages cs ON cs.id = cl.stage_id
	LEFT JOIN whatsapp_trackable_links tl ON tl.id = COALESCE(c.trackable_link_id, tu.trackable_link_id)
`

func (r *WhatsAppRepo) ListAuditConversations(ctx context.Context, f domain.ConversationAuditFilter) (*domain.ConversationAuditResponse, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 5000 {
		f.Limit = 5000
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	baseWhere, baseArgs := buildAuditBaseWhere(f)

	resp := &domain.ConversationAuditResponse{
		Items:     make([]domain.ConversationAuditItem, 0),
		Campaigns: make([]string, 0),
		UTMs:      make([]string, 0),
	}

	// 1. Contadores por origem (calculados sobre o filtro base sem restringir origin_category,
	// mantendo todos os contadores das pílulas visíveis ao alternar entre canais).
	originSQL := `
		WITH base AS (
			SELECT (` + auditOriginCaseSQL + `) AS origin_cat
			` + auditEnrichedFromSQL + `
			WHERE ` + baseWhere + `
		)
		SELECT
			COUNT(*)::INT,
			COUNT(*) FILTER (WHERE origin_cat = 'meta_ads')::INT,
			COUNT(*) FILTER (WHERE origin_cat = 'google_ads')::INT,
			COUNT(*) FILTER (WHERE origin_cat = 'tiktok_ads')::INT,
			COUNT(*) FILTER (WHERE origin_cat = 'bio_link')::INT,
			COUNT(*) FILTER (WHERE origin_cat = 'site')::INT
		FROM base
	`
	if err := r.db.QueryRow(ctx, originSQL, baseArgs...).Scan(
		&resp.OriginCounts.All,
		&resp.OriginCounts.MetaAds,
		&resp.OriginCounts.GoogleAds,
		&resp.OriginCounts.TikTokAds,
		&resp.OriginCounts.BioLink,
		&resp.OriginCounts.Site,
	); err != nil {
		return nil, fmt.Errorf("audit origin counts: %w", err)
	}

	// 2. Filtro final incluindo origin_category (se selecionado) para KPIs e listagem.
	finalArgs := append([]any{}, baseArgs...)
	originFilterSQL := ""
	if f.OriginCategory != "" && f.OriginCategory != "all" {
		finalArgs = append(finalArgs, f.OriginCategory)
		originFilterSQL = fmt.Sprintf(" WHERE origin_category = $%d", len(finalArgs))
	}

	cteSQL := `
		WITH enriched AS (
			SELECT
				c.id AS chat_id,
				c.instance_id,
				i.name AS instance_name,
				COALESCE(i.pixel_id, 0) AS pixel_id,
				COALESCE(NULLIF(px.name, ''), COALESCE(tu.pixel_name, '')) AS pixel_name,
				COALESCE(NULLIF(i.currency, ''), 'BRL') AS currency,
				c.tracked_user_id,
				cl.id AS crm_lead_id,
				COALESCE(
					NULLIF(TRIM(CONCAT_WS(' ', tu.first_name, tu.last_name)), ''),
					NULLIF(TRIM(COALESCE(c.contact_name, '')), ''),
					NULLIF(TRIM(c.push_name), ''),
					NULLIF(c.phone, ''),
					'Lead WhatsApp'
				) AS lead_name,
				COALESCE(NULLIF(c.phone, ''), COALESCE(tu.phone, '')) AS phone,
				COALESCE(tu.email, '') AS email,
				c.profile_pic,
				(` + auditOriginCaseSQL + `) AS origin_category,
				c.origin_type,
				COALESCE(NULLIF(c.campaign_name, ''), NULLIF(tu.last_utm_campaign, ''), NULLIF(tu.first_utm_campaign, ''), '') AS campaign_name,
				COALESCE(NULLIF(c.ad_name, ''), NULLIF(tu.last_utm_content, ''), NULLIF(tu.first_utm_content, ''), '') AS ad_name,
				COALESCE(NULLIF(tu.last_utm_source, ''), NULLIF(tu.first_utm_source, ''), '') AS utm_source,
				COALESCE(tl.name, '') AS trackable_link_name,
				cl.stage_id,
				COALESCE(cs.name, '') AS stage_name,
				COALESCE(cs.color, '') AS stage_color,
				COALESCE(cl.value, 0) AS lead_value,
				(` + auditIsWonCaseSQL + `) AS is_won,
				c.ai_qualified,
				c.sla_status,
				c.total_messages,
				c.inbound_messages,
				c.outbound_messages,
				c.first_response_sec,
				c.avg_response_sec,
				c.first_inbound_at,
				c.last_inbound_at,
				c.last_outbound_at,
				c.last_message_time,
				c.created_at
			` + auditEnrichedFromSQL + `
			WHERE ` + baseWhere + `
		)
	`

	wonSubWhere := "WHERE e2.is_won = TRUE"
	if originFilterSQL != "" {
		wonSubWhere = strings.Replace(originFilterSQL, "WHERE origin_category", "WHERE e2.is_won = TRUE AND e2.origin_category", 1)
	}

	kpiSQL := cteSQL + `
		SELECT
			COUNT(*)::INT,
			COUNT(*) FILTER (WHERE sla_status IN ('unanswered', 'waiting_reply'))::INT,
			COUNT(*) FILTER (WHERE sla_status = 'unanswered')::INT,
			ROUND(AVG(first_response_sec))::INT,
			COUNT(DISTINCT COALESCE(crm_lead_id, -chat_id)) FILTER (WHERE is_won = TRUE)::INT,
			COALESCE((
				SELECT SUM(w.lead_value) FROM (
					SELECT DISTINCT ON (COALESCE(e2.crm_lead_id, -e2.chat_id)) e2.lead_value
					FROM enriched e2
					` + wonSubWhere + `
					ORDER BY COALESCE(e2.crm_lead_id, -e2.chat_id), e2.lead_value DESC
				) w
			), 0)
		FROM enriched
	` + originFilterSQL
	if err := r.db.QueryRow(ctx, kpiSQL, finalArgs...).Scan(
		&resp.Total,
		&resp.KPIs.WaitingReplyCount,
		&resp.KPIs.UnansweredCount,
		&resp.KPIs.AvgFirstResponseSec,
		&resp.KPIs.WonCount,
		&resp.KPIs.WonRevenue,
	); err != nil {
		return nil, fmt.Errorf("audit kpis: %w", err)
	}
	resp.KPIs.TotalConversations = resp.Total

	// 3. Opções distintas de Campanha e UTM para os dropdowns de filtro rápido.
	_ = r.loadAuditDistinctFacets(ctx, baseWhere, baseArgs, resp)

	// 4. Paginação ordenada com tags agregadas por lead.
	listArgs := append([]any{}, finalArgs...)
	listArgs = append(listArgs, f.Limit, f.Offset)
	limitPos := len(listArgs) - 1
	offsetPos := len(listArgs)

	listSQL := cteSQL + `
		SELECT
			e.chat_id, e.instance_id, e.instance_name, e.pixel_id, e.pixel_name, e.currency,
			e.tracked_user_id, e.crm_lead_id, e.lead_name, e.phone, e.email, e.profile_pic,
			e.origin_category, e.origin_type, e.campaign_name, e.ad_name, e.utm_source, e.trackable_link_name,
			e.stage_id, e.stage_name, e.stage_color, e.lead_value, e.is_won,
			COALESCE((
				SELECT json_agg(json_build_object('id', tg.id, 'name', tg.name, 'color', tg.color) ORDER BY tg.name ASC)
				FROM crm_lead_tags lt
				JOIN crm_tags tg ON tg.id = lt.crm_tag_id
				WHERE e.crm_lead_id IS NOT NULL AND lt.crm_lead_id = e.crm_lead_id
			), '[]'::json) AS tags_json,
			e.ai_qualified, e.sla_status, e.total_messages, e.inbound_messages, e.outbound_messages,
			e.first_response_sec, e.avg_response_sec, e.first_inbound_at, e.last_inbound_at, e.last_outbound_at,
			e.last_message_time, e.created_at
		FROM enriched e
	` + strings.Replace(originFilterSQL, "WHERE origin_category", "WHERE e.origin_category", 1) + `
		ORDER BY ` + resolveAuditOrderBy(f.SortBy, f.SortDir) + `
		LIMIT $` + fmt.Sprintf("%d", limitPos) + ` OFFSET $` + fmt.Sprintf("%d", offsetPos)

	rows, err := r.db.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("audit list rows: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var it domain.ConversationAuditItem
		var tagsRaw []byte
		if err := rows.Scan(
			&it.ChatID, &it.InstanceID, &it.InstanceName, &it.PixelID, &it.PixelName, &it.Currency,
			&it.TrackedUserID, &it.CRMLeadID, &it.LeadName, &it.Phone, &it.Email, &it.ProfilePic,
			&it.OriginCategory, &it.OriginType, &it.CampaignName, &it.AdName, &it.UTMSource, &it.TrackableLinkName,
			&it.StageID, &it.StageName, &it.StageColor, &it.LeadValue, &it.IsWon,
			&tagsRaw,
			&it.AIQualified, &it.SLAStatus, &it.TotalMessages, &it.InboundMessages, &it.OutboundMessages,
			&it.FirstResponseSec, &it.AvgResponseSec, &it.FirstInboundAt, &it.LastInboundAt, &it.LastOutboundAt,
			&it.LastMessageTime, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit row: %w", err)
		}
		it.UTMSource = util.StripUserIDFromUTM(it.UTMSource)
		it.Tags = []domain.CRMTag{}
		if len(tagsRaw) > 0 {
			_ = json.Unmarshal(tagsRaw, &it.Tags)
		}
		resp.Items = append(resp.Items, it)
	}
	return resp, rows.Err()
}
