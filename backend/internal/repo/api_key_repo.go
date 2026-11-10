package repo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

const apiKeyColumns = `id, account_id, name, key_prefix, key_hash,
	last_used_at, revoked_at, created_at, updated_at`

// PlaintextPrefix é o prefixo obrigatório do segredo. Útil pra reconhecer
// rapidamente uma key vazada em logs/grep.
const PlaintextPrefix = "trk_"

// Create gera um segredo novo, calcula hash e insere.
//
// O plaintext é retornado APENAS aqui — o caller deve mostrá-lo ao admin
// imediatamente; não é armazenado em parte alguma e não pode ser recuperado
// depois.
func (r *APIKeyRepo) Create(ctx context.Context, accountID int64, name string) (*domain.APIKey, string, error) {
	if accountID <= 0 {
		return nil, "", errors.New("account_id obrigatório")
	}
	if name == "" {
		return nil, "", errors.New("name obrigatório")
	}

	// 32 bytes random -> base64url (~43 chars sem padding)
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, "", fmt.Errorf("rand: %w", err)
	}
	body := base64.RawURLEncoding.EncodeToString(buf)
	plaintext := PlaintextPrefix + body
	hash := hashKey(plaintext)
	prefix := plaintext
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}

	k := &domain.APIKey{
		AccountID: accountID,
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   hash,
	}
	const q = `INSERT INTO api_keys (account_id, name, key_prefix, key_hash)
	           VALUES ($1, $2, $3, $4)
	           RETURNING id, created_at, updated_at`
	if err := r.pool.QueryRow(ctx, q, k.AccountID, k.Name, k.KeyPrefix, k.KeyHash).
		Scan(&k.ID, &k.CreatedAt, &k.UpdatedAt); err != nil {
		return nil, "", err
	}
	return k, plaintext, nil
}

// ListByAccount devolve todas as keys (incluindo revogadas) de uma conta.
// Se accountID == 0 devolve todas (uso owner).
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
		out = append(out, *k)
	}
	return out, rows.Err()
}

// FindActiveByPlaintext lookup por SHA-256 do plaintext informado.
// Retorna ErrNotFound se o hash não casar OU a key estiver revogada.
func (r *APIKeyRepo) FindActiveByPlaintext(ctx context.Context, plaintext string) (*domain.APIKey, error) {
	hash := hashKey(plaintext)
	cacheKey := "api_key_hash:" + hash

	// 1. Tentar Redis
	if r.rdb != nil {
		if val, err := r.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var k domain.APIKey
			if err := json.Unmarshal([]byte(val), &k); err == nil {
				return &k, nil
			}
		}
	}

	q := `SELECT ` + apiKeyColumns + ` FROM api_keys
	      WHERE key_hash = $1 AND revoked_at IS NULL LIMIT 1`
	k, err := scanAPIKey(r.pool.QueryRow(ctx, q, hash))
	if err != nil {
		return nil, err
	}

	// 2. Salvar no Redis
	if r.rdb != nil {
		if b, err := json.Marshal(k); err == nil {
			r.rdb.Set(ctx, cacheKey, b, 5*time.Minute)
		}
	}

	return k, nil
}

// FindByID localiza uma key (mesmo revogada) — usada para validações de scope.
func (r *APIKeyRepo) FindByID(ctx context.Context, id int64) (*domain.APIKey, error) {
	q := `SELECT ` + apiKeyColumns + ` FROM api_keys WHERE id = $1`
	return scanAPIKey(r.pool.QueryRow(ctx, q, id))
}

// Revoke marca a key como revogada (idempotente — preserva timestamp original).
func (r *APIKeyRepo) Revoke(ctx context.Context, id int64) error {
	k, err := r.FindByID(ctx, id)
	if err != nil {
		return err // ErrNotFound
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at = COALESCE(revoked_at, NOW()), updated_at = NOW()
		 WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Invalida o cache instantaneamente
	if r.rdb != nil {
		r.rdb.Del(ctx, "api_key_hash:"+k.KeyHash)
	}

	return nil
}

// TouchLastUsed atualiza last_used_at de forma best-effort. Erros são
// ignorados pelo caller (não queremos derrubar requests por causa disso).
func (r *APIKeyRepo) TouchLastUsed(ctx context.Context, id int64) error {
	if r.rdb != nil {
		debounceKey := fmt.Sprintf("api_key_touch:%d", id)
		ok, err := r.rdb.SetNX(ctx, debounceKey, "1", time.Hour).Result()
		if err == nil && !ok {
			// Já atualizado recentemente, pula o update no PostgreSQL
			return nil
		}
	}

	_, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

func hashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func scanAPIKey(row rowScanner) (*domain.APIKey, error) {
	var k domain.APIKey
	var lastUsed, revoked *time.Time
	err := row.Scan(
		&k.ID, &k.AccountID, &k.Name, &k.KeyPrefix, &k.KeyHash,
		&lastUsed, &revoked, &k.CreatedAt, &k.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	k.LastUsedAt = lastUsed
	k.RevokedAt = revoked
	return &k, nil
}
