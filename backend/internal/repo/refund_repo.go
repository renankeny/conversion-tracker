package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type RefundRepo struct {
	pool *pgxpool.Pool
}

func NewRefundRepo(pool *pgxpool.Pool) *RefundRepo {
	return &RefundRepo{pool: pool}
}

// Upsert grava um estorno de forma idempotente por (account_id, gateway, order_id).
// Retorna inserted=true somente na primeira inserção (evita abater CRM ou notificar duas vezes).
func (r *RefundRepo) Upsert(ctx context.Context, ref *domain.SaleRefund) (bool, error) {
	if ref.RefundedAt.IsZero() {
		ref.RefundedAt = time.Now()
	}
	if strings.TrimSpace(ref.Currency) == "" {
		ref.Currency = "BRL"
	}
	if ref.GoogleRetractionStatus == "" {
		ref.GoogleRetractionStatus = domain.AdjustmentStatusPending
	}
	if ref.GA4RefundStatus == "" {
		ref.GA4RefundStatus = domain.AdjustmentStatusPending
	}
	var inserted bool
	const q = `
		INSERT INTO sale_refunds (
			account_id, pixel_name, gateway, order_id, refund_type, amount, currency,
			original_event_id, tracked_user_id, group_id, crm_lead_id, product_name,
			google_retraction_status, ga4_refund_status, raw_payload, refunded_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (account_id, gateway, order_id) DO UPDATE SET
			refund_type = CASE
				WHEN EXCLUDED.refund_type = 'chargeback' THEN 'chargeback'
				ELSE sale_refunds.refund_type
			END,
			amount          = GREATEST(sale_refunds.amount, EXCLUDED.amount),
			pixel_name      = CASE WHEN sale_refunds.pixel_name = '' THEN EXCLUDED.pixel_name ELSE sale_refunds.pixel_name END,
			tracked_user_id = COALESCE(sale_refunds.tracked_user_id, EXCLUDED.tracked_user_id),
			group_id        = COALESCE(sale_refunds.group_id, EXCLUDED.group_id),
			crm_lead_id     = COALESCE(sale_refunds.crm_lead_id, EXCLUDED.crm_lead_id),
			updated_at      = NOW()
		RETURNING id, (xmax = 0) AS inserted, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, q,
		ref.AccountID, ref.PixelName, ref.Gateway, ref.OrderID, ref.RefundType, ref.Amount, ref.Currency,
		ref.OriginalEventID, ref.TrackedUserID, ref.GroupID, ref.CRMLeadID, ref.ProductName,
		ref.GoogleRetractionStatus, ref.GA4RefundStatus, ref.RawPayload, ref.RefundedAt,
	).Scan(&ref.ID, &inserted, &ref.CreatedAt, &ref.UpdatedAt)
	return inserted, err
}

func (r *RefundRepo) FindByID(ctx context.Context, id int64) (*domain.SaleRefund, error) {
	var ref domain.SaleRefund
	const q = `
		SELECT id, account_id, pixel_name, gateway, order_id, refund_type, amount, currency,
		       original_event_id, tracked_user_id, group_id, crm_lead_id, product_name,
		       google_retraction_status, google_retraction_error,
		       ga4_refund_status, ga4_refund_error, refunded_at, created_at, updated_at
		FROM sale_refunds
		WHERE id = $1
	`
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&ref.ID, &ref.AccountID, &ref.PixelName, &ref.Gateway, &ref.OrderID, &ref.RefundType,
		&ref.Amount, &ref.Currency, &ref.OriginalEventID, &ref.TrackedUserID, &ref.GroupID,
		&ref.CRMLeadID, &ref.ProductName, &ref.GoogleRetractionStatus, &ref.GoogleRetractionError,
		&ref.GA4RefundStatus, &ref.GA4RefundError, &ref.RefundedAt, &ref.CreatedAt, &ref.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &ref, err
}

func (r *RefundRepo) UpdateAdjustmentStatus(ctx context.Context, id int64, channel, status, errMsg string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	if channel == "google" {
		_, err := r.pool.Exec(ctx, `
			UPDATE sale_refunds
			SET google_retraction_status = $2, google_retraction_error = $3, updated_at = NOW()
			WHERE id = $1
		`, id, status, errMsg)
		return err
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE sale_refunds
		SET ga4_refund_status = $2, ga4_refund_error = $3, updated_at = NOW()
		WHERE id = $1
	`, id, status, errMsg)
	return err
}

func (r *RefundRepo) SetCRMLeadID(ctx context.Context, refundID, leadID int64) {
	if refundID <= 0 || leadID <= 0 {
		return
	}
	_, _ = r.pool.Exec(ctx, `UPDATE sale_refunds SET crm_lead_id = $2, updated_at = NOW() WHERE id = $1`, refundID, leadID)
}

// FindOriginalSaleByOrder localiza a venda original em sent_events (recente) ou webhook_logs (histórico)
// para herdar pixel_name, external_id, event_id e valor exato mesmo semanas após o expurgo de sent_events.
func (r *RefundRepo) FindOriginalSaleByOrder(ctx context.Context, accountID int64, orderID string) (pixelName, externalID, eventID string, value float64) {
	clean := strings.TrimSpace(orderID)
	if accountID <= 0 || clean == "" {
		return "", "", "", 0
	}
	err := r.pool.QueryRow(ctx, `
		SELECT pixel_name, external_id, event_id, COALESCE(purchase_value, 0)
		FROM sent_events
		WHERE account_id = $1
		  AND event_name = 'Purchase'
		  AND (event_id = $2 OR event_id LIKE $2 || ':%')
		ORDER BY created_at DESC
		LIMIT 1
	`, accountID, clean).Scan(&pixelName, &externalID, &eventID, &value)
	if err == nil && (pixelName != "" || externalID != "") {
		return pixelName, externalID, eventID, value
	}

	_ = r.pool.QueryRow(ctx, `
		SELECT pixel_name, external_id, event_id,
		       COALESCE((input_json->>'value')::numeric, 0)
		FROM webhook_logs
		WHERE account_id = $1
		  AND event_name = 'Purchase'
		  AND (event_id = $2 OR COALESCE(input_json->>'order_id', '') = $2)
		ORDER BY received_at DESC
		LIMIT 1
	`, accountID, clean).Scan(&pixelName, &externalID, &eventID, &value)
	return pixelName, externalID, eventID, value
}

// SummaryByPeriod agrega reembolsos e chargebacks por conta, pixel opcional e período.
func (r *RefundRepo) SummaryByPeriod(ctx context.Context, accountID int64, pixelName string, from, to time.Time) (domain.RefundSummary, error) {
	var out domain.RefundSummary
	const q = `
		SELECT
			COALESCE(COUNT(*) FILTER (WHERE refund_type = 'refund'), 0),
			COALESCE(COUNT(*) FILTER (WHERE refund_type = 'chargeback'), 0),
			COALESCE(COUNT(*), 0),
			COALESCE(SUM(amount) FILTER (WHERE refund_type = 'refund'), 0),
			COALESCE(SUM(amount) FILTER (WHERE refund_type = 'chargeback'), 0),
			COALESCE(SUM(amount), 0)
		FROM sale_refunds
		WHERE ($1 = 0 OR account_id = $1)
		  AND ($2 = '' OR pixel_name = $2)
		  AND refunded_at >= $3 AND refunded_at <= $4
	`
	err := r.pool.QueryRow(ctx, q, accountID, pixelName, from, to).Scan(
		&out.RefundCount, &out.ChargebackCount, &out.TotalCount,
		&out.RefundAmount, &out.ChargebackAmt, &out.TotalAmount,
	)
	return out, err
}
