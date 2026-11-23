package repo

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const apiKeyColumns = `id, account_id, name, key_prefix, key_hash, key_encrypted,
	scopes, read_only, last_used_at, revoked_at, created_at, updated_at`

const PlaintextPrefix = "trk_"

func generateSecretMaterial() (plaintext, prefix, hash, encrypted string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", "", fmt.Errorf("rand: %w", err)
	}
	plaintext = PlaintextPrefix + base64.RawURLEncoding.EncodeToString(buf)
	hash = hashKey(plaintext)
	prefix = plaintext
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	enc, encErr := util.Encrypt(plaintext)
	if encErr == nil {
		encrypted = enc
	}
	return plaintext, prefix, hash, encrypted, nil
}

// RevealPlaintext decifra `key_encrypted` quando disponível para exibição autorizada no painel.
func RevealPlaintext(k *domain.APIKey) string {
	if k == nil || k.KeyEncrypted == "" {
		return ""
	}
	dec := util.Decrypt(k.KeyEncrypted)
	if strings.HasPrefix(dec, PlaintextPrefix) {
		return dec
	}
	return ""
}

func hashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func scanAPIKey(row rowScanner) (*domain.APIKey, error) {
	var k domain.APIKey
	var lastUsed, revoked *time.Time
	err := row.Scan(
		&k.ID, &k.AccountID, &k.Name, &k.KeyPrefix, &k.KeyHash, &k.KeyEncrypted,
		&k.Scopes, &k.ReadOnly, &lastUsed, &revoked, &k.CreatedAt, &k.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(k.Scopes) == 0 {
		k.Scopes = []string{domain.APIScopeAll}
	}
	k.LastUsedAt = lastUsed
	k.RevokedAt = revoked
	return &k, nil
}
