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

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func (h *AdminMetaAccounts) OAuthStart(w http.ResponseWriter, r *http.Request) {
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
	h.state.Put(r.Context(), state, cache.OAuthStateEntry{
		MetaAccountID: a.ID,
		AccountID:     a.AccountID,
	})
	switchAccount := r.URL.Query().Get("switch_account") == "1"
	authURL, err := h.oauth.StartCentralOAuth(r.Context(), a.ID, state, switchAccount)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"url":   authURL,
		"state": state,
	})
}

func (h *AdminMetaAccounts) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errParam := q.Get("error"); errParam != "" {
		desc := q.Get("error_description")
		http.Redirect(w, r, h.frontResultURL("error", errParam+": "+desc, 0), http.StatusFound)
		return
	}
	ticketVec := strings.TrimSpace(q.Get("v"))
	if ticketVec == "" {
		http.Redirect(w, r, h.frontResultURL("error", "pacote de autorizacao ausente", 0), http.StatusFound)
		return
	}
	ticket, err := h.oauth.OpenCallbackTicket(ticketVec)
	if err != nil {
		http.Redirect(w, r, h.frontResultURL("error", "credencial invalida ou expirada", 0), http.StatusFound)
		return
	}
	entry, ok := h.state.Take(r.Context(), ticket.StateNonce)
	if !ok || entry.MetaAccountID == 0 {
		http.Redirect(w, r, h.frontResultURL("error", "sessao OAuth invalida ou expirada", 0), http.StatusFound)
		return
	}
	var expPtr *time.Time
	if ticket.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(ticket.ExpiresIn) * time.Second)
		expPtr = &t
	}
	if err := h.accounts.SetOAuthTokens(r.Context(), entry.MetaAccountID, ticket.VaultToken, ticket.UserID, ticket.Name, ticket.Email, ticket.PictureURL, expPtr); err != nil {
		slog.Error("save meta vault token", "error", err, "meta_account_id", entry.MetaAccountID)
		http.Redirect(w, r, h.frontResultURL("error", "erro ao salvar credencial Meta", entry.MetaAccountID), http.StatusFound)
		return
	}
	label := ticket.Name
	if label == "" {
		label = ticket.Email
	}
	http.Redirect(w, r, h.frontResultURL("success", label, entry.MetaAccountID), http.StatusFound)
}

func (h *AdminMetaAccounts) Disconnect(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	if err := h.accounts.ClearOAuthTokens(r.Context(), a.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao desconectar conta Meta")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"disconnected": true})
}

func (h *AdminMetaAccounts) copyOAuthFromSource(ctx context.Context, dst *domain.MetaAccount, sourceID int64) error {
	src, err := h.accounts.FindByID(ctx, sourceID)
	if err != nil || src == nil || src.AccountID != dst.AccountID || !src.IsConnected() {
		return errors.New("conta de origem invalida ou desconectada")
	}
	if err := h.accounts.SetOAuthTokens(ctx, dst.ID, src.VaultTokenEncrypted, src.OAuthUserID, src.OAuthName, src.OAuthEmail, src.OAuthPictureURL, src.TokenExpiresAt); err != nil {
		return errors.New("erro ao reutilizar conexao OAuth da Meta")
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

func (h *AdminMetaAccounts) ListAdAccounts(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	res, err := h.client.ListAdAccounts(r.Context(), a)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if oauthPic, _ := res["oauth_picture_url"].(string); oauthPic != "" && a.OAuthPictureURL == "" {
		_ = h.accounts.UpdatePictures(r.Context(), a.ID, oauthPic, "")
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminMetaAccounts) ListPages(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	res, err := h.client.ListPages(r.Context(), a)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminMetaAccounts) ListPixels(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	adAccID := strings.TrimSpace(r.URL.Query().Get("ad_account_id"))
	res, err := h.client.ListPixels(r.Context(), a, adAccID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminMetaAccounts) frontResultURL(status, message string, accountID int64) string {
	host := strings.TrimRight(h.frontHost, "/")
	q := "?meta_ads_oauth=" + status
	if accountID > 0 {
		q += "&account_id=" + strconv.FormatInt(accountID, 10)
	}
	if message != "" {
		q += "&message=" + urlEncode(message)
	}
	return host + "/integracoes/meta-ads" + q
}

