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

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminGoogleAdsAccounts struct {
	accounts  *repo.GoogleAdsAccountRepo
	client    *googleads.Client
	oauth     googleads.OAuthConfig
	state     *cache.OAuthStateStore
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
	h.state = cache.NewOAuthStateStore(rdb)
}

type googleAdsAccountInput struct {
	Name              string                             `json:"name"`
	CustomerID        string                             `json:"customer_id"`
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
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, a.PublicView())
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *AdminGoogleAdsAccounts) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	a, err := h.accounts.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conta nao encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && a.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	writeJSON(w, http.StatusOK, a.PublicView())
}

func (h *AdminGoogleAdsAccounts) Create(w http.ResponseWriter, r *http.Request) {
	var in googleAdsAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "nome e obrigatorio")
		return
	}

	scope := TenantFromContext(r.Context())
	a := &domain.GoogleAdsAccount{
		AccountID:         scope.AccountID,
		Name:              in.Name,
		CustomerID:        strings.TrimSpace(in.CustomerID),
		LoginCustomerID:   strings.TrimSpace(in.LoginCustomerID),
		ConversionActions: sanitizeConversionActions(in.ConversionActions),
		Enabled:           true,
	}
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
	if scope.IsOwner && a.AccountID == 0 {
		a.AccountID = 1
	}
	if err := h.accounts.Create(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao criar conta")
		return
	}
	if in.ReuseOAuthFromID > 0 {
		if err := h.copyOAuthFromSource(r.Context(), a, in.ReuseOAuthFromID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusCreated, a.PublicView())
}

func (h *AdminGoogleAdsAccounts) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	a, err := h.accounts.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conta nao encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && a.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}

	var in googleAdsAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		a.Name = name
	}
	a.CustomerID = strings.TrimSpace(in.CustomerID)
	a.LoginCustomerID = strings.TrimSpace(in.LoginCustomerID)
	a.ConversionActions = sanitizeConversionActions(in.ConversionActions)
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
	if err := h.accounts.Update(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao atualizar conta")
		return
	}
	writeJSON(w, http.StatusOK, a.PublicView())
}

func (h *AdminGoogleAdsAccounts) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	a, err := h.accounts.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conta nao encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && a.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	if err := h.accounts.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir conta")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *AdminGoogleAdsAccounts) copyOAuthFromSource(ctx context.Context, dst *domain.GoogleAdsAccount, sourceID int64) error {
	src, err := h.accounts.FindByID(ctx, sourceID)
	if err != nil || src == nil || src.AccountID != dst.AccountID || !src.IsConnected() {
		return errors.New("conta Google de origem invalida ou desconectada")
	}
	if err := h.accounts.SetTokens(ctx, dst.ID, src.RefreshToken, src.OAuthEmail, src.OAuthPictureURL); err != nil {
		return errors.New("erro ao reutilizar conexao OAuth do Google Ads")
	}
	dst.RefreshToken = src.RefreshToken
	dst.OAuthEmail = src.OAuthEmail
	dst.OAuthPictureURL = src.OAuthPictureURL
	return nil
}

func (h *AdminGoogleAdsAccounts) ReuseOAuth(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	var body struct {
		SourceAccountID int64 `json:"source_account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SourceAccountID <= 0 {
		writeError(w, http.StatusBadRequest, "source_account_id obrigatorio")
		return
	}
	if err := h.copyOAuthFromSource(r.Context(), a, body.SourceAccountID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a.PublicView())
}
