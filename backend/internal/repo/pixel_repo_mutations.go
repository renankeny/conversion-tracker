package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// RegenerateToken gera um novo public_token para o pixel.
func (r *PixelRepo) RegenerateToken(ctx context.Context, id int64) (string, error) {
	newToken := domain.GeneratePublicToken()
	_, err := r.pool.Exec(ctx,
		"UPDATE pixels SET public_token = $1, updated_at = NOW() WHERE id = $2",
		newToken, id,
	)
	if err != nil {
		return "", err
	}
	return newToken, nil
}

// UpdateDashboardConfig atualiza apenas a config da dashboard do pixel.
func (r *PixelRepo) UpdateDashboardConfig(ctx context.Context, id int64, cfg domain.DashboardConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal dashboard_config: %w", err)
	}
	const q = `UPDATE pixels SET dashboard_config = $1, updated_at = NOW() WHERE id = $2 RETURNING id`
	var gotID int64
	err = r.pool.QueryRow(ctx, q, raw, id).Scan(&gotID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// GenerateDashboardShareToken cria (ou substitui) o token do link público da dashboard do pixel.
func (r *PixelRepo) GenerateDashboardShareToken(ctx context.Context, id int64) (string, error) {
	newToken := domain.GenerateDashboardShareToken(id)
	_, err := r.pool.Exec(ctx,
		"UPDATE pixels SET dashboard_share_token = $1, updated_at = NOW() WHERE id = $2",
		newToken, id,
	)
	if err != nil {
		return "", err
	}
	return newToken, nil
}

// UpdateFinancialConfig atualiza apenas as taxas e custos operacionais do pixel.
func (r *PixelRepo) UpdateFinancialConfig(ctx context.Context, id, accountID int64, cfg domain.PixelFinancialConfig) error {
	if cfg.GatewayFeePct < 0 {
		cfg.GatewayFeePct = 0
	}
	if cfg.GatewayFixedFee < 0 {
		cfg.GatewayFixedFee = 0
	}
	if cfg.TaxPct < 0 {
		cfg.TaxPct = 0
	}
	if cfg.ProductCost < 0 {
		cfg.ProductCost = 0
	}
	const q = `
		UPDATE pixels
		SET gateway_fee_pct = $1, gateway_fixed_fee = $2, tax_pct = $3, product_cost = $4, updated_at = NOW()
		WHERE id = $5 AND ($6 = 0 OR account_id = $6)
		RETURNING name, public_token
	`
	var name, pubToken string
	err := r.pool.QueryRow(ctx, q, cfg.GatewayFeePct, cfg.GatewayFixedFee, cfg.TaxPct, cfg.ProductCost, id, accountID).Scan(&name, &pubToken)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	r.InvalidateCache(ctx, name, pubToken)
	return nil
}

// RevokeDashboardShareToken desativa o link público da dashboard do pixel.
func (r *PixelRepo) RevokeDashboardShareToken(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE pixels SET dashboard_share_token = '', updated_at = NOW() WHERE id = $1",
		id,
	)
	return err
}

// MigratePixelAccountData move contadores, eventos e leads históricos de um pixel
// quando o pixel é transferido de fromAccountID para toAccountID.
func (r *PixelRepo) MigratePixelAccountData(ctx context.Context, pixelName string, fromAccountID, toAccountID int64) error {
	if pixelName == "" || fromAccountID <= 0 || toAccountID <= 0 || fromAccountID == toAccountID {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migrate pixel account tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const mergeCounters = `
		INSERT INTO event_counters (account_id, pixel_name, event_name, bucket_at, count_total, count_sent, count_failed, updated_at)
		SELECT $1, pixel_name, event_name, bucket_at, count_total, count_sent, count_failed, NOW()
		FROM event_counters
		WHERE pixel_name = $2 AND account_id = $3
		ON CONFLICT (account_id, pixel_name, event_name, bucket_at)
		DO UPDATE SET
			count_total  = event_counters.count_total + EXCLUDED.count_total,
			count_sent   = event_counters.count_sent + EXCLUDED.count_sent,
			count_failed = event_counters.count_failed + EXCLUDED.count_failed,
			updated_at   = NOW()`
	if _, err := tx.Exec(ctx, mergeCounters, toAccountID, pixelName, fromAccountID); err != nil {
		return fmt.Errorf("merge event_counters: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM event_counters WHERE pixel_name = $1 AND account_id = $2`, pixelName, fromAccountID); err != nil {
		return fmt.Errorf("cleanup old event_counters: %w", err)
	}

	const moveEvents = `
		UPDATE sent_events se
		SET account_id = $1
		WHERE se.pixel_name = $2 AND se.account_id = $3
		  AND NOT EXISTS (
		      SELECT 1 FROM sent_events existing
		      WHERE existing.account_id = $1
		        AND existing.pixel_name = se.pixel_name
		        AND existing.channel = se.channel
		        AND existing.event_id = se.event_id
		  )`
	if _, err := tx.Exec(ctx, moveEvents, toAccountID, pixelName, fromAccountID); err != nil {
		return fmt.Errorf("move sent_events: %w", err)
	}

	const moveUsers = `
		UPDATE tracked_users u
		SET account_id = $1, group_id = NULL
		WHERE u.pixel_name = $2 AND u.account_id = $3
		  AND NOT EXISTS (
		      SELECT 1 FROM tracked_users existing
		      WHERE existing.account_id = $1
		        AND existing.external_id = u.external_id
		  )`
	if _, err := tx.Exec(ctx, moveUsers, toAccountID, pixelName, fromAccountID); err != nil {
		return fmt.Errorf("move tracked_users: %w", err)
	}

	const detachOldCRMLeads = `
		DELETE FROM crm_leads cl
		USING tracked_users u
		WHERE cl.tracked_user_id = u.id
		  AND cl.account_id = $1
		  AND u.account_id = $2
		  AND u.pixel_name = $3`
	if _, err := tx.Exec(ctx, detachOldCRMLeads, fromAccountID, toAccountID, pixelName); err != nil {
		return fmt.Errorf("cleanup cross-tenant crm_leads: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	cache.BumpDashboardVersion(ctx, r.rdb, fromAccountID)
	cache.BumpDashboardVersion(ctx, r.rdb, toAccountID)
	return nil
}
