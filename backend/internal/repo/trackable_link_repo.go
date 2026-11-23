package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type TrackableLinkRepo struct {
	pool *pgxpool.Pool
}

func NewTrackableLinkRepo(pool *pgxpool.Pool) *TrackableLinkRepo {
	return &TrackableLinkRepo{pool: pool}
}

const selectTrackableLinkCols = `
	l.id, l.account_id, l.pixel_id,
	COALESCE(p.name, ''), COALESCE(p.label, ''), COALESCE(p.tracking_host, ''),
	l.slug, l.name, l.routing_mode, l.whatsapp_instance_id,
	COALESCE(wi.name, ''), COALESCE(wi.provider, ''), COALESCE(wi.connection_status, ''),
	l.fallback_phone, l.prefill_message,
	l.utm_source, l.utm_medium, l.utm_campaign, l.utm_content, l.utm_term,
	l.send_events_enabled, l.event_trigger_mode, l.event_name,
	l.send_meta, l.send_tiktok, l.send_google_ads, l.send_ga4,
	l.enabled, l.created_at, l.updated_at
`

func scanTrackableLink(row pgx.Row, out *domain.TrackableLink) error {
	err := row.Scan(
		&out.ID, &out.AccountID, &out.PixelID,
		&out.PixelName, &out.PixelLabel, &out.TrackingHost,
		&out.Slug, &out.Name, &out.RoutingMode, &out.WhatsAppInstanceID,
		&out.InstanceName, &out.InstanceProvider, &out.InstanceStatus,
		&out.FallbackPhone, &out.PrefillMessage,
		&out.UTMSource, &out.UTMMedium, &out.UTMCampaign, &out.UTMContent, &out.UTMTerm,
		&out.SendEventsEnabled, &out.EventTriggerMode, &out.EventName,
		&out.SendMeta, &out.SendTikTok, &out.SendGoogleAds, &out.SendGA4,
		&out.Enabled, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return err
	}
	out.RoutingMode = domain.NormalizeLinkRoutingMode(out.RoutingMode)
	out.EventTriggerMode = domain.NormalizeLinkTriggerMode(out.EventTriggerMode)
	out.EventName = domain.NormalizeLinkEventName(out.EventName)
	return nil
}

