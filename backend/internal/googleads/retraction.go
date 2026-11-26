package googleads

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// RetractResult consolida o resultado da chamada ConversionAdjustmentUploadService (RETRACTION) v26.
type RetractResult struct {
	OK                  bool   `json:"ok"`
	StatusCode          int    `json:"status_code"`
	PartialFailureError string `json:"partial_failure_error"`
	Body                string `json:"body"`
}

// RetractConversion envia ajuste oficial RETRACTION (Google Ads API v26) para anular uma venda estornada pelo order_id.
func (c *Client) RetractConversion(
	ctx context.Context,
	account *domain.GoogleAdsAccount,
	action *domain.GoogleAdsConversionAction,
	orderID string,
	refundedAt time.Time,
) (RetractResult, error) {
	if account == nil || !account.HasCredentials() {
		return RetractResult{}, errors.New("conta Google Ads sem credenciais")
	}
	if action == nil || strings.TrimSpace(action.ConversionActionID) == "" {
		return RetractResult{}, errors.New("ação de conversão Purchase não configurada na conta Google Ads")
	}
	cleanOrder := strings.TrimSpace(orderID)
	if cleanOrder == "" {
		return RetractResult{}, errors.New("order_id ausente para retratação")
	}

	operatingID, actionID := resolveTargetOperatingAndAction(account, action)
	if refundedAt.IsZero() {
		refundedAt = time.Now()
	}
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err == nil {
		refundedAt = refundedAt.In(loc)
	}
	adjDateTime := refundedAt.Format("2006-01-02 15:04:05-07:00")

	payload := map[string]any{
		"order_id":             cleanOrder,
		"conversion_action_id": actionID,
		"owner_customer_id":    operatingID,
		"adjustment_date_time": adjDateTime,
	}
	var res RetractResult
	if err := c.oauth.CallCentralProxy(ctx, "retract_conversion", account.RefreshToken, account.CustomerID, account.LoginCustomerID, payload, &res); err != nil {
		return res, err
	}
	if !res.OK {
		if res.PartialFailureError != "" {
			return res, fmt.Errorf("google ads retraction partial failure: %s", res.PartialFailureError)
		}
		return res, fmt.Errorf("google ads retraction status %d: %s", res.StatusCode, res.Body)
	}
	return res, nil
}
