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
// Servers (Configuração Global por provedor direto: Evolution Go e WAHA)
// =====================================================================

func defaultServerNameForProvider(provider string) string {
	if domain.NormalizeWhatsAppServerProvider(provider) == domain.WhatsAppProviderWAHA {
		return "WAHA API"
	}
	return "Evolution Go"
}

// UpsertGlobalServer cria ou atualiza o servidor global de Conexão Direta (Evolution Go ou WAHA).
func (r *WhatsAppRepo) UpsertGlobalServer(ctx context.Context, s *domain.WhatsAppServer) error {
	provider := domain.NormalizeWhatsAppServerProvider(s.Provider)
	cleanURL := strings.TrimRight(strings.TrimSpace(s.URL), "/")
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = defaultServerNameForProvider(provider)
	}

	existing, err := r.GetGlobalServerByProvider(ctx, provider)
	if err != nil {
		return err
	}

	plainKey := strings.TrimSpace(s.APIKey)
	if existing != nil {
		if plainKey == "" {
			plainKey = existing.APIKey
		}
		encKey, err := util.Encrypt(plainKey)
		if err != nil {
			return err
		}
		if _, err := r.db.Exec(ctx, `
			UPDATE whatsapp_servers
			SET name = $1, url = $2, api_key = $3, updated_at = NOW()
			WHERE is_active = TRUE AND provider = $4
		`, name, cleanURL, encKey, provider); err != nil {
			return err
		}
		err = r.db.QueryRow(ctx, `
			SELECT id, account_id, created_at, updated_at
			FROM whatsapp_servers
			WHERE id = $1
		`, existing.ID).Scan(&s.ID, &s.AccountID, &s.CreatedAt, &s.UpdatedAt)
		s.Provider = provider
		s.Name = name
		s.URL = cleanURL
		s.APIKey = plainKey
		s.HasAPIKey = plainKey != ""
		s.IsActive = true
		return err
	}

	encKey, err := util.Encrypt(plainKey)
	if err != nil {
		return err
	}
	accID := s.AccountID
	if accID <= 0 {
		_ = r.db.QueryRow(ctx, `SELECT id FROM accounts ORDER BY id ASC LIMIT 1`).Scan(&accID)
		if accID <= 0 {
			accID = 1
		}
	}
	err = r.db.QueryRow(ctx, `
		INSERT INTO whatsapp_servers (account_id, provider, name, url, api_key, is_active)
		VALUES ($1, $2, $3, $4, $5, TRUE)
		RETURNING id, account_id, created_at, updated_at
	`, accID, provider, name, cleanURL, encKey).Scan(&s.ID, &s.AccountID, &s.CreatedAt, &s.UpdatedAt)
	s.Provider = provider
	s.Name = name
	s.URL = cleanURL
	s.APIKey = plainKey
	s.HasAPIKey = plainKey != ""
	s.IsActive = true
	return err
}

func (r *WhatsAppRepo) UpsertAccountServer(ctx context.Context, s *domain.WhatsAppServer) error {
	return r.UpsertGlobalServer(ctx, s)
}

func (r *WhatsAppRepo) GetGlobalServerByProvider(ctx context.Context, provider string) (*domain.WhatsAppServer, error) {
	prov := domain.NormalizeWhatsAppServerProvider(provider)
	var s domain.WhatsAppServer
	var encKey string
	err := r.db.QueryRow(ctx, `
		SELECT id, account_id, COALESCE(provider, 'evolution_go'), name, url, api_key, is_active, created_at, updated_at
		FROM whatsapp_servers
		WHERE is_active = TRUE AND COALESCE(provider, 'evolution_go') = $1
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`, prov).Scan(&s.ID, &s.AccountID, &s.Provider, &s.Name, &s.URL, &encKey, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Provider = domain.NormalizeWhatsAppServerProvider(s.Provider)
	s.APIKey = util.Decrypt(encKey)
	s.HasAPIKey = s.APIKey != ""
	return &s, nil
}

// GetGlobalServer retorna o servidor global Evolution Go (mantém retrocompatibilidade).
func (r *WhatsAppRepo) GetGlobalServer(ctx context.Context) (*domain.WhatsAppServer, error) {
	return r.GetGlobalServerByProvider(ctx, domain.WhatsAppProviderEvolutionGo)
}

func (r *WhatsAppRepo) GetAccountServer(ctx context.Context, _ int64) (*domain.WhatsAppServer, error) {
	return r.GetGlobalServer(ctx)
}

func (r *WhatsAppRepo) CountInstancesByServerProvider(ctx context.Context, provider string) (int, error) {
	prov := domain.NormalizeWhatsAppServerProvider(provider)
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM whatsapp_instances
		WHERE server_id IS NOT NULL AND COALESCE(provider, 'evolution_go') = $1
	`, prov).Scan(&count)
	return count, err
}

func (r *WhatsAppRepo) CountEvolutionInstancesGlobal(ctx context.Context) (int, error) {
	return r.CountInstancesByServerProvider(ctx, domain.WhatsAppProviderEvolutionGo)
}

func (r *WhatsAppRepo) DeleteGlobalServerByProvider(ctx context.Context, provider string) error {
	prov := domain.NormalizeWhatsAppServerProvider(provider)
	_, err := r.db.Exec(ctx, `DELETE FROM whatsapp_servers WHERE COALESCE(provider, 'evolution_go') = $1`, prov)
	return err
}

func (r *WhatsAppRepo) DeleteGlobalServer(ctx context.Context) error {
	return r.DeleteGlobalServerByProvider(ctx, domain.WhatsAppProviderEvolutionGo)
}

func (r *WhatsAppRepo) DeleteAccountServer(ctx context.Context, _ int64) error {
	return r.DeleteGlobalServer(ctx)
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
