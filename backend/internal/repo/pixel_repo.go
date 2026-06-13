package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

var ErrNotFound = errors.New("not found")

type PixelRepo struct {
	pool *pgxpool.Pool
}

func NewPixelRepo(pool *pgxpool.Pool) *PixelRepo {
	return &PixelRepo{pool: pool}
}

const pixelColumns = `id, name, label, pixel_ids, access_tokens, test_codes,
		domains, tracker_config, enabled, tracking_host, created_at, updated_at`

func (r *PixelRepo) FindByName(ctx context.Context, name string) (*domain.Pixel, error) {
	q := `SELECT ` + pixelColumns + ` FROM pixels WHERE name = $1`
	return scanPixel(r.pool.QueryRow(ctx, q, name))
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

// ActiveDomains devolve a união de todos os domínios cadastrados em pixels
// habilitados. Usado pra alimentar o cache do CORS handler.
func (r *PixelRepo) ActiveDomains(ctx context.Context) ([]string, error) {
	const q = `SELECT DISTINCT unnest(domains) FROM pixels WHERE enabled = TRUE`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		if d != "" {
			out = append(out, d)
		}
	}
	return out, rows.Err()
}

func (r *PixelRepo) Create(ctx context.Context, p *domain.Pixel) error {
	cfgJSON, err := json.Marshal(p.TrackerConfig)
	if err != nil {
		return fmt.Errorf("marshal tracker_config: %w", err)
	}
	const q = `INSERT INTO pixels (name, label, pixel_ids, access_tokens, test_codes,
	                               domains, tracker_config, enabled, tracking_host)
	           VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	           RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, q,
		p.Name, p.Label, p.PixelIDs, p.AccessTokens, p.TestCodes,
		p.Domains, cfgJSON, p.Enabled, p.TrackingHost,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *PixelRepo) Update(ctx context.Context, p *domain.Pixel) error {
	cfgJSON, err := json.Marshal(p.TrackerConfig)
	if err != nil {
		return fmt.Errorf("marshal tracker_config: %w", err)
	}
	const q = `UPDATE pixels
	           SET name=$1, label=$2, pixel_ids=$3, access_tokens=$4, test_codes=$5,
	               domains=$6, tracker_config=$7, enabled=$8, tracking_host=$9, updated_at=NOW()
	           WHERE id=$10
	           RETURNING updated_at`
	return r.pool.QueryRow(ctx, q,
		p.Name, p.Label, p.PixelIDs, p.AccessTokens, p.TestCodes,
		p.Domains, cfgJSON, p.Enabled, p.TrackingHost, p.ID,
	).Scan(&p.UpdatedAt)
}

func (r *PixelRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pixels WHERE id = $1`, id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPixel(row rowScanner) (*domain.Pixel, error) {
	var p domain.Pixel
	var cfgJSON []byte
	err := row.Scan(
		&p.ID, &p.Name, &p.Label, &p.PixelIDs, &p.AccessTokens, &p.TestCodes,
		&p.Domains, &cfgJSON, &p.Enabled, &p.TrackingHost, &p.CreatedAt, &p.UpdatedAt,
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
	return &p, nil
}