func (r *TrackableLinkRepo) Create(ctx context.Context, l *domain.TrackableLink) error {
	l.Slug = strings.ToLower(strings.TrimSpace(l.Slug))
	l.RoutingMode = domain.NormalizeLinkRoutingMode(l.RoutingMode)
	l.EventTriggerMode = domain.NormalizeLinkTriggerMode(l.EventTriggerMode)
	l.EventName = domain.NormalizeLinkEventName(l.EventName)

	return r.pool.QueryRow(ctx, `
		INSERT INTO whatsapp_trackable_links (
			account_id, pixel_id, slug, name,
			routing_mode, whatsapp_instance_id, fallback_phone, prefill_message,
			utm_source, utm_medium, utm_campaign, utm_content, utm_term,
			send_events_enabled, event_trigger_mode, event_name,
			send_meta, send_tiktok, send_google_ads, send_ga4, enabled
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
		RETURNING id, created_at, updated_at
	`,
		l.AccountID, l.PixelID, l.Slug, strings.TrimSpace(l.Name),
		l.RoutingMode, l.WhatsAppInstanceID, strings.TrimSpace(l.FallbackPhone), l.PrefillMessage,
		strings.TrimSpace(l.UTMSource), strings.TrimSpace(l.UTMMedium), strings.TrimSpace(l.UTMCampaign),
		strings.TrimSpace(l.UTMContent), strings.TrimSpace(l.UTMTerm),
		l.SendEventsEnabled, l.EventTriggerMode, l.EventName,
		l.SendMeta, l.SendTikTok, l.SendGoogleAds, l.SendGA4, l.Enabled,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
}

func (r *TrackableLinkRepo) Update(ctx context.Context, l *domain.TrackableLink) error {
	l.Slug = strings.ToLower(strings.TrimSpace(l.Slug))
	l.RoutingMode = domain.NormalizeLinkRoutingMode(l.RoutingMode)
	l.EventTriggerMode = domain.NormalizeLinkTriggerMode(l.EventTriggerMode)
	l.EventName = domain.NormalizeLinkEventName(l.EventName)

	tag, err := r.pool.Exec(ctx, `
		UPDATE whatsapp_trackable_links SET
			slug                 = $3,
			name                 = $4,
			routing_mode         = $5,
			whatsapp_instance_id = $6,
			fallback_phone       = $7,
			prefill_message      = $8,
			utm_source           = $9,
			utm_medium           = $10,
			utm_campaign         = $11,
			utm_content          = $12,
			utm_term             = $13,
			send_events_enabled  = $14,
			event_trigger_mode   = $15,
			event_name           = $16,
			send_meta            = $17,
			send_tiktok          = $18,
			send_google_ads      = $19,
			send_ga4             = $20,
			enabled              = $21,
			updated_at           = NOW()
		WHERE id = $1 AND account_id = $2
	`,
		l.ID, l.AccountID, l.Slug, strings.TrimSpace(l.Name),
		l.RoutingMode, l.WhatsAppInstanceID, strings.TrimSpace(l.FallbackPhone), l.PrefillMessage,
		strings.TrimSpace(l.UTMSource), strings.TrimSpace(l.UTMMedium), strings.TrimSpace(l.UTMCampaign),
		strings.TrimSpace(l.UTMContent), strings.TrimSpace(l.UTMTerm),
		l.SendEventsEnabled, l.EventTriggerMode, l.EventName,
		l.SendMeta, l.SendTikTok, l.SendGoogleAds, l.SendGA4, l.Enabled,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TrackableLinkRepo) Delete(ctx context.Context, accountID, pixelID, id int64) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM whatsapp_trackable_links
		WHERE id = $1 AND pixel_id = $2 AND ($3 = 0 OR account_id = $3)
	`, id, pixelID, accountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TrackableLinkRepo) GetByID(ctx context.Context, id int64) (*domain.TrackableLink, error) {
	var out domain.TrackableLink
	row := r.pool.QueryRow(ctx, `
		SELECT `+selectTrackableLinkCols+`
		FROM whatsapp_trackable_links l
		JOIN pixels p ON l.pixel_id = p.id
		LEFT JOIN whatsapp_instances wi ON l.whatsapp_instance_id = wi.id
		WHERE l.id = $1
	`, id)
	if err := scanTrackableLink(row, &out); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *TrackableLinkRepo) FindBySlug(ctx context.Context, slug string) (*domain.TrackableLink, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return nil, ErrNotFound
	}
	var out domain.TrackableLink
	row := r.pool.QueryRow(ctx, `
		SELECT `+selectTrackableLinkCols+`
		FROM whatsapp_trackable_links l
		JOIN pixels p ON l.pixel_id = p.id
		LEFT JOIN whatsapp_instances wi ON l.whatsapp_instance_id = wi.id
		WHERE l.slug = $1
	`, slug)
	if err := scanTrackableLink(row, &out); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *TrackableLinkRepo) ListByPixel(ctx context.Context, accountID, pixelID int64) ([]domain.TrackableLink, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+selectTrackableLinkCols+`
		FROM whatsapp_trackable_links l
		JOIN pixels p ON l.pixel_id = p.id
		LEFT JOIN whatsapp_instances wi ON l.whatsapp_instance_id = wi.id
		WHERE l.pixel_id = $1 AND ($2 = 0 OR l.account_id = $2)
		ORDER BY l.id DESC
	`, pixelID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.TrackableLink, 0)
	for rows.Next() {
		var item domain.TrackableLink
		if err := scanTrackableLink(rows, &item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
