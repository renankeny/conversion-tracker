package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func (r *WhatsAppRepo) loadAuditDistinctFacets(
	ctx context.Context,
	baseWhere string,
	baseArgs []any,
	resp *domain.ConversationAuditResponse,
) error {
	campRows, err := r.db.Query(ctx, `
		SELECT DISTINCT camp FROM (
			SELECT COALESCE(NULLIF(TRIM(c.campaign_name), ''), NULLIF(TRIM(tu.last_utm_campaign), ''), NULLIF(TRIM(tu.first_utm_campaign), '')) AS camp
			`+auditEnrichedFromSQL+`
			WHERE `+baseWhere+`
		) s
		WHERE camp IS NOT NULL AND camp <> ''
		ORDER BY camp ASC
		LIMIT 100
	`, baseArgs...)
	if err == nil {
		defer campRows.Close()
		for campRows.Next() {
			var v string
			if campRows.Scan(&v) == nil && strings.TrimSpace(v) != "" {
				resp.Campaigns = append(resp.Campaigns, v)
			}
		}
	}

	utmRows, err := r.db.Query(ctx, `
		SELECT DISTINCT utm FROM (
			SELECT COALESCE(NULLIF(TRIM(tu.last_utm_source), ''), NULLIF(TRIM(tu.first_utm_source), '')) AS utm
			`+auditEnrichedFromSQL+`
			WHERE `+baseWhere+`
		) s
		WHERE utm IS NOT NULL AND utm <> ''
		ORDER BY utm ASC
		LIMIT 100
	`, baseArgs...)
	if err == nil {
		defer utmRows.Close()
		seen := make(map[string]bool)
		for utmRows.Next() {
			var v string
			if utmRows.Scan(&v) == nil {
				clean := strings.TrimSpace(util.StripUserIDFromUTM(v))
				if clean != "" && !seen[clean] {
					seen[clean] = true
					resp.UTMs = append(resp.UTMs, clean)
				}
			}
		}
	}
	return nil
}

// GetChatByIDForPixel retorna a conversa apenas se pertencer estritamente à conta e ao Pixel informados.
func (r *WhatsAppRepo) GetChatByIDForPixel(ctx context.Context, accountID, pixelID int64, pixelName string, chatID int64) (*domain.WhatsAppChat, error) {
	var c domain.WhatsAppChat
	row := r.db.QueryRow(ctx, `
		SELECT `+selectChatColumns+`
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON i.id = c.instance_id
		LEFT JOIN tracked_users tu ON tu.id = c.tracked_user_id
		WHERE c.id = $1
		  AND i.account_id = $2
		  AND c.tracked_user_id IS NOT NULL
		  AND (i.pixel_id = $3 OR COALESCE(tu.pixel_name, '') = $4)
		LIMIT 1
	`, chatID, accountID, pixelID, pixelName)
	if err := scanWhatsAppChat(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}
