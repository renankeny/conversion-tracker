package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type WhatsAppRepo struct {
	db *pgxpool.Pool
}

func NewWhatsAppRepo(db *pgxpool.Pool) *WhatsAppRepo {
	return &WhatsAppRepo{db: db}
}

// =====================================================================
// Servers (Multi-Tenant por Conta)
// =====================================================================

// UpsertAccountServer cria ou atualiza o servidor Evolution Go ativo da conta.
// Se s.APIKey vier vazio em uma atualização, preserva a api_key criptografada anterior.
func (r *WhatsAppRepo) UpsertAccountServer(ctx context.Context, s *domain.WhatsAppServer) error {
	cleanURL := strings.TrimRight(strings.TrimSpace(s.URL), "/")
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = "Evolution Go"
	}

	existing, err := r.GetAccountServer(ctx, s.AccountID)
	if err != nil {
		return err
	}

	plainKey := strings.TrimSpace(s.APIKey)
	if existing != nil {
		if plainKey == "" {
			plainKey = existing.APIKey
		}
		encKey := util.Encrypt(plainKey)
		err = r.db.QueryRow(ctx, `
			UPDATE whatsapp_servers
			SET name = $1, url = $2, api_key = $3, is_active = TRUE, updated_at = NOW()
			WHERE id = $4 AND account_id = $5
			RETURNING id, created_at, updated_at
		`, name, cleanURL, encKey, existing.ID, s.AccountID).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
		s.Name = name
		s.URL = cleanURL
		s.APIKey = plainKey
		s.HasAPIKey = plainKey != ""
		s.IsActive = true
		return err
	}

	encKey := util.Encrypt(plainKey)
	err = r.db.QueryRow(ctx, `
		INSERT INTO whatsapp_servers (account_id, name, url, api_key, is_active)
		VALUES ($1, $2, $3, $4, TRUE)
		RETURNING id, created_at, updated_at
	`, s.AccountID, name, cleanURL, encKey).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	s.Name = name
	s.URL = cleanURL
	s.APIKey = plainKey
	s.HasAPIKey = plainKey != ""
	s.IsActive = true
	return err
}

func (r *WhatsAppRepo) GetAccountServer(ctx context.Context, accountID int64) (*domain.WhatsAppServer, error) {
	var s domain.WhatsAppServer
	var encKey string
	err := r.db.QueryRow(ctx, `
		SELECT id, account_id, name, url, api_key, is_active, created_at, updated_at
		FROM whatsapp_servers
		WHERE account_id = $1 AND is_active = TRUE
		ORDER BY id DESC
		LIMIT 1
	`, accountID).Scan(&s.ID, &s.AccountID, &s.Name, &s.URL, &encKey, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.APIKey = util.Decrypt(encKey)
	s.HasAPIKey = s.APIKey != ""
	return &s, nil
}

func (r *WhatsAppRepo) DeleteAccountServer(ctx context.Context, accountID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM whatsapp_servers WHERE account_id = $1`, accountID)
	return err
}

// =====================================================================
// LID <-> Phone Mappings (Resolução determinística @lid)
// =====================================================================

func (r *WhatsAppRepo) UpsertLIDMapping(ctx context.Context, accountID int64, lidJid, phoneJid string) error {
	lidJid = strings.TrimSpace(lidJid)
	phoneJid = strings.TrimSpace(phoneJid)
	if accountID <= 0 || lidJid == "" || phoneJid == "" {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO whatsapp_lid_mappings (account_id, lid_jid, phone_jid, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (account_id, lid_jid) DO UPDATE SET
			phone_jid = EXCLUDED.phone_jid,
			updated_at = NOW()
	`, accountID, lidJid, phoneJid)
	return err
}

func (r *WhatsAppRepo) ResolvePhoneByLID(ctx context.Context, accountID int64, lidJid string) string {
	lidJid = strings.TrimSpace(lidJid)
	if accountID <= 0 || lidJid == "" {
		return ""
	}
	var phoneJid string
	_ = r.db.QueryRow(ctx, `
		SELECT phone_jid FROM whatsapp_lid_mappings
		WHERE account_id = $1 AND lid_jid = $2
	`, accountID, lidJid).Scan(&phoneJid)
	return phoneJid
}

func (r *WhatsAppRepo) ResolveLIDByPhone(ctx context.Context, accountID int64, phoneJid string) string {
	phoneJid = strings.TrimSpace(phoneJid)
	if accountID <= 0 || phoneJid == "" {
		return ""
	}
	var lidJid string
	_ = r.db.QueryRow(ctx, `
		SELECT lid_jid FROM whatsapp_lid_mappings
		WHERE account_id = $1 AND phone_jid = $2
		ORDER BY updated_at DESC
		LIMIT 1
	`, accountID, phoneJid).Scan(&lidJid)
	return lidJid
}
