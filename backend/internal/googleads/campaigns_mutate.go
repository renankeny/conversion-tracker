package googleads

import (
	"context"
	"errors"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func (c *Client) MutateStatus(
	ctx context.Context,
	account *domain.GoogleAdsAccount,
	customerID, loginCustomerID, level, entitySubtype, objectID, parentID, status string,
) error {
	if account == nil || strings.TrimSpace(account.RefreshToken) == "" {
		return errors.New("conta Google Ads não conectada")
	}
	cid := strings.TrimSpace(customerID)
	if cid == "" {
		cid = strings.TrimSpace(account.CustomerID)
	}
	lcid := strings.TrimSpace(loginCustomerID)
	if lcid == "" {
		lcid = strings.TrimSpace(account.LoginCustomerID)
	}
	payload := map[string]any{
		"level":          level,
		"entity_subtype": entitySubtype,
		"object_id":      objectID,
		"parent_id":      parentID,
		"status":         status,
	}
	var out map[string]any
	return c.oauth.CallCentralProxy(ctx, "mutate_status", account.RefreshToken, cid, lcid, payload, &out)
}

func (c *Client) MutateBudget(
	ctx context.Context,
	account *domain.GoogleAdsAccount,
	customerID, loginCustomerID, budgetID string,
	dailyBudget float64,
) error {
	if account == nil || strings.TrimSpace(account.RefreshToken) == "" {
		return errors.New("conta Google Ads não conectada")
	}
	cid := strings.TrimSpace(customerID)
	if cid == "" {
		cid = strings.TrimSpace(account.CustomerID)
	}
	lcid := strings.TrimSpace(loginCustomerID)
	if lcid == "" {
		lcid = strings.TrimSpace(account.LoginCustomerID)
	}
	payload := map[string]any{
		"budget_id":    budgetID,
		"daily_budget": dailyBudget,
	}
	var out map[string]any
	return c.oauth.CallCentralProxy(ctx, "mutate_budget", account.RefreshToken, cid, lcid, payload, &out)
}
