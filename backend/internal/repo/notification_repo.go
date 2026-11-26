package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type NotificationRepo struct {
	pool *pgxpool.Pool
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool}
}

type VAPIDRecord struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

func (r *NotificationRepo) GetOrCreateVAPID(ctx context.Context, gen func() (pub, priv, sub string, err error)) (*VAPIDRecord, error) {
	if existing, err := r.loadVAPID(ctx); err == nil && existing != nil {
		return existing, nil
	}
	pub, privPlain, subject, err := gen()
	if err != nil {
		return nil, err
	}
	privEnc, err := util.Encrypt(privPlain)
	if err != nil {
		return nil, fmt.Errorf("encrypt vapid private key: %w", err)
	}
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO push_vapid_config (id, public_key, private_key_encrypted, subject)
		VALUES (1, $1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, pub, privEnc, subject)
	return r.loadVAPID(ctx)
}

func (r *NotificationRepo) loadVAPID(ctx context.Context) (*VAPIDRecord, error) {
	var pub, privEnc, subject string
	err := r.pool.QueryRow(ctx, `
		SELECT public_key, private_key_encrypted, subject
		FROM push_vapid_config WHERE id = 1
	`).Scan(&pub, &privEnc, &subject)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select push_vapid_config: %w", err)
	}
	return &VAPIDRecord{
		PublicKey:  pub,
		PrivateKey: util.Decrypt(privEnc),
		Subject:    subject,
	}, nil
}

func (r *NotificationRepo) GetPreferences(ctx context.Context, userID int64) (domain.NotificationPreferences, error) {
	def := domain.DefaultNotificationPreferences(userID)
	var p domain.NotificationPreferences
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, account_scope_mode, COALESCE(enabled_account_ids, '{}'),
			whatsapp_connection, lead_created, ai_lead_qualified, sale_approved,
			COALESCE(sale_refunded, TRUE),
			meta_account_lifecycle, google_ads_account_lifecycle, pixel_lifecycle,
			event_dispatch_error, updated_at
		FROM user_notification_preferences
		WHERE user_id = $1
	`, userID).Scan(
		&p.UserID, &p.AccountScopeMode, &p.EnabledAccountIDs,
		&p.WhatsAppConnection, &p.LeadCreated, &p.AILeadQualified, &p.SaleApproved,
		&p.SaleRefunded,
		&p.MetaAccountLifecycle, &p.GoogleAdsAccountLifecycle, &p.PixelLifecycle,
		&p.EventDispatchError, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return def, nil
	}
	if err != nil {
		return def, fmt.Errorf("get notification preferences: %w", err)
	}
	p.AccountScopeMode = domain.NormalizeAccountScopeMode(p.AccountScopeMode)
	if p.EnabledAccountIDs == nil {
		p.EnabledAccountIDs = []int64{}
	}
	return p, nil
}

func (r *NotificationRepo) UpsertPreferences(ctx context.Context, p domain.NotificationPreferences) (domain.NotificationPreferences, error) {
	p.AccountScopeMode = domain.NormalizeAccountScopeMode(p.AccountScopeMode)
	if p.EnabledAccountIDs == nil {
		p.EnabledAccountIDs = []int64{}
	}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_notification_preferences (
			user_id, account_scope_mode, enabled_account_ids,
			whatsapp_connection, lead_created, ai_lead_qualified, sale_approved, sale_refunded,
			meta_account_lifecycle, google_ads_account_lifecycle, pixel_lifecycle,
			event_dispatch_error, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			account_scope_mode           = EXCLUDED.account_scope_mode,
			enabled_account_ids          = EXCLUDED.enabled_account_ids,
			whatsapp_connection          = EXCLUDED.whatsapp_connection,
			lead_created                 = EXCLUDED.lead_created,
			ai_lead_qualified            = EXCLUDED.ai_lead_qualified,
			sale_approved                = EXCLUDED.sale_approved,
			sale_refunded                = EXCLUDED.sale_refunded,
			meta_account_lifecycle       = EXCLUDED.meta_account_lifecycle,
			google_ads_account_lifecycle = EXCLUDED.google_ads_account_lifecycle,
			pixel_lifecycle              = EXCLUDED.pixel_lifecycle,
			event_dispatch_error         = EXCLUDED.event_dispatch_error,
			updated_at                   = NOW()
		RETURNING updated_at
	`,
		p.UserID, p.AccountScopeMode, p.EnabledAccountIDs,
		p.WhatsAppConnection, p.LeadCreated, p.AILeadQualified, p.SaleApproved, p.SaleRefunded,
		p.MetaAccountLifecycle, p.GoogleAdsAccountLifecycle, p.PixelLifecycle,
		p.EventDispatchError,
	).Scan(&p.UpdatedAt)
	if err != nil {
		return p, fmt.Errorf("upsert notification preferences: %w", err)
	}
	return p, nil
}

// InsertWithMax10 insere a notificação e remove imediatamente qualquer registro
// antigo além das 10 notificações mais recentes daquele usuário.
func (r *NotificationRepo) InsertWithMax10(ctx context.Context, n *domain.UserNotification) error {
	if n.ActionURL == "" {
		n.ActionURL = "/"
	}
	err := r.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO user_notifications (
				user_id, account_id, account_name, category, title, body, action_url, is_read
			) VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE)
			RETURNING id, created_at
		),
		pruned AS (
			DELETE FROM user_notifications
			WHERE user_id = $1
			  AND id NOT IN (
				SELECT id FROM (
					SELECT id FROM user_notifications WHERE user_id = $1
					UNION ALL
					SELECT id FROM inserted
				) recent
				ORDER BY id DESC
				LIMIT 10
			  )
		)
		SELECT id, created_at FROM inserted
	`,
		n.UserID, n.AccountID, n.AccountName, n.Category, n.Title, n.Body, n.ActionURL,
	).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert notification with cap 10: %w", err)
	}
	return nil
}

func (r *NotificationRepo) ListRecentByUser(ctx context.Context, userID int64) ([]domain.UserNotification, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, account_id, account_name, category, title, body, action_url, is_read, created_at
		FROM user_notifications
		WHERE user_id = $1
		ORDER BY id DESC
		LIMIT 10
	`, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("list user_notifications: %w", err)
	}
	defer rows.Close()

	out := make([]domain.UserNotification, 0, 10)
	unread := 0
	for rows.Next() {
		var item domain.UserNotification
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.AccountID, &item.AccountName,
			&item.Category, &item.Title, &item.Body, &item.ActionURL,
			&item.IsRead, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user_notifications: %w", err)
		}
		if !item.IsRead {
			unread++
		}
		out = append(out, item)
	}
	return out, unread, rows.Err()
}
