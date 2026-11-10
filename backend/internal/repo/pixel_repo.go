package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

var ErrNotFound = errors.New("not found")

type PixelRepo struct {
	pool  *pgxpool.Pool
	rdb   *redis.Client
	cache *cache.PixelCache
}

func NewPixelRepo(pool *pgxpool.Pool) *PixelRepo {
	return &PixelRepo{pool: pool}
}

func (r *PixelRepo) SetRedis(rdb *redis.Client) {
	r.rdb = rdb
	r.cache = cache.NewPixelCache(rdb)
}

func (r *PixelRepo) PixelCache() *cache.PixelCache {
	if r == nil {
		return nil
	}
	return r.cache
}

func (r *PixelRepo) InvalidateCache(ctx context.Context, keys ...string) {
	if r.cache != nil {
		r.cache.InvalidateKeys(ctx, keys...)
	}
	cache.PublishInvalidate(ctx, r.rdb, cache.InvalidateEvent{Kind: "pixel", Keys: keys})
}

const pixelColumns = `id, name, label, public_token, pixel_ids, access_tokens, test_codes,
		domains, tracker_config, enabled, tracking_host, account_id,
		tiktok_pixel_ids, tiktok_access_tokens, tiktok_test_codes,
		google_ads_customer_id, google_ads_login_customer_id,
		google_ads_conversion_action_id, google_ads_refresh_token,
		google_ads_oauth_email, google_ads_account_id,
		ga4_measurement_id, ga4_api_secret, meta_permanent_token, meta_ad_account_id, meta_account_id,
		dashboard_config, dashboard_share_token,
		aggregation_enabled, aggregation_window_minutes,
		created_at, updated_at`

