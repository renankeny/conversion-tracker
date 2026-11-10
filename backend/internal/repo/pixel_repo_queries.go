package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func (r *PixelRepo) FindByName(ctx context.Context, name string) (*domain.Pixel, error) {
	loader := func(c context.Context) (*domain.Pixel, error) {
		q := `SELECT ` + pixelColumns + ` FROM pixels WHERE name = $1`
		return scanPixel(r.pool.QueryRow(c, q, name))
	}
	if r.cache != nil {
		return r.cache.GetOrLoadPixel(ctx, name, loader)
	}
	return loader(ctx)
}

// FindByNameOrToken busca por pixel_name OU public_token com cache Redis criptografado.
func (r *PixelRepo) FindByNameOrToken(ctx context.Context, nameOrToken string) (*domain.Pixel, error) {
	loader := func(c context.Context) (*domain.Pixel, error) {
		q := `SELECT ` + pixelColumns + ` FROM pixels WHERE name = $1 OR (public_token = $1 AND public_token <> '')`
		return scanPixel(r.pool.QueryRow(c, q, nameOrToken))
	}
	if r.cache != nil {
		return r.cache.GetOrLoadPixel(ctx, nameOrToken, loader)
	}
	return loader(ctx)
}

func (r *PixelRepo) FindByID(ctx context.Context, id int64) (*domain.Pixel, error) {
	q := `SELECT ` + pixelColumns + ` FROM pixels WHERE id = $1`
	return scanPixel(r.pool.QueryRow(ctx, q, id))
}

// FindByDomain busca um pixel habilitado cujo array `domains` contenha o host informado.
// Match exato (sem wildcard). Retorna o primeiro match.
func (r *PixelRepo) FindByDomain(ctx context.Context, host string) (*domain.Pixel, error) {
	q := `SELECT ` + pixelColumns + ` FROM pixels
	      WHERE enabled = TRUE AND domains @> ARRAY[$1]::text[]
	      ORDER BY id LIMIT 1`
	return scanPixel(r.pool.QueryRow(ctx, q, host))
}

func (r *PixelRepo) List(ctx context.Context) ([]domain.Pixel, error) {
	q := `SELECT ` + pixelColumns + ` FROM pixels ORDER BY id DESC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Pixel
	for rows.Next() {
		p, err := scanPixel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ListByAccount filtra pixels por account_id. Se accountID == 0 retorna todos
// (uso owner). Caso contrário filtra (uso member).
func (r *PixelRepo) ListByAccount(ctx context.Context, accountID int64) ([]domain.Pixel, error) {
	if accountID == 0 {
		return r.List(ctx)
	}
	q := `SELECT ` + pixelColumns + ` FROM pixels WHERE account_id = $1 ORDER BY id DESC`
	rows, err := r.pool.Query(ctx, q, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Pixel
	for rows.Next() {
		p, err := scanPixel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ActiveDomains devolve a união de todos os domínios cadastrados em pixels
// com enabled = true. Usado para atualizar o DomainSet do CORS.
func (r *PixelRepo) ActiveDomains(ctx context.Context) ([]string, error) {
	const q = `SELECT DISTINCT unnest(domains) FROM pixels WHERE enabled = TRUE`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// ActiveTrackingHosts devolve todos os domínios de tracking únicos cadastrados.
// Usado para gerar a configuração dinâmica do Traefik.
func (r *PixelRepo) ActiveTrackingHosts(ctx context.Context) ([]string, error) {
	const q = `SELECT DISTINCT tracking_host FROM pixels WHERE enabled = TRUE AND tracking_host != ''`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// FindByDashboardShareToken resolve o pixel a partir do token do link
// público de dashboard. Usado pela rota view-only, sem autenticação.
func (r *PixelRepo) FindByDashboardShareToken(ctx context.Context, token string) (*domain.Pixel, error) {
	if token == "" {
		return nil, ErrNotFound
	}
	loader := func(c context.Context) (*domain.Pixel, error) {
		q := `SELECT ` + pixelColumns + ` FROM pixels WHERE dashboard_share_token = $1`
		return scanPixel(r.pool.QueryRow(c, q, token))
	}
	if r.cache != nil {
		return r.cache.GetOrLoadSharePixel(ctx, token, loader)
	}
	return loader(ctx)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPixel(row rowScanner) (*domain.Pixel, error) {
	var p domain.Pixel
	var cfgJSON []byte
	var dashCfgJSON []byte
	err := row.Scan(
		&p.ID, &p.Name, &p.Label, &p.PublicToken, &p.PixelIDs, &p.AccessTokens, &p.TestCodes,
		&p.Domains, &cfgJSON, &p.Enabled, &p.TrackingHost, &p.AccountID,
		&p.TikTokPixelIDs, &p.TikTokAccessTokens, &p.TikTokTestCodes,
		&p.GoogleAdsCustomerID, &p.GoogleAdsLoginCustomerID,
		&p.GoogleAdsConversionActionID, &p.GoogleAdsRefreshToken,
		&p.GoogleAdsOAuthEmail, &p.GoogleAdsAccountID,
		&p.GA4MeasurementID, &p.GA4APISecret, &p.MetaPermanentToken, &p.MetaAdAccountID, &p.MetaAccountID,
		&dashCfgJSON, &p.DashboardShareToken,
		&p.AggregationEnabled, &p.AggregationWindowMinutes,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(cfgJSON) > 0 {
		if err := json.Unmarshal(cfgJSON, &p.TrackerConfig); err != nil {
			return nil, fmt.Errorf("unmarshal tracker_config: %w", err)
		}
	}
	if len(dashCfgJSON) > 0 {
		if err := json.Unmarshal(dashCfgJSON, &p.DashboardConfig); err != nil {
			return nil, fmt.Errorf("unmarshal dashboard_config: %w", err)
		}
	}

	// Decrypt
	for i, t := range p.AccessTokens {
		p.AccessTokens[i] = util.Decrypt(t)
	}
	for i, t := range p.TikTokAccessTokens {
		p.TikTokAccessTokens[i] = util.Decrypt(t)
	}
	p.GA4APISecret = util.Decrypt(p.GA4APISecret)
	p.MetaPermanentToken = util.Decrypt(p.MetaPermanentToken)

	return &p, nil
}
