package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func (h *AdminGoogleAdsAccounts) loadValidOAuthSource(ctx context.Context, scope TenantScope, dstAccountID, sourceID int64) (*domain.GoogleAdsAccount, error) {
	src, err := h.accounts.FindByID(ctx, sourceID)
	if err != nil || src == nil || strings.TrimSpace(src.RefreshToken) == "" {
		return nil, errors.New("conta Google de origem inválida ou desconectada")
	}
	if !scope.IsOwner && src.AccountID != dstAccountID {
		return nil, errors.New("sem acesso à conta Google de origem")
	}
	return src, nil
}

func (h *AdminGoogleAdsAccounts) applyOAuthSource(ctx context.Context, dst *domain.GoogleAdsAccount, src *domain.GoogleAdsAccount) error {
	if dst.OAuthEmail != "" && src.OAuthEmail != "" && !strings.EqualFold(dst.OAuthEmail, src.OAuthEmail) {
		_ = h.accounts.ClearTokens(ctx, dst.ID)
		dst.CustomerID, dst.CustomerName, dst.LoginCustomerID = "", "", ""
		dst.ConversionActions = []domain.GoogleAdsConversionAction{}
	}
	if err := h.accounts.SetTokens(ctx, dst.ID, src.RefreshToken, src.OAuthEmail, src.OAuthPictureURL); err != nil {
		return errors.New("erro ao reutilizar conexão OAuth do Google Ads")
	}
	dst.RefreshToken = src.RefreshToken
	dst.OAuthEmail = src.OAuthEmail
	dst.OAuthPictureURL = src.OAuthPictureURL
	return nil
}

func (h *AdminGoogleAdsAccounts) ReuseOAuth(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para gerenciar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	var body struct {
		SourceAccountID int64 `json:"source_account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SourceAccountID <= 0 {
		writeError(w, http.StatusBadRequest, "source_account_id é obrigatório")
		return
	}
	scope := TenantFromContext(r.Context())
	src, err := h.loadValidOAuthSource(r.Context(), scope, a.AccountID, body.SourceAccountID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.applyOAuthSource(r.Context(), a, src); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.invalidateAccountCaches(r.Context(), a.AccountID, a.ID)
	writeJSON(w, http.StatusOK, a.PublicView())
}

func (h *AdminGoogleAdsAccounts) cachedRemoteMap(ctx context.Context, cacheKey string, bypass bool, loader func() (map[string]any, error)) (map[string]any, error) {
	if !bypass && h.rdb != nil && cacheKey != "" {
		if raw, err := h.rdb.Get(ctx, cacheKey).Bytes(); err == nil && len(raw) > 0 {
			var cached map[string]any
			if json.Unmarshal(raw, &cached) == nil {
				return cached, nil
			}
		}
	}
	res, err := loader()
	if err != nil {
		return nil, err
	}
	if h.rdb != nil && cacheKey != "" && res != nil {
		if raw, merr := json.Marshal(res); merr == nil {
			_ = h.rdb.Set(ctx, cacheKey, raw, 3*time.Minute).Err()
		}
	}
	return res, nil
}

func (h *AdminGoogleAdsAccounts) ListCustomers(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	refresh := r.URL.Query().Get("refresh") == "1"
	key := fmt.Sprintf("tf:gads:customers:%d:%d", a.ID, a.UpdatedAt.UnixNano())
	res, err := h.cachedRemoteMap(r.Context(), key, refresh, func() (map[string]any, error) {
		return h.client.ListCustomers(r.Context(), a)
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if pic, _ := res["oauth_picture_url"].(string); pic != "" && a.OAuthPictureURL == "" {
		_ = h.accounts.UpdateOAuthPicture(r.Context(), a.ID, pic)
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminGoogleAdsAccounts) ListConversionActions(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	cid := strings.TrimSpace(r.URL.Query().Get("customer_id"))
	lcid := strings.TrimSpace(r.URL.Query().Get("login_customer_id"))
	refresh := r.URL.Query().Get("refresh") == "1"
	key := fmt.Sprintf("tf:gads:actions:%d:%d:%s:%s", a.ID, a.UpdatedAt.UnixNano(), cid, lcid)
	res, err := h.cachedRemoteMap(r.Context(), key, refresh, func() (map[string]any, error) {
		return h.client.ListConversionActions(r.Context(), a, cid, lcid)
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminGoogleAdsAccounts) CreateConversionAction(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para gerenciar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	var body struct {
		CustomerID      string `json:"customer_id"`
		LoginCustomerID string `json:"login_customer_id"`
		Name            string `json:"name"`
		EventType       string `json:"event_type"`
		Category        string `json:"category"`
		Currency        string `json:"currency"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	res, err := h.client.CreateConversionAction(r.Context(), a, body.CustomerID, body.LoginCustomerID, map[string]any{
		"name":       strings.TrimSpace(body.Name),
		"event_type": strings.TrimSpace(body.EventType),
		"category":   strings.TrimSpace(body.Category),
		"currency":   strings.TrimSpace(body.Currency),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.invalidateAccountCaches(r.Context(), a.AccountID, a.ID)
	writeJSON(w, http.StatusCreated, res)
}

func onlyDigitsStr(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func sanitizeConversionActions(in []domain.GoogleAdsConversionAction) []domain.GoogleAdsConversionAction {
	out := make([]domain.GoogleAdsConversionAction, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, ca := range in {
		ev := strings.TrimSpace(ca.EventType)
		id := strings.TrimSpace(ca.ConversionActionID)
		if ev == "" || id == "" {
			continue
		}
		dedupKey := ev + "::" + id
		if _, exists := seen[dedupKey]; exists {
			continue
		}
		seen[dedupKey] = struct{}{}
		out = append(out, domain.GoogleAdsConversionAction{
			EventType:          ev,
			ConversionActionID: id,
			Name:               strings.TrimSpace(ca.Name),
			ActionType:         strings.ToUpper(strings.TrimSpace(ca.ActionType)),
			OwnerCustomerID:    onlyDigitsStr(ca.OwnerCustomerID),
			Enabled:            ca.Enabled,
		})
	}
	return out
}
