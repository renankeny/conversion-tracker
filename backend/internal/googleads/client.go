// Package googleads (cliente da Data Manager & Google Ads API via Gateway Criptografado).
package googleads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type Client struct {
	developerToken string
	oauth          OAuthConfig
}

func New(developerToken string, oauth OAuthConfig) *Client {
	return &Client{
		developerToken: developerToken,
		oauth:          oauth,
	}
}

type SendResult struct {
	CustomerID  string
	ActionID    string
	StatusCode  int
	Body        string
	RequestBody string
	Err         error
}

type SendOptions struct {
	ValidateOnly bool
}

func (c *Client) ListCustomers(ctx context.Context, account *domain.GoogleAdsAccount) (map[string]any, error) {
	if account == nil || account.RefreshToken == "" {
		return map[string]any{"customers": []any{}}, nil
	}
	var out map[string]any
	if err := c.oauth.CallCentralProxy(ctx, "list_customers", account.RefreshToken, account.CustomerID, account.LoginCustomerID, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListConversionActions(ctx context.Context, account *domain.GoogleAdsAccount, customerID, loginCustomerID string) (map[string]any, error) {
	if account == nil || account.RefreshToken == "" {
		return map[string]any{"conversion_actions": []any{}}, nil
	}
	cid := strings.TrimSpace(customerID)
	if cid == "" {
		cid = account.CustomerID
	}
	lcid := strings.TrimSpace(loginCustomerID)
	if lcid == "" {
		lcid = account.LoginCustomerID
	}
	var out map[string]any
	if err := c.oauth.CallCentralProxy(ctx, "list_conversion_actions", account.RefreshToken, cid, lcid, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateConversionAction(ctx context.Context, account *domain.GoogleAdsAccount, customerID, loginCustomerID string, reqPayload map[string]any) (map[string]any, error) {
	if account == nil || account.RefreshToken == "" {
		return nil, fmt.Errorf("conta Google Ads sem conexão OAuth ativa")
	}
	cid := strings.TrimSpace(customerID)
	if cid == "" {
		cid = account.CustomerID
	}
	lcid := strings.TrimSpace(loginCustomerID)
	if lcid == "" {
		lcid = account.LoginCustomerID
	}
	var out map[string]any
	if err := c.oauth.CallCentralProxy(ctx, "create_conversion_action", account.RefreshToken, cid, lcid, reqPayload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Send(ctx context.Context, account *domain.GoogleAdsAccount, action *domain.GoogleAdsConversionAction, in domain.SendEventInput) []SendResult {
	return c.SendWithOptions(ctx, account, action, in, SendOptions{})
}

func (c *Client) SendWithOptions(ctx context.Context, account *domain.GoogleAdsAccount, action *domain.GoogleAdsConversionAction, in domain.SendEventInput, opts SendOptions) []SendResult {
	if account == nil || !account.HasCredentials() {
		return nil
	}
	if action == nil || strings.TrimSpace(action.ConversionActionID) == "" {
		return nil
	}

	userData := buildUserData(in)
	hasClickID := in.GCLID != "" || in.WBraid != "" || in.GBraid != ""
	if !hasClickID && len(userData) == 0 {
		return nil
	}

	operatingID, actionID := resolveTargetOperatingAndAction(account, action)
	loginID := digitsOnlyStr(account.LoginCustomerID)

	res := SendResult{CustomerID: account.CustomerID, ActionID: actionID}
	body := buildDataManagerPayload(operatingID, loginID, actionID, in, userData, opts.ValidateOnly)
	raw, err := json.Marshal(body)
	if err != nil {
		res.Err = err
		return []SendResult{res}
	}
	res.RequestBody = string(raw)

	code, respBody, err := c.sendConversionWithLease(ctx, account, raw)
	if err != nil {
		res.Err = err
		return []SendResult{res}
	}

	if code == 400 && isDestinationNotFoundBody(respBody) && loginID != "" && loginID != operatingID {
		altBody := buildDataManagerPayload(loginID, loginID, actionID, in, userData, opts.ValidateOnly)
		if altRaw, merr := json.Marshal(altBody); merr == nil {
			if altCode, altResp, altErr := c.sendConversionWithLease(ctx, account, altRaw); altErr == nil {
				code, respBody = altCode, altResp
				res.RequestBody = string(altRaw)
			}
		}
	}

	res.StatusCode = code
	res.Body = respBody
	if res.StatusCode >= 300 {
		res.Err = fmt.Errorf("%s", formatGoogleDataManagerError(res.StatusCode, res.Body, action, operatingID))
	} else if res.StatusCode >= 200 && res.StatusCode < 300 {
		if pfMsg := detectPartialFailure(res.Body); pfMsg != "" {
			res.StatusCode = http.StatusBadRequest
			res.Err = fmt.Errorf("google ads partial failure: %s", pfMsg)
		}
	}
	return []SendResult{res}
}
