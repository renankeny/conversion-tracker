package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type APIKeyRepo struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
}

func NewAPIKeyRepo(pool *pgxpool.Pool, rdb *redis.Client) *APIKeyRepo {
	return &APIKeyRepo{pool: pool, rdb: rdb}
}

// Create gera um segredo novo cifrado em AES-256-GCM, salva permissões e insere no banco.
func (r *APIKeyRepo) Create(ctx context.Context, accountID int64, name string, scopes []string, readOnly bool) (*domain.APIKey, string, error) {
	if accountID <= 0 {
		return nil, "", errors.New("account_id obrigatório")
	}
	if name == "" {
		return nil, "", errors.New("name obrigatório")
	}
	normScopes, ok := domain.NormalizeAPIKeyScopes(scopes, true)
	if !ok {
		return nil, "", errors.New("selecione ao menos uma permissão para a API Key")
	}

	plaintext, prefix, hash, encrypted, err := generateSecretMaterial()
	if err != nil {
		return nil, "", err
	}

	k := &domain.APIKey{
		AccountID:    accountID,
		Name:         name,
		KeyPrefix:    prefix,
		KeyHash:      hash,
		KeyEncrypted: encrypted,
		Plaintext:    plaintext,
		Scopes:       normScopes,
		ReadOnly:     readOnly,
	}
	const q = `INSERT INTO api_keys (account_id, name, key_prefix, key_hash, key_encrypted, scopes, read_only)
	           VALUES ($1, $2, $3, $4, $5, $6, $7)
	           RETURNING id, created_at, updated_at`
	if err := r.pool.QueryRow(ctx, q, k.AccountID, k.Name, k.KeyPrefix, k.KeyHash, k.KeyEncrypted, k.Scopes, k.ReadOnly).
		Scan(&k.ID, &k.CreatedAt, &k.UpdatedAt); err != nil {
		return nil, "", err
	}
	return k, plaintext, nil
}

// Update altera nome, escopos e modo somente leitura de uma key ativa, invalidando o cache Redis.
func (r *APIKeyRepo) Update(ctx context.Context, id int64, name string, scopes []string, readOnly bool) (*domain.APIKey, error) {
	existing, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !existing.IsActive() {
		return nil, errors.New("não é possível editar uma API key revogada")
	}
	normScopes, ok := domain.NormalizeAPIKeyScopes(scopes, false)
	if !ok {
		return nil, errors.New("selecione ao menos uma permissão para a API Key")
	}

	const q = `UPDATE api_keys
	           SET name = $2, scopes = $3, read_only = $4, updated_at = NOW()
	           WHERE id = $1
	           RETURNING ` + apiKeyColumns
	updated, err := scanAPIKey(r.pool.QueryRow(ctx, q, id, name, normScopes, readOnly))
	if err != nil {
		return nil, err
	}
	if r.rdb != nil {
		r.rdb.Del(ctx, "api_key_hash:"+existing.KeyHash)
	}
	updated.Plaintext = RevealPlaintext(updated)
	return updated, nil
}

// RegenerateSecret gera um novo segredo `trk_...` preservando nome, empresa e permissões.
func (r *APIKeyRepo) RegenerateSecret(ctx context.Context, id int64) (*domain.APIKey, string, error) {
	existing, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if !existing.IsActive() {
		return nil, "", errors.New("não é possível regerar uma API key revogada")
	}
	plaintext, prefix, hash, encrypted, err := generateSecretMaterial()
	if err != nil {
		return nil, "", err
	}
	const q = `UPDATE api_keys
	           SET key_prefix = $2, key_hash = $3, key_encrypted = $4, updated_at = NOW()
	           WHERE id = $1
	           RETURNING ` + apiKeyColumns
	updated, err := scanAPIKey(r.pool.QueryRow(ctx, q, id, prefix, hash, encrypted))
	if err != nil {
		return nil, "", err
	}
	if r.rdb != nil {
		r.rdb.Del(ctx, "api_key_hash:"+existing.KeyHash)
	}
	updated.Plaintext = plaintext
	return updated, plaintext, nil
}

func (r *APIKeyRepo) ListByAccount(ctx context.Context, accountID int64) ([]domain.APIKey, error) {
	args := []any{}
	where := ""
	if accountID != 0 {
		where = "WHERE account_id = $1"
		args = append(args, accountID)
	}
	q := `SELECT ` + apiKeyColumns + ` FROM api_keys ` + where + ` ORDER BY id DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		k.Plaintext = RevealPlaintext(k)
		out = append(out, *k)
	}
	return out, rows.Err()
}

func (r *APIKeyRepo) FindActiveByPlaintext(ctx context.Context, plaintext string) (*domain.APIKey, error) {
	hash := hashKey(plaintext)
	cacheKey := "api_key_hash:" + hash

	if r.rdb != nil {
		if val, err := r.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var k domain.APIKey
			if err := json.Unmarshal([]byte(val), &k); err == nil {
				return &k, nil
			}
		}
	}

	q := `SELECT ` + apiKeyColumns + ` FROM api_keys WHERE key_hash = $1 AND revoked_at IS NULL LIMIT 1`
	k, err := scanAPIKey(r.pool.QueryRow(ctx, q, hash))
	if err != nil {
		return nil, err
	}

	if r.rdb != nil {
		if b, err := json.Marshal(k); err == nil {
			r.rdb.Set(ctx, cacheKey, b, 5*time.Minute)
		}
	}
	return k, nil
}

func (r *APIKeyRepo) FindByID(ctx context.Context, id int64) (*domain.APIKey, error) {
	q := `SELECT ` + apiKeyColumns + ` FROM api_keys WHERE id = $1`
	return scanAPIKey(r.pool.QueryRow(ctx, q, id))
}

func (r *APIKeyRepo) Revoke(ctx context.Context, id int64) error {
	k, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE api_keys SET revoked_at = COALESCE(revoked_at, NOW()), updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if r.rdb != nil {
		r.rdb.Del(ctx, "api_key_hash:"+k.KeyHash)
	}
	return nil
}

func (r *APIKeyRepo) TouchLastUsed(ctx context.Context, id int64) error {
	if r.rdb != nil {
		debounceKey := fmt.Sprintf("api_key_touch:%d", id)
		ok, err := r.rdb.SetNX(ctx, debounceKey, "1", time.Hour).Result()
		if err == nil && !ok {
			return nil
		}
	}
	_, err := r.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}
