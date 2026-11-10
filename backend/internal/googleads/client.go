// Package googleads (cliente da Data Manager & Google Ads API via Gateway Criptografado).
package googleads

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
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

func (c *Client) Send(ctx context.Context, account *domain.GoogleAdsAccount, action *domain.GoogleAdsConversionAction, in domain.SendEventInput) []SendResult {
	return c.SendWithOptions(ctx, account, action, in, SendOptions{})
}

func (c *Client) SendWithOptions(ctx context.Context, account *domain.GoogleAdsAccount, action *domain.GoogleAdsConversionAction, in domain.SendEventInput, opts SendOptions) []SendResult {
	if account == nil || !account.HasCredentials() {
		return nil
	}
	if action == nil || action.ConversionActionID == "" {
		return nil
	}

	userData := buildUserData(in)
	hasClickID := in.GCLID != "" || in.WBraid != "" || in.GBraid != ""
	if !hasClickID && len(userData) == 0 {
		return nil
	}

	res := SendResult{CustomerID: account.CustomerID, ActionID: action.ConversionActionID}
	body := buildDataManagerPayload(account, action, in, userData, opts.ValidateOnly)
	raw, err := json.Marshal(body)
	if err != nil {
		res.Err = err
		return []SendResult{res}
	}
	res.RequestBody = string(raw)

	var proxyOut struct {
		CustomerID string `json:"customer_id"`
		StatusCode int    `json:"status_code"`
		Body       string `json:"body"`
	}
	if err := c.oauth.CallCentralProxy(ctx, "send_conversion", account.RefreshToken, account.CustomerID, account.LoginCustomerID, raw, &proxyOut); err != nil {
		res.Err = err
		return []SendResult{res}
	}

	res.StatusCode = proxyOut.StatusCode
	res.Body = proxyOut.Body
	if res.StatusCode >= 300 {
		res.Err = fmt.Errorf("google ads api returned %d: %s", res.StatusCode, res.Body)
	}
	return []SendResult{res}
}

func buildDataManagerPayload(account *domain.GoogleAdsAccount, action *domain.GoogleAdsConversionAction, in domain.SendEventInput, userData []map[string]any, validateOnly bool) map[string]any {
	operatingID := stripDashes(account.CustomerID)
	loginID := stripDashes(account.LoginCustomerID)
	if loginID == "" {
		loginID = operatingID
	}

	destination := map[string]any{
		"operatingAccount":     map[string]any{"accountType": "GOOGLE_ADS", "accountId": operatingID},
		"loginAccount":         map[string]any{"accountType": "GOOGLE_ADS", "accountId": loginID},
		"productDestinationId": strings.TrimSpace(action.ConversionActionID),
	}

	t := time.Unix(in.EventTime, 0)
	if in.EventTime <= 0 {
		t = time.Now()
	}
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	eventTimestamp := t.In(loc).Format(time.RFC3339)

	adIdentifiers := map[string]any{}
	switch {
	case in.GCLID != "":
		adIdentifiers["gclid"] = in.GCLID
	case in.GBraid != "":
		adIdentifiers["gbraid"] = in.GBraid
	case in.WBraid != "":
		adIdentifiers["wbraid"] = in.WBraid
	}

	event := map[string]any{
		"eventTimestamp": eventTimestamp,
		"eventSource":    "WEB",
	}
	if len(adIdentifiers) > 0 {
		event["adIdentifiers"] = adIdentifiers
	}
	if in.Value != nil {
		event["conversionValue"] = *in.Value
		event["currency"] = orDefault(in.Currency, "BRL")
	}
	if txID := orDefault(in.OrderID, in.EventID); txID != "" {
		event["transactionId"] = txID
	}
	if len(userData) > 0 {
		event["userData"] = map[string]any{"userIdentifiers": userData}
	}

	return map[string]any{
		"destinations": []map[string]any{destination},
		"encoding":     "HEX",
		"events":       []map[string]any{event},
		"validateOnly": validateOnly,
	}
}

func buildUserData(in domain.SendEventInput) []map[string]any {
	out := make([]map[string]any, 0, 4)
	if h := util.SHA256Hex(util.NormalizeEmail(in.Email)); h != "" {
		out = append(out, map[string]any{"emailAddress": h})
	}
	if h := util.SHA256Hex(util.NormalizePhoneE164(in.Phone)); h != "" {
		out = append(out, map[string]any{"phoneNumber": h})
	}
	first := util.NormalizeName(in.FirstName)
	last := util.NormalizeName(in.LastName)
	zip := util.NormalizeZip(in.PostalCode)
	// Google Data Manager API exige given_name, family_name, region_code E postal_code
	// simultaneamente quando o objeto `address` é enviado em userIdentifiers.
	if first != "" && last != "" && zip != "" {
		country := strings.ToUpper(util.NormalizeCountry(in.Country))
		if len(country) != 2 {
			country = "BR"
		}
		addr := map[string]any{
			"givenName":  util.SHA256Hex(first),
			"familyName": util.SHA256Hex(last),
			"postalCode": zip,
			"regionCode": country,
		}
		out = append(out, map[string]any{"address": addr})
	}
	return out
}

func stripDashes(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "-", "")
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
