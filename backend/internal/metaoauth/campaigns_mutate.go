package metaoauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func directGraphGETOptionalProof(ctx context.Context, lease metaLeaseEntry, relPath string, params url.Values) ([]byte, int, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", lease.accessToken)
	if lease.appSecretProof != "" {
		params.Set("appsecret_proof", lease.appSecretProof)
	}
	ver := strings.TrimSpace(lease.graphVersion)
	if ver == "" {
		ver = metaCampaignsGraphVersion
	}
	u := fmt.Sprintf("%s/%s/%s?%s", strings.TrimRight(metaDirectBaseURL, "/"), ver, strings.TrimLeft(relPath, "/"), params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := newDirectMetaHTTPClient(15 * time.Second).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return raw, detectMetaErrStatus(resp.StatusCode, raw), parseMetaGraphError(resp.StatusCode, raw)
	}
	return raw, resp.StatusCode, nil
}

func directGraphPOSTForm(ctx context.Context, lease metaLeaseEntry, objectID string, form url.Values) ([]byte, int, error) {
	cleanID := digitsOnlyStr(objectID)
	if cleanID == "" {
		return nil, http.StatusBadRequest, errors.New("object_id Meta inválido")
	}
	if form == nil {
		form = url.Values{}
	}
	form.Set("access_token", lease.accessToken)
	if lease.appSecretProof != "" {
		form.Set("appsecret_proof", lease.appSecretProof)
	}
	ver := strings.TrimSpace(lease.graphVersion)
	if ver == "" {
		ver = metaCampaignsGraphVersion
	}
	u := fmt.Sprintf("%s/%s/%s", strings.TrimRight(metaDirectBaseURL, "/"), ver, cleanID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := newDirectMetaHTTPClient(15 * time.Second).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return raw, detectMetaErrStatus(resp.StatusCode, raw), parseMetaGraphError(resp.StatusCode, raw)
	}
	return raw, resp.StatusCode, nil
}

func detectMetaErrStatus(httpStatus int, raw []byte) int {
	var gErr struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &gErr)
	lower := strings.ToLower(gErr.Error.Message)
	if gErr.Error.Code == 190 || strings.Contains(lower, "validating access token") || strings.Contains(lower, "invalid oauth access token") {
		return http.StatusUnauthorized
	}
	return httpStatus
}

func parseMetaGraphError(httpStatus int, raw []byte) error {
	var gErr struct {
		Error struct {
			Message      string `json:"message"`
			ErrorUserMsg string `json:"error_user_msg"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &gErr)
	msg := strings.TrimSpace(gErr.Error.ErrorUserMsg)
	if msg == "" {
		msg = strings.TrimSpace(gErr.Error.Message)
	}
	if msg == "" {
		msg = fmt.Sprintf("Meta Graph API status %d", httpStatus)
	}
	return &ProxyError{StatusCode: detectMetaErrStatus(httpStatus, raw), Body: string(raw), Message: "Meta Graph API v26.0: " + msg}
}

// MutateObjectStatus ativa ou pausa Campanha, Conjunto ou Anúncio na Meta Graph API v26.0.
func (c *Client) MutateObjectStatus(ctx context.Context, acc *domain.MetaAccount, permanentToken, objectID, status string) error {
	target := "PAUSED"
	if strings.EqualFold(status, "ACTIVE") || strings.EqualFold(status, "ENABLED") {
		target = "ACTIVE"
	}
	form := url.Values{"status": {target}}
	if acc != nil && strings.TrimSpace(acc.VaultTokenEncrypted) != "" {
		_, err := runWithMetaLease(ctx, c, acc, func(lease metaLeaseEntry) (bool, int, error) {
			lease.graphVersion = metaCampaignsGraphVersion
			_, code, callErr := directGraphPOSTForm(ctx, lease, objectID, form)
			return callErr == nil, code, callErr
		}, nil)
		return err
	}
	if tok := strings.TrimSpace(permanentToken); tok != "" {
		lease := metaLeaseEntry{accessToken: tok, graphVersion: metaCampaignsGraphVersion}
		_, _, err := directGraphPOSTForm(ctx, lease, objectID, form)
		return err
	}
	return errors.New("conta Meta Ads sem conexão OAuth ou token permanente")
}

// MutateObjectDailyBudget altera o orçamento diário (em moeda real) de Campanha ou Conjunto na Meta v26.0.
func (c *Client) MutateObjectDailyBudget(ctx context.Context, acc *domain.MetaAccount, permanentToken, objectID string, dailyBudget float64, currency string) error {
	if dailyBudget <= 0 {
		return errors.New("daily_budget deve ser maior que zero")
	}
	minor := int64(math.Round(dailyBudget * metaBudgetDivisor(currency)))
	if minor <= 0 {
		return errors.New("daily_budget inválido após conversão")
	}
	form := url.Values{"daily_budget": {strconv.FormatInt(minor, 10)}}
	if acc != nil && strings.TrimSpace(acc.VaultTokenEncrypted) != "" {
		_, err := runWithMetaLease(ctx, c, acc, func(lease metaLeaseEntry) (bool, int, error) {
			lease.graphVersion = metaCampaignsGraphVersion
			_, code, callErr := directGraphPOSTForm(ctx, lease, objectID, form)
			return callErr == nil, code, callErr
		}, nil)
		return err
	}
	if tok := strings.TrimSpace(permanentToken); tok != "" {
		lease := metaLeaseEntry{accessToken: tok, graphVersion: metaCampaignsGraphVersion}
		_, _, err := directGraphPOSTForm(ctx, lease, objectID, form)
		return err
	}
	return errors.New("conta Meta Ads sem conexão OAuth ou token permanente")
}

func metaBudgetDivisor(currency string) float64 {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "CLP", "JPY", "KRW", "PYG", "VND", "ISK", "HUF", "TWD", "IDR":
		return 1.0
	default:
		return 100.0
	}
}

func metaFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f
	case int64:
		return float64(x)
	default:
		return 0
	}
}

func metaInt64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	case int64:
		return x
	default:
		return 0
	}
}

func metaStr(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return strings.TrimSpace(v)
}

func normalizeMetaStatus(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "ACTIVE":
		return "ACTIVE"
	case "PAUSED":
		return "PAUSED"
	case "ARCHIVED", "DELETED":
		return "ARCHIVED"
	default:
		return "UNKNOWN"
	}
}
