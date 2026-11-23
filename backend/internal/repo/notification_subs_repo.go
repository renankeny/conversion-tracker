package repo

import (
	"context"
	"fmt"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type NotificationRecipient struct {
	UserID      int64
	AccountName string
	Prefs       domain.NotificationPreferences
}

func (r *NotificationRepo) MarkRead(ctx context.Context, userID, notifID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE user_notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2
	`, notifID, userID)
	return err
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE user_notifications SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}

func (r *NotificationRepo) UpsertSubscription(ctx context.Context, sub *domain.PushSubscription) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO user_push_subscriptions (
			user_id, endpoint, p256dh, auth_secret, device_label, user_agent, last_seen_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (endpoint) DO UPDATE SET
			user_id      = EXCLUDED.user_id,
			p256dh       = EXCLUDED.p256dh,
			auth_secret  = EXCLUDED.auth_secret,
			device_label = EXCLUDED.device_label,
			user_agent   = EXCLUDED.user_agent,
			last_seen_at = NOW()
		RETURNING id, created_at, last_seen_at
	`,
		sub.UserID, sub.Endpoint, sub.P256dh, sub.AuthSecret, sub.DeviceLabel, sub.UserAgent,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.LastSeenAt)
}

func (r *NotificationRepo) ListSubscriptionsByUser(ctx context.Context, userID int64) ([]domain.PushSubscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, endpoint, p256dh, auth_secret, device_label, user_agent, created_at, last_seen_at
		FROM user_push_subscriptions
		WHERE user_id = $1
		ORDER BY last_seen_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user_push_subscriptions: %w", err)
	}
	defer rows.Close()

	out := make([]domain.PushSubscription, 0, 4)
	for rows.Next() {
		var s domain.PushSubscription
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.AuthSecret,
			&s.DeviceLabel, &s.UserAgent, &s.CreatedAt, &s.LastSeenAt,
		); err != nil {
			return nil, fmt.Errorf("scan user_push_subscriptions: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *NotificationRepo) DeleteSubscription(ctx context.Context, userID, subID int64) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM user_push_subscriptions WHERE id = $1 AND user_id = $2
	`, subID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *NotificationRepo) DeleteSubscriptionByEndpoint(ctx context.Context, endpoint string) {
	_, _ = r.pool.Exec(ctx, `DELETE FROM user_push_subscriptions WHERE endpoint = $1`, endpoint)
}

func (r *NotificationRepo) ResolveAccountName(ctx context.Context, accountID int64) string {
	if accountID <= 0 {
		return ""
	}
	var name string
	_ = r.pool.QueryRow(ctx, `SELECT name FROM accounts WHERE id = $1`, accountID).Scan(&name)
	return name
}

// ListRecipientsForAccount retorna todos os usuários admin que possuem acesso à empresa:
// superadmins/owners globais, dono direto da empresa (accounts.owner_id) e usuários vinculados (account_id).
func (r *NotificationRepo) ListRecipientsForAccount(ctx context.Context, accountID int64, ownersOnly bool) ([]NotificationRecipient, error) {
	accName := r.ResolveAccountName(ctx, accountID)
	rows, err := r.pool.Query(ctx, `
		SELECT u.id,
			COALESCE(p.account_scope_mode, 'all'),
			COALESCE(p.enabled_account_ids, '{}'),
			COALESCE(p.whatsapp_connection, TRUE),
			COALESCE(p.lead_created, TRUE),
			COALESCE(p.ai_lead_qualified, TRUE),
			COALESCE(p.sale_approved, TRUE),
			COALESCE(p.meta_account_lifecycle, TRUE),
			COALESCE(p.google_ads_account_lifecycle, TRUE),
			COALESCE(p.pixel_lifecycle, TRUE),
			COALESCE(p.event_dispatch_error, FALSE)
		FROM admin_users u
		LEFT JOIN accounts a ON a.id = $1
		LEFT JOIN user_notification_preferences p ON p.user_id = u.id
		WHERE (
			COALESCE(u.role, 'superadmin') IN ('superadmin', 'owner')
			OR (a.owner_id IS NOT NULL AND u.id = a.owner_id)
			OR (NOT $2::boolean AND $1 > 0 AND u.account_id = $1)
		)
	`, accountID, ownersOnly)
	if err != nil {
		return nil, fmt.Errorf("list notification recipients: %w", err)
	}
	defer rows.Close()

	var out []NotificationRecipient
	for rows.Next() {
		var rec NotificationRecipient
		rec.AccountName = accName
		if err := rows.Scan(
			&rec.UserID,
			&rec.Prefs.AccountScopeMode,
			&rec.Prefs.EnabledAccountIDs,
			&rec.Prefs.WhatsAppConnection,
			&rec.Prefs.LeadCreated,
			&rec.Prefs.AILeadQualified,
			&rec.Prefs.SaleApproved,
			&rec.Prefs.MetaAccountLifecycle,
			&rec.Prefs.GoogleAdsAccountLifecycle,
			&rec.Prefs.PixelLifecycle,
			&rec.Prefs.EventDispatchError,
		); err != nil {
			return nil, fmt.Errorf("scan notification recipient: %w", err)
		}
		rec.Prefs.UserID = rec.UserID
		out = append(out, rec)
	}
	return out, rows.Err()
}
