package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const maskedKeyMarker = "••••"

func maskOpenAIKey(plain string) string {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return ""
	}
	if len(plain) <= 8 {
		return "sk-" + maskedKeyMarker
	}
	return "sk-" + maskedKeyMarker + plain[len(plain)-4:]
}

func isMaskedOpenAIKey(val string) bool {
	return strings.Contains(val, maskedKeyMarker)
}

func sanitizePrefsForClient(raw []byte) map[string]any {
	prefs := make(map[string]any)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &prefs)
	}
	delete(prefs, "has_openai_key")
	rawKey, ok := prefs["openai_api_key"].(string)
	if !ok || strings.TrimSpace(rawKey) == "" {
		prefs["has_openai_key"] = false
		delete(prefs, "openai_api_key")
		return prefs
	}
	plain := strings.TrimSpace(util.Decrypt(strings.TrimSpace(rawKey)))
	if plain == "" {
		prefs["has_openai_key"] = false
		delete(prefs, "openai_api_key")
		return prefs
	}
	prefs["has_openai_key"] = true
	prefs["openai_api_key"] = maskOpenAIKey(plain)
	return prefs
}

func preparePrefsForStorage(ctx context.Context, pool *pgxpool.Pool, userID int64, incoming map[string]any) ([]byte, map[string]any, error) {
	existing := make(map[string]any)
	var existingBytes []byte
	_ = pool.QueryRow(ctx, `SELECT preferences FROM admin_users WHERE id = $1`, userID).Scan(&existingBytes)
	if len(existingBytes) > 0 {
		_ = json.Unmarshal(existingBytes, &existing)
	}

	merged := make(map[string]any, len(existing)+len(incoming))
	for k, v := range existing {
		if k != "has_openai_key" {
			merged[k] = v
		}
	}
	for k, v := range incoming {
		if k == "has_openai_key" || k == "openai_api_key" {
			continue
		}
		merged[k] = v
	}

	if rawVal, provided := incoming["openai_api_key"]; provided {
		keyStr, _ := rawVal.(string)
		keyStr = strings.TrimSpace(keyStr)
		if keyStr == "" {
			delete(merged, "openai_api_key")
		} else if !isMaskedOpenAIKey(keyStr) {
			merged["openai_api_key"] = util.Encrypt(keyStr)
		}
	}

	storedBytes, err := json.Marshal(merged)
	if err != nil {
		return nil, nil, err
	}
	return storedBytes, sanitizePrefsForClient(storedBytes), nil
}

func loadDecryptedOpenAIKey(ctx context.Context, pool *pgxpool.Pool, userID int64) string {
	if userID <= 0 || pool == nil {
		return ""
	}
	var raw []byte
	if err := pool.QueryRow(ctx, `SELECT preferences FROM admin_users WHERE id = $1`, userID).Scan(&raw); err != nil || len(raw) == 0 {
		return ""
	}
	var prefs map[string]any
	if err := json.Unmarshal(raw, &prefs); err != nil {
		return ""
	}
	stored, ok := prefs["openai_api_key"].(string)
	if !ok || strings.TrimSpace(stored) == "" {
		return ""
	}
	return strings.TrimSpace(util.Decrypt(strings.TrimSpace(stored)))
}
