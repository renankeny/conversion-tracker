package repo

const (
	waChatIsMetaSQL   = `(c.origin_type IN ('meta_ads_ctwa', 'meta_ads_lp') OR (c.origin_type NOT IN ('google_ads', 'tiktok_ads') AND (c.ctwa_clid <> '' OR c.source_id <> '' OR (u.id IS NOT NULL AND ` + metaSourceSQL + `))))`
	waChatIsGoogleSQL = `(c.origin_type = 'google_ads' OR (c.origin_type NOT IN ('meta_ads_ctwa', 'meta_ads_lp', 'tiktok_ads') AND u.id IS NOT NULL AND ` + googleSourceSQL + `))`
	waChatIsTikTokSQL = `(c.origin_type = 'tiktok_ads' OR (c.origin_type NOT IN ('meta_ads_ctwa', 'meta_ads_lp', 'google_ads') AND u.id IS NOT NULL AND (u.ttclid <> '' OR u.last_utm_source ILIKE '%tiktok%' OR u.first_utm_source ILIKE '%tiktok%')))`
)

// waChatPeriodEntrySQL avalia se a conversa WhatsApp (alias c) teve entrada legítima
// no período [fromParam, toParam):
//  1. Conversa iniciada no período com ao menos 1 mensagem recebida do lead no período; OU
//  2. Lead enviou no período uma nova mensagem de anúncio CTWA (ctwa_clid / source_id) ou
//     mensagem com código de protocolo do tracker (novo ou repetido via cookie).
func waChatPeriodEntrySQL(fromParam, toParam string) string {
	return `((c.created_at >= ` + fromParam + ` AND c.created_at < ` + toParam + ` AND EXISTS (
		SELECT 1 FROM whatsapp_messages m0
		WHERE m0.chat_id = c.id AND m0.is_from_me = FALSE
		  AND m0.timestamp >= ` + fromParam + ` AND m0.timestamp < ` + toParam + `
	)) OR EXISTS (
		SELECT 1 FROM whatsapp_messages mp
		WHERE mp.chat_id = c.id AND mp.is_from_me = FALSE
		  AND mp.timestamp >= ` + fromParam + ` AND mp.timestamp < ` + toParam + `
		  AND (mp.ctwa_clid <> '' OR mp.source_id <> '' OR mp.protocol_code <> '')
	))`
}

// waChatEntryTimeSQL retorna o instante exato da entrada de anúncio/protocolo no período [fromParam, toParam).
func waChatEntryTimeSQL(fromParam, toParam string) string {
	return `COALESCE(
		(SELECT MIN(mp.timestamp) FROM whatsapp_messages mp
		 WHERE mp.chat_id = c.id AND mp.is_from_me = FALSE
		   AND mp.timestamp >= ` + fromParam + ` AND mp.timestamp < ` + toParam + `
		   AND (mp.ctwa_clid <> '' OR mp.source_id <> '' OR mp.protocol_code <> '')),
		c.created_at
	)`
}

func attributionSource(originType, ctwaClid, fbc, gclid, gbraid, wbraid, ttclid string) string {
	if originType != "" && originType != "unknown" {
		return originType
	}
	switch {
	case ctwaClid != "":
		return "meta_ads_ctwa"
	case fbc != "":
		return "meta_ads_lp"
	case gclid != "" || gbraid != "" || wbraid != "":
		return "google_ads"
	case ttclid != "":
		return "tiktok_ads"
	default:
		return "website_organic"
	}
}

func buildWhatsAppPortalStatsConds(f WhatsAppStatsFilter, extraConds ...string) ([]string, []any) {
	if !f.From.IsZero() && !f.To.IsZero() {
		args := []any{f.AccountID, f.From, f.To}
		conds := append([]string{"i.account_id = $1", waChatPeriodEntrySQL("$2", "$3")}, extraConds...)
		if f.PixelName != "" {
			conds = append(conds, "p.name = $4")
			args = append(args, f.PixelName)
		}
		return conds, args
	}
	return buildWhatsAppStatsConds(f, "c.created_at", extraConds...)
}
