package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func (h *AdminMetaAccounts) OAuthStart(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para conectar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	stateBytes := make([]byte, 24)
	if _, err := rand.Read(stateBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar estado OAuth")
		return
	}
	state := hex.EncodeToString(stateBytes)
	popup := r.URL.Query().Get("popup") == "1"
	h.state.Put(r.Context(), state, cache.OAuthStateEntry{
		MetaAccountID: a.ID,
		AccountID:     a.AccountID,
		Popup:         popup,
	})
	switchAccount := r.URL.Query().Get("switch_account") == "1"
	authURL, err := h.oauth.StartCentralOAuth(r.Context(), a.ID, state, switchAccount)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": authURL, "state": state})
}

func (h *AdminMetaAccounts) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	popupHint := q.Get("popup") == "1"
	if st := strings.TrimSpace(q.Get("state")); st != "" && q.Get("error") != "" {
		if entry, ok := h.state.Take(r.Context(), st); ok && entry.Popup {
			popupHint = true
		}
	}
	if errParam := q.Get("error"); errParam != "" {
		msg := errParam
		if desc := q.Get("error_description"); desc != "" {
			msg += ": " + desc
		}
		http.Redirect(w, r, h.frontResultURL("error", msg, 0, popupHint), http.StatusFound)
		return
	}
	ticketVec := strings.TrimSpace(q.Get("v"))
	if ticketVec == "" {
		http.Redirect(w, r, h.frontResultURL("error", "pacote de autorização ausente", 0, popupHint), http.StatusFound)
		return
	}
	ticket, err := h.oauth.OpenCallbackTicket(ticketVec)
	if err != nil {
		http.Redirect(w, r, h.frontResultURL("error", "credencial inválida ou expirada", 0, popupHint), http.StatusFound)
		return
	}
	entry, ok := h.state.Take(r.Context(), ticket.StateNonce)
	if !ok || entry.MetaAccountID == 0 {
		http.Redirect(w, r, h.frontResultURL("error", "sessão OAuth inválida ou expirada", 0, popupHint), http.StatusFound)
		return
	}
	if entry.Popup {
		popupHint = true
	}
	prev, _ := h.accounts.FindByID(r.Context(), entry.MetaAccountID)
	if prev != nil && prev.OAuthUserID != "" && ticket.UserID != "" && prev.OAuthUserID != ticket.UserID {
		_ = h.accounts.ClearOAuthTokens(r.Context(), entry.MetaAccountID)
	}
	var expPtr *time.Time
	if ticket.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(ticket.ExpiresIn) * time.Second)
		expPtr = &t
	}
	if err := h.accounts.SetOAuthTokens(r.Context(), entry.MetaAccountID, ticket.VaultToken, ticket.UserID, ticket.Name, ticket.Email, ticket.PictureURL, expPtr); err != nil {
		slog.Error("save meta vault token", "error", err, "meta_account_id", entry.MetaAccountID)
		http.Redirect(w, r, h.frontResultURL("error", "erro ao salvar credencial Meta", entry.MetaAccountID, popupHint), http.StatusFound)
		return
	}
	h.invalidateAccountCaches(r.Context(), entry.AccountID, entry.MetaAccountID)
	label := ticket.Name
	if label == "" {
		label = ticket.Email
	}
	http.Redirect(w, r, h.frontResultURL("success", label, entry.MetaAccountID, popupHint), http.StatusFound)
}

func (h *AdminMetaAccounts) Disconnect(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para desconectar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	if err := h.accounts.ClearOAuthTokens(r.Context(), a.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao desconectar conta Meta")
		return
	}
	h.invalidateAccountCaches(r.Context(), a.AccountID, a.ID)
	writeJSON(w, http.StatusOK, map[string]any{"disconnected": true})
}

func (h *AdminMetaAccounts) loadValidOAuthSource(ctx context.Context, scope TenantScope, dstAccountID, sourceID int64) (*domain.MetaAccount, error) {
	src, err := h.accounts.FindByID(ctx, sourceID)
	if err != nil || src == nil || !src.IsConnected() {
		return nil, errors.New("conta Meta de origem inválida ou desconectada")
	}
	if !scope.IsOwner && src.AccountID != dstAccountID {
		return nil, errors.New("sem acesso à conta Meta de origem")
	}
	return src, nil
}

func (h *AdminMetaAccounts) applyOAuthSource(ctx context.Context, dst *domain.MetaAccount, src *domain.MetaAccount) error {
	if dst.OAuthUserID != "" && src.OAuthUserID != "" && dst.OAuthUserID != src.OAuthUserID {
		_ = h.accounts.ClearOAuthTokens(ctx, dst.ID)
		dst.AdAccountID, dst.AdAccountName, dst.AdAccountPictureURL = "", "", ""
		dst.DefaultPageID, dst.DefaultPageName, dst.DefaultPagePictureURL = "", "", ""
	}
	if err := h.accounts.SetOAuthTokens(ctx, dst.ID, src.VaultTokenEncrypted, src.OAuthUserID, src.OAuthName, src.OAuthEmail, src.OAuthPictureURL, src.TokenExpiresAt); err != nil {
		return errors.New("erro ao reutilizar conexão OAuth da Meta")
	}
	dst.VaultTokenEncrypted = src.VaultTokenEncrypted
	dst.OAuthUserID = src.OAuthUserID
	dst.OAuthName = src.OAuthName
	dst.OAuthEmail = src.OAuthEmail
	dst.OAuthPictureURL = src.OAuthPictureURL
	dst.TokenExpiresAt = src.TokenExpiresAt
	return nil
}

func (h *AdminMetaAccounts) ReuseOAuth(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminMetaAccounts) frontResultURL(status, message string, accountID int64, popup bool) string {
	host := strings.TrimRight(h.frontHost, "/")
	if popup {
		q := "?channel=meta&status=" + status
		if accountID > 0 {
			q += "&account_id=" + strconv.FormatInt(accountID, 10)
		}
		if message != "" {
			q += "&message=" + urlEncode(message)
		}
		return host + "/oauth-complete" + q
	}
	q := "?meta_ads_oauth=" + status
	if accountID > 0 {
		q += "&account_id=" + strconv.FormatInt(accountID, 10)
	}
	if message != "" {
		q += "&message=" + urlEncode(message)
	}
	return host + "/integracoes/meta-ads" + q
}
