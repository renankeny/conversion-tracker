package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// URLMapping representa uma linha de mapeamento black -> white.
type URLMapping struct {
	ID        int64     `json:"id"`
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type URLMappingRepo struct {
	pool *pgxpool.Pool
}

func NewURLMappingRepo(pool *pgxpool.Pool) *URLMappingRepo {
	return &URLMappingRepo{pool: pool}
}

const urlMappingCols = `id, source, target, enabled, created_at, updated_at`

// List devolve todas as linhas (incluindo desabilitadas) ordenadas por id desc.
func (r *URLMappingRepo) List(ctx context.Context) ([]URLMapping, error) {
	q := `SELECT ` + urlMappingCols + ` FROM url_mappings ORDER BY id DESC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []URLMapping
	for rows.Next() {
		var m URLMapping
		if err := rows.Scan(&m.ID, &m.Source, &m.Target, &m.Enabled, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ActiveAsMap devolve um snapshot dos mapeamentos habilitados pra alimentar o util.URLMapper.
func (r *URLMappingRepo) ActiveAsMap(ctx context.Context) (map[string]string, error) {
	q := `SELECT source, target FROM url_mappings WHERE enabled = TRUE`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var s, t string
		if err := rows.Scan(&s, &t); err != nil {
			return nil, err
		}
		out[s] = t
	}
	return out, rows.Err()
}

// Create insere um novo mapeamento.
func (r *URLMappingRepo) Create(ctx context.Context, m *URLMapping) error {
	const q = `INSERT INTO url_mappings (source, target, enabled)
	           VALUES ($1, $2, $3)
	           RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, q, m.Source, m.Target, m.Enabled).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

// Update atualiza um mapeamento existente.
func (r *URLMappingRepo) Update(ctx context.Context, m *URLMapping) error {
	const q = `UPDATE url_mappings
	           SET source = $1, target = $2, enabled = $3, updated_at = NOW()
	           WHERE id = $4
	           RETURNING updated_at`
	err := r.pool.QueryRow(ctx, q, m.Source, m.Target, m.Enabled, m.ID).Scan(&m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// Delete remove um mapeamento. Operação não destrutiva de dados (apenas remove a regra
// de transformação de URL); nada de tracking é apagado.
func (r *URLMappingRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM url_mappings WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
