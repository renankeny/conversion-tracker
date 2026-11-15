package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminGoogleAdsAccounts struct {
	accounts  *repo.GoogleAdsAccountRepo
	pixels    *repo.PixelRepo
	client    *googleads.Client
	oauth     googleads.OAuthConfig
	state     *cache.OAuthStateStore
	rdb       *redis.Client
	frontHost string
}

func NewAdminGoogleAdsAccounts(accounts *repo.GoogleAdsAccountRepo, client *googleads.Client, oauth googleads.OAuthConfig, frontHost string) *AdminGoogleAdsAccounts {
	return &AdminGoogleAdsAccounts{
		accounts:  accounts,
		client:    client,
		oauth:     oauth,
		state:     cache.NewOAuthStateStore(nil),
		frontHost: frontHost,
	}
}

func (h *AdminGoogleAdsAccounts) SetRedis(rdb *redis.Client) {
	h.rdb = rdb
	h.state = cache.NewOAuthStateStore(rdb)
}

func (h *AdminGoogleAdsAccounts) SetPixels(pixels *repo.PixelRepo) {
	h.pixels = pixels
}

func (h *AdminGoogleAdsAccounts) invalidateAccountCaches(ctx context.Context, accountID, gadsAccountID int64) {
	if h.pixels != nil && gadsAccountID > 0 {
		h.pixels.InvalidateByGoogleAdsAccount(ctx, gadsAccountID)
	}
	cache.BumpDashboardVersion(ctx, h.rdb, accountID)
}

type googleAdsAccountInput struct {
	Name              string                             `json:"name"`
	CustomerID        string                             `json:"customer_id"`
	CustomerName      string                             `json:"customer_name"`
	LoginCustomerID   string                             `json:"login_customer_id"`
	ConversionActions []domain.GoogleAdsConversionAction `json:"conversion_actions"`
	ReuseOAuthFromID  int64                              `json:"reuse_oauth_from_id"`
	Enabled           *bool                              `json:"enabled"`
}

func (h *AdminGoogleAdsAccounts) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	var (
		list []domain.GoogleAdsAccount
		err  error
	)
	if scope.IsOwner && scope.AccountID == 0 {
		list, err = h.accounts.List(r.Context())
	} else {
		list, err = h.accounts.ListByAccount(r.Context(), scope.AccountID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar contas Google Ads")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, a.PublicView())
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *AdminGoogleAdsAccounts) Get(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, a.PublicView())
}

func (h *AdminGoogleAdsAccounts) Create(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para gerenciar contas de anúncios")
		return
	}
	var in googleAdsAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "o nome da conexão é obrigatório")
		return
	}
	scope := TenantFromContext(r.Context())
	accID := scope.AccountID
	var oauthSrc *domain.GoogleAdsAccount
	if in.ReuseOAuthFromID > 0 {
		src, err := h.loadValidOAuthSource(r.Context(), scope, accID, in.ReuseOAuthFromID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		oauthSrc = src
		if scope.IsOwner && accID == 0 {
			accID = src.AccountID
		}
	}
	if scope.IsOwner && accID == 0 {
		accID = 1
	}
	a := &domain.GoogleAdsAccount{
		AccountID:         accID,
		Name:              in.Name,
		CustomerID:        strings.TrimSpace(in.CustomerID),
		CustomerName:      strings.TrimSpace(in.CustomerName),
		LoginCustomerID:   strings.TrimSpace(in.LoginCustomerID),
		ConversionActions: sanitizeConversionActions(in.ConversionActions),
		Enabled:           true,
	}
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
	if err := h.accounts.Create(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao criar conta Google Ads")
		return
	}
	if oauthSrc != nil {
		if err := h.applyOAuthSource(r.Context(), a, oauthSrc); err != nil {
			_ = h.accounts.Delete(r.Context(), a.ID)
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	h.invalidateAccountCaches(r.Context(), a.AccountID, a.ID)
	writeJSON(w, http.StatusCreated, a.PublicView())
}

func (h *AdminGoogleAdsAccounts) Update(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para gerenciar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	var in googleAdsAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		a.Name = name
	}
	a.CustomerID = strings.TrimSpace(in.CustomerID)
	a.CustomerName = strings.TrimSpace(in.CustomerName)
	if a.CustomerID == "" {
		a.CustomerName = ""
	}
	a.LoginCustomerID = strings.TrimSpace(in.LoginCustomerID)
	a.ConversionActions = sanitizeConversionActions(in.ConversionActions)
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
	if err := h.accounts.Update(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao atualizar conta Google Ads")
		return
	}
	h.invalidateAccountCaches(r.Context(), a.AccountID, a.ID)
	writeJSON(w, http.StatusOK, a.PublicView())
}

func (h *AdminGoogleAdsAccounts) Delete(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para gerenciar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	h.invalidateAccountCaches(r.Context(), a.AccountID, a.ID)
	if err := h.accounts.Delete(r.Context(), a.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir conta Google Ads")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *AdminGoogleAdsAccounts) loadAuthorizedAccount(w http.ResponseWriter, r *http.Request) (*domain.GoogleAdsAccount, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return nil, false
	}
	a, err := h.accounts.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conta Google Ads não encontrada")
			return nil, false
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar conta Google Ads")
		return nil, false
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && a.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a esta conta")
		return nil, false
	}
	return a, true
}
