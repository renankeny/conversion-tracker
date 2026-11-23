package repo

import (
	"fmt"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

const auditOriginCaseSQL = `
	CASE
		WHEN c.origin_type IN ('meta_ads_ctwa', 'meta_ads_lp')
		  OR COALESCE(c.ctwa_clid, '') <> ''
		  OR COALESCE(tu.ctwa_clid, '') <> ''
		  OR COALESCE(tu.fbc, '') <> ''
		  OR LOWER(COALESCE(NULLIF(tu.last_utm_source, ''), tu.first_utm_source, '')) ~ '(facebook|instagram|^fb$|^ig$|meta)'
		THEN 'meta_ads'
		WHEN c.origin_type = 'google_ads'
		  OR COALESCE(tu.gclid, '') <> ''
		  OR COALESCE(tu.gbraid, '') <> ''
		  OR COALESCE(tu.wbraid, '') <> ''
		  OR LOWER(COALESCE(NULLIF(tu.last_utm_source, ''), tu.first_utm_source, '')) ~ '(google|adwords|youtube|gads)'
		THEN 'google_ads'
		WHEN c.origin_type = 'tiktok_ads'
		  OR COALESCE(tu.ttclid, '') <> ''
		  OR LOWER(COALESCE(NULLIF(tu.last_utm_source, ''), tu.first_utm_source, '')) ~ '(tiktok|^tt$|bytedance)'
		THEN 'tiktok_ads'
		WHEN c.origin_type = 'bio_link_organic'
		  OR c.trackable_link_id IS NOT NULL
		  OR tu.trackable_link_id IS NOT NULL
		THEN 'bio_link'
		ELSE 'site'
	END
`

const auditIsWonCaseSQL = `
	CASE
		WHEN cs.id IS NOT NULL AND (
			cs.trigger_event = 'Purchase'
			OR cs.trigger_event_ctwa = 'Purchase'
			OR (
				cs.name !~* '(pr[eé][\s-]*venda|p[oó]s[\s-]*venda)'
				AND cs.name ~* '\y(ganh[oa]s?|fechad[oa]s?|conclu[ií]d[oa]s?|comprou|pago|venda\s+(ganha|fechada|conclu[ií]da|realizada))\y'
			)
		) THEN TRUE
		ELSE FALSE
	END
`

// buildAuditBaseWhere monta as cláusulas WHERE comuns (exceto origin_category,
// permitindo reutilizar o mesmo conjunto base para calcular os contadores das pílulas de origem).
func buildAuditBaseWhere(f domain.ConversationAuditFilter) (string, []any) {
	conds := []string{
		"i.account_id = $1",
		"c.tracked_user_id IS NOT NULL",
	}
	args := []any{f.AccountID}
	nextIdx := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if !f.From.IsZero() {
		conds = append(conds, "c.last_message_time >= "+nextIdx(f.From))
	}
	if !f.To.IsZero() {
		conds = append(conds, "c.last_message_time <= "+nextIdx(f.To))
	}
	if f.PixelID > 0 && f.PixelName != "" {
		pID := nextIdx(f.PixelID)
		pName := nextIdx(f.PixelName)
		conds = append(conds, fmt.Sprintf("(i.pixel_id = %s OR tu.pixel_name = %s)", pID, pName))
	} else if f.PixelID > 0 {
		conds = append(conds, "i.pixel_id = "+nextIdx(f.PixelID))
	} else if f.PixelName != "" {
		pName := nextIdx(f.PixelName)
		conds = append(conds, fmt.Sprintf("(tu.pixel_name = %s OR px.name = %s)", pName, pName))
	}
	if f.InstanceID > 0 {
		conds = append(conds, "c.instance_id = "+nextIdx(f.InstanceID))
	}
	switch f.SLAStatus {
	case "pending":
		conds = append(conds, "c.sla_status IN ('unanswered', 'waiting_reply')")
	case "unanswered", "waiting_reply", "replied":
		conds = append(conds, "c.sla_status = "+nextIdx(f.SLAStatus))
	}
	if f.StageID > 0 {
		conds = append(conds, "cl.stage_id = "+nextIdx(f.StageID))
	}
	if f.TagID > 0 {
		conds = append(conds, fmt.Sprintf("EXISTS (SELECT 1 FROM crm_lead_tags lt WHERE lt.crm_lead_id = cl.id AND lt.crm_tag_id = %s)", nextIdx(f.TagID)))
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		pat := nextIdx("%" + s + "%")
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, s)
		if digits != "" {
			dPat := nextIdx("%" + digits + "%")
			conds = append(conds, fmt.Sprintf("(COALESCE(c.contact_name,'') ILIKE %s OR c.push_name ILIKE %s OR CONCAT_WS(' ', tu.first_name, tu.last_name) ILIKE %s OR c.phone LIKE %s OR COALESCE(tu.phone,'') LIKE %s OR c.campaign_name ILIKE %s)", pat, pat, pat, dPat, dPat, pat))
		} else {
			conds = append(conds, fmt.Sprintf("(COALESCE(c.contact_name,'') ILIKE %s OR c.push_name ILIKE %s OR CONCAT_WS(' ', tu.first_name, tu.last_name) ILIKE %s OR COALESCE(tu.email,'') ILIKE %s OR c.campaign_name ILIKE %s)", pat, pat, pat, pat, pat))
		}
	}
	if camp := strings.TrimSpace(f.Campaign); camp != "" {
		pCamp := nextIdx(camp)
		conds = append(conds, fmt.Sprintf("COALESCE(NULLIF(c.campaign_name, ''), NULLIF(tu.last_utm_campaign, ''), NULLIF(tu.first_utm_campaign, '')) = %s", pCamp))
	}
	if utm := strings.TrimSpace(f.UTM); utm != "" {
		pUtm := nextIdx("%" + utm + "%")
		conds = append(conds, fmt.Sprintf("(COALESCE(tu.last_utm_source,'') ILIKE %s OR COALESCE(tu.first_utm_source,'') ILIKE %s OR COALESCE(tu.last_utm_campaign,'') ILIKE %s)", pUtm, pUtm, pUtm))
	}

	return strings.Join(conds, " AND "), args
}

func resolveAuditOrderBy(sortBy, sortDir string) string {
	dir := "DESC"
	if strings.EqualFold(sortDir, "asc") {
		dir = "ASC"
	}
	switch sortBy {
	case "created_at":
		return "created_at " + dir + ", chat_id DESC"
	case "first_response_sec":
		return "first_response_sec " + dir + " NULLS LAST, last_message_time DESC"
	case "avg_response_sec":
		return "avg_response_sec " + dir + " NULLS LAST, last_message_time DESC"
	case "last_inbound_at":
		return "last_inbound_at " + dir + " NULLS LAST, last_message_time DESC"
	case "last_outbound_at":
		return "last_outbound_at " + dir + " NULLS LAST, last_message_time DESC"
	case "total_messages":
		return "total_messages " + dir + ", last_message_time DESC"
	case "lead_value":
		return "lead_value " + dir + ", last_message_time DESC"
	default:
		return "last_message_time " + dir + ", chat_id DESC"
	}
}