func (r *PixelRepo) Create(ctx context.Context, p *domain.Pixel) error {
	cfgJSON, err := json.Marshal(p.TrackerConfig)
	if err != nil {
		return fmt.Errorf("marshal tracker_config: %w", err)
	}
	if p.AccountID == 0 {
		p.AccountID = 1 // default account
	}
	// Gera token público opaco (128 bits hex) se não fornecido.
	if p.PublicToken == "" {
		p.PublicToken = domain.GeneratePublicToken()
	}

	encAccessTokens := make([]string, len(p.AccessTokens))
	for i, t := range p.AccessTokens {
		encAccessTokens[i] = util.Encrypt(t)
	}
	encTikTokTokens := make([]string, len(p.TikTokAccessTokens))
	for i, t := range p.TikTokAccessTokens {
		encTikTokTokens[i] = util.Encrypt(t)
	}
	encGA4Secret := util.Encrypt(p.GA4APISecret)
	encMetaPermanentToken := util.Encrypt(p.MetaPermanentToken)

	if p.AggregationWindowMinutes <= 0 {
		p.AggregationWindowMinutes = 10
	}

	const q = `INSERT INTO pixels (name, label, public_token, pixel_ids, access_tokens, test_codes,
	                               domains, tracker_config, enabled, tracking_host, account_id,
	                               tiktok_pixel_ids, tiktok_access_tokens, tiktok_test_codes,
	                               google_ads_customer_id, google_ads_login_customer_id,
	                               google_ads_conversion_action_id, google_ads_refresh_token,
	                               google_ads_oauth_email, google_ads_account_id,
	                               ga4_measurement_id, ga4_api_secret, meta_permanent_token, meta_ad_account_id, meta_account_id,
	                               aggregation_enabled, aggregation_window_minutes)
	           VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
	           RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, q,
		p.Name, p.Label, p.PublicToken, nilIfNil(p.PixelIDs), nilIfNil(encAccessTokens), nilIfNil(p.TestCodes),
		nilIfNil(p.Domains), cfgJSON, p.Enabled, p.TrackingHost, p.AccountID,
		nilIfNil(p.TikTokPixelIDs), nilIfNil(encTikTokTokens), nilIfNil(p.TikTokTestCodes),
		p.GoogleAdsCustomerID, p.GoogleAdsLoginCustomerID,
		p.GoogleAdsConversionActionID, p.GoogleAdsRefreshToken,
		p.GoogleAdsOAuthEmail, p.GoogleAdsAccountID,
		p.GA4MeasurementID, encGA4Secret, encMetaPermanentToken, p.MetaAdAccountID, p.MetaAccountID,
		p.AggregationEnabled, p.AggregationWindowMinutes,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *PixelRepo) Update(ctx context.Context, p *domain.Pixel) error {
	cfgJSON, err := json.Marshal(p.TrackerConfig)
	if err != nil {
		return fmt.Errorf("marshal tracker_config: %w", err)
	}

	if p.AggregationWindowMinutes <= 0 {
		p.AggregationWindowMinutes = 10
	}

	encAccessTokens := make([]string, len(p.AccessTokens))
	for i, t := range p.AccessTokens {
		encAccessTokens[i] = util.Encrypt(t)
	}
	encTikTokTokens := make([]string, len(p.TikTokAccessTokens))
	for i, t := range p.TikTokAccessTokens {
		encTikTokTokens[i] = util.Encrypt(t)
	}
	encGA4Secret := util.Encrypt(p.GA4APISecret)
	encMetaPermanentToken := util.Encrypt(p.MetaPermanentToken)

	const q = `UPDATE pixels
	           SET name=$1, label=$2, pixel_ids=$3, access_tokens=$4, test_codes=$5,
	               domains=$6, tracker_config=$7, enabled=$8, tracking_host=$9,
	               tiktok_pixel_ids=$10, tiktok_access_tokens=$11, tiktok_test_codes=$12,
	               google_ads_account_id=$13,
	               ga4_measurement_id=$14, ga4_api_secret=$15, meta_permanent_token=$16, meta_ad_account_id=$17, meta_account_id=$18,
	               aggregation_enabled=$19, aggregation_window_minutes=$20,
	               account_id=$21,
	               updated_at=NOW()
	           WHERE id=$22
	           RETURNING updated_at`
	return r.pool.QueryRow(ctx, q,
		p.Name, p.Label, nilIfNil(p.PixelIDs), nilIfNil(encAccessTokens), nilIfNil(p.TestCodes),
		nilIfNil(p.Domains), cfgJSON, p.Enabled, p.TrackingHost,
		nilIfNil(p.TikTokPixelIDs), nilIfNil(encTikTokTokens), nilIfNil(p.TikTokTestCodes),
		p.GoogleAdsAccountID,
		p.GA4MeasurementID, encGA4Secret, encMetaPermanentToken, p.MetaAdAccountID, p.MetaAccountID,
		p.AggregationEnabled, p.AggregationWindowMinutes,
		p.AccountID,
		p.ID,
	).Scan(&p.UpdatedAt)
}

// nilIfNil garante que slices nil viram []string{} (postgres TEXT[] não aceita NULL
// quando a coluna é NOT NULL DEFAULT '{}').
func nilIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// RegenerateToken gera um novo public_token para o pixel.
// O token antigo deixa de funcionar imediatamente — instalações existentes
// precisarão ter o snippet atualizado.
func (r *PixelRepo) RegenerateToken(ctx context.Context, id int64) (string, error) {
	newToken := domain.GeneratePublicToken()
	_, err := r.pool.Exec(ctx,
		"UPDATE pixels SET public_token = $1, updated_at = NOW() WHERE id = $2",
		newToken, id,
	)
	if err != nil {
		return "", err
	}
	return newToken, nil
}

func (r *PixelRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pixels WHERE id = $1`, id)
	return err
}

// UpdateDashboardConfig atualiza apenas a config da dashboard do pixel
// (cards, ordem, funil, nomes customizados). Cada pixel tem a sua.
func (r *PixelRepo) UpdateDashboardConfig(ctx context.Context, id int64, cfg domain.DashboardConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal dashboard_config: %w", err)
	}
	const q = `UPDATE pixels SET dashboard_config = $1, updated_at = NOW() WHERE id = $2 RETURNING id`
	var gotID int64
	err = r.pool.QueryRow(ctx, q, raw, id).Scan(&gotID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// GenerateDashboardShareToken cria (ou substitui) o token do link público
// de visualização da dashboard do pixel (view-only, sem login).
func (r *PixelRepo) GenerateDashboardShareToken(ctx context.Context, id int64) (string, error) {
	newToken := domain.GenerateDashboardShareToken(id)
	_, err := r.pool.Exec(ctx,
		"UPDATE pixels SET dashboard_share_token = $1, updated_at = NOW() WHERE id = $2",
		newToken, id,
	)
	if err != nil {
		return "", err
	}
	return newToken, nil
}

// RevokeDashboardShareToken desativa o link público — o token antigo para
// de resolver qualquer pixel.
func (r *PixelRepo) RevokeDashboardShareToken(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE pixels SET dashboard_share_token = '', updated_at = NOW() WHERE id = $1",
		id,
	)
	return err
}
