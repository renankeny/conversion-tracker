package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminMetaAccounts struct {
	accounts  *repo.MetaAccountRepo
	client    *metaoauth.Client
	oauth     metaoauth.Config
	state     *cache.OAuthStateStore
	rdb       *redis.Client
	frontHost string
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

func (h *AdminMetaAccounts) maybeBackfillPicture(ctx context.Context, a *domain.MetaAccount) {
	if !a.IsConnected() || a.EffectivePictureURL() != "" || h.client == nil {
		return
	}
	checkKey := fmt.Sprintf("tf:meta:pic_checked:%d", a.ID)
	if h.rdb != nil {
		if exists, _ := h.rdb.Exists(ctx, checkKey).Result(); exists > 0 {
			return
		}
		_ = h.rdb.Set(ctx, checkKey, "1", 6*time.Hour).Err()
	}
	res, err := h.client.ListAdAccounts(ctx, a)
	if err != nil {
		return
	}
	oauthPic, _ := res["oauth_picture_url"].(string)
	adPic := ""
	if items, ok := res["ad_accounts"].([]any); ok {
		for _, raw := range items {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			idStr, _ := m["id"].(string)
			accStr, _ := m["account_id"].(string)
			if a.AdAccountID != "" && (idStr == a.AdAccountID || "act_"+accStr == a.AdAccountID) {
				adPic, _ = m["picture_url"].(string)
				break
			}
		}
		if adPic == "" && len(items) == 1 {
			if m, ok := items[0].(map[string]any); ok {
				adPic, _ = m["picture_url"].(string)
			}
		}
	}
	if oauthPic != "" || adPic != "" {
		_ = h.accounts.UpdatePictures(ctx, a.ID, oauthPic, adPic)
		if oauthPic != "" {
			a.OAuthPictureURL = oauthPic
		}
		if adPic != "" {
			a.AdAccountPictureURL = adPic
		}
	}
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
		h.maybeBackfillPicture(r.Context(), &list[i])
		out = append(out, list[i].PublicView())
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *AdminMetaAccounts) Get(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	h.maybeBackfillPicture(r.Context(), a)
	writeJSON(w, http.StatusOK, a.PublicView())
}

func (h *AdminMetaAccounts) Create(w http.ResponseWriter, r *http.Request) {
	var in metaAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "nome e obrigatorio")
		return
	}
	scope := TenantFromContext(r.Context())
	accID := scope.AccountID
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
		DefaultPageName:       strings.TrimSpace(in.DefaultPageName),
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
	if in.ReuseOAuthFromID > 0 {
		if err := h.copyOAuthFromSource(r.Context(), a, in.ReuseOAuthFromID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusCreated, a.PublicView())
}

func (h *AdminMetaAccounts) Update(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	var in metaAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
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
	writeJSON(w, http.StatusOK, a.PublicView())
}

func (h *AdminMetaAccounts) Delete(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	if err := h.accounts.Delete(r.Context(), a.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir conta Meta Ads")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *AdminMetaAccounts) loadAuthorizedAccount(w http.ResponseWriter, r *http.Request) (*domain.MetaAccount, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalido")
		return nil, false
	}
	a, err := h.accounts.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conta Meta Ads nao encontrada")
			return nil, false
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar conta Meta Ads")
		return nil, false
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && a.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return nil, false
	}
	return a, true
}
