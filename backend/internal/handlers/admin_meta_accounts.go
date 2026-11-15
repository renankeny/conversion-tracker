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
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminMetaAccounts struct {
	accounts    *repo.MetaAccountRepo
	pixels      *repo.PixelRepo
	client      *metaoauth.Client
	oauth       metaoauth.Config
	state       *cache.OAuthStateStore
	rdb         *redis.Client
	frontHost   string
	metaPartner string
}

func NewAdminMetaAccounts(accounts *repo.MetaAccountRepo, client *metaoauth.Client, oauth metaoauth.Config, frontHost string) *AdminMetaAccounts {
	return &AdminMetaAccounts{
		accounts:  accounts,
		client:    client,
		oauth:     oauth,
		state:     cache.NewOAuthStateStore(nil),
		frontHost: frontHost,
	}
}

func (h *AdminMetaAccounts) SetRedis(rdb *redis.Client) {
	h.rdb = rdb
	h.state = cache.NewOAuthStateStore(rdb)
}

func (h *AdminMetaAccounts) SetPixels(pixels *repo.PixelRepo) {
	h.pixels = pixels
}

func (h *AdminMetaAccounts) SetPartnerAgent(partner string) {
	h.metaPartner = partner
}

func (h *AdminMetaAccounts) invalidateAccountCaches(ctx context.Context, accountID, metaAccountID int64) {
	if h.pixels != nil && metaAccountID > 0 {
		h.pixels.InvalidateByMetaAccount(ctx, metaAccountID)
	}
	cache.BumpDashboardVersion(ctx, h.rdb, accountID)
}

type metaAccountInput struct {
	Name                  string `json:"name"`
	AdAccountID           string `json:"ad_account_id"`
	AdAccountName         string `json:"ad_account_name"`
	AdAccountPictureURL   string `json:"ad_account_picture_url"`
	DefaultPageID         string `json:"default_page_id"`
	DefaultPageName       string `json:"default_page_name"`
	DefaultPagePictureURL string `json:"default_page_picture_url"`
	ReuseOAuthFromID      int64  `json:"reuse_oauth_from_id"`
	Enabled               *bool  `json:"enabled"`
}

func (h *AdminMetaAccounts) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	var (
		list []domain.MetaAccount
		err  error
	)
	if scope.IsOwner && scope.AccountID == 0 {
		list, err = h.accounts.List(r.Context())
	} else {
		list, err = h.accounts.ListByAccount(r.Context(), scope.AccountID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar contas Meta Ads")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, list[i].PublicView())
	}
	h.backfillPicturesAsync(list)
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *AdminMetaAccounts) Get(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, a.PublicView())
}

func (h *AdminMetaAccounts) Create(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para gerenciar contas de anúncios")
		return
	}
	var in metaAccountInput
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
	var oauthSrc *domain.MetaAccount
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
	a := &domain.MetaAccount{
		AccountID:             accID,
		Name:                  in.Name,
		AdAccountID:           strings.TrimSpace(in.AdAccountID),
		AdAccountName:         strings.TrimSpace(in.AdAccountName),
		AdAccountPictureURL:   strings.TrimSpace(in.AdAccountPictureURL),
		DefaultPageID:         strings.TrimSpace(in.DefaultPageID),
		DefaultPageName:       stringTrim(in.DefaultPageName),
		DefaultPagePictureURL: strings.TrimSpace(in.DefaultPagePictureURL),
		Enabled:               true,
	}
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
	if err := h.accounts.Create(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao criar conta Meta Ads")
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

func (h *AdminMetaAccounts) Update(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para gerenciar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	var in metaAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		a.Name = name
	}
	a.AdAccountID = strings.TrimSpace(in.AdAccountID)
	a.AdAccountName = strings.TrimSpace(in.AdAccountName)
	if pic := strings.TrimSpace(in.AdAccountPictureURL); pic != "" || a.AdAccountID == "" {
		a.AdAccountPictureURL = pic
	}
	a.DefaultPageID = strings.TrimSpace(in.DefaultPageID)
	a.DefaultPageName = strings.TrimSpace(in.DefaultPageName)
	if ppic := strings.TrimSpace(in.DefaultPagePictureURL); ppic != "" || a.DefaultPageID == "" {
		a.DefaultPagePictureURL = ppic
	}
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
	if err := h.accounts.Update(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao atualizar conta Meta Ads")
		return
	}
	h.invalidateAccountCaches(r.Context(), a.AccountID, a.ID)
	writeJSON(w, http.StatusOK, a.PublicView())
}

func (h *AdminMetaAccounts) Delete(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusInternalServerError, "erro ao excluir conta Meta Ads")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *AdminMetaAccounts) loadAuthorizedAccount(w http.ResponseWriter, r *http.Request) (*domain.MetaAccount, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return nil, false
	}
	a, err := h.accounts.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conta Meta Ads não encontrada")
			return nil, false
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar conta Meta Ads")
		return nil, false
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && a.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a esta conta")
		return nil, false
	}
	return a, true
}

func stringTrim(s string) string { return strings.TrimSpace(s) }
