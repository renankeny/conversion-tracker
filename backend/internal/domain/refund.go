package domain

import "time"

const (
	RefundTypeRefund     = "refund"
	RefundTypeChargeback = "chargeback"

	CRMRefundNone       = "none"
	CRMRefundRefunded   = "refunded"
	CRMRefundChargeback = "chargeback"

	AdjustmentStatusPending = "pending"
	AdjustmentStatusSent    = "sent"
	AdjustmentStatusSkipped = "skipped"
	AdjustmentStatusFailed  = "failed"
)

// SaleRefund representa um estorno (reembolso ou chargeback) persistido permanentemente
// para abatimento de receita líquida, LTV, metas e retratação em Google Ads / GA4.
type SaleRefund struct {
	ID                     int64     `json:"id"`
	AccountID              int64     `json:"account_id"`
	PixelName              string    `json:"pixel_name"`
	Gateway                string    `json:"gateway"`
	OrderID                string    `json:"order_id"`
	RefundType             string    `json:"refund_type"` // "refund" | "chargeback"
	Amount                 float64   `json:"amount"`
	Currency               string    `json:"currency"`
	OriginalEventID        string    `json:"original_event_id"`
	TrackedUserID          *int64    `json:"tracked_user_id,omitempty"`
	GroupID                *int64    `json:"group_id,omitempty"`
	CRMLeadID              *int64    `json:"crm_lead_id,omitempty"`
	ProductName            string    `json:"product_name"`
	GoogleRetractionStatus string    `json:"google_retraction_status"`
	GoogleRetractionError  string    `json:"google_retraction_error,omitempty"`
	GA4RefundStatus        string    `json:"ga4_refund_status"`
	GA4RefundError         string    `json:"ga4_refund_error,omitempty"`
	RawPayload             []byte    `json:"-"`
	RefundedAt             time.Time `json:"refunded_at"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// RefundSummary agrega totais de estorno para um recorte de conta/pixel/período.
type RefundSummary struct {
	RefundCount     int64   `json:"refund_count"`
	ChargebackCount int64   `json:"chargeback_count"`
	TotalCount      int64   `json:"total_count"`
	RefundAmount    float64 `json:"refund_amount"`
	ChargebackAmt   float64 `json:"chargeback_amount"`
	TotalAmount     float64 `json:"total_amount"`
}

func NormalizeRefundType(evName string) string {
	if evName == EventChargeback || evName == RefundTypeChargeback {
		return RefundTypeChargeback
	}
	return RefundTypeRefund
}
