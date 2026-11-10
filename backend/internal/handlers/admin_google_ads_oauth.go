package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// OAuthStart solicita ao gateway central OAuth a URL assinada do Google OAuth.
func (h *AdminGoogleAdsAccounts) OAuthStart(w http.ResponseWriter, r *http.Request) {
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

	stateBytes := make([]byte, 24)
	if _, err := rand.Read(stateBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "rand error")
		return
	}
	state := hex.EncodeToString(stateBytes)
	h.state.Put(r.Context(), state, cache.OAuthStateEntry{
		GoogleAdsAccountID: a.ID,
		AccountID:          a.AccountID,
	})

	authURL, err := h.oauth.StartCentralOAuth(r.Context(), a.ID, state)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"url":   authURL,
		"state": state,
	})
}

// OAuthCallback recebe o pacote criptografado (?v=...) redirecionado pelo gateway central OAuth.
func (h *AdminGoogleAdsAccounts) OAuthCallback(w http.ResponseWriter, r *http.Request) {
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
	if !ok {
		http.Redirect(w, r, h.frontResultURL("error", "state invalido ou expirado", 0), http.StatusFound)
		return
	}

	if err := h.accounts.SetTokens(r.Context(), entry.GoogleAdsAccountID, ticket.VaultToken, ticket.Email, ticket.PictureURL); err != nil {
		slog.Error("save google ads vault token", "err", err, "google_ads_account_id", entry.GoogleAdsAccountID)
		http.Redirect(w, r, h.frontResultURL("error", "erro ao salvar credenciais", entry.GoogleAdsAccountID), http.StatusFound)
		return
	}

	http.Redirect(w, r, h.frontResultURL("success", ticket.Email, entry.GoogleAdsAccountID), http.StatusFound)
}

// ListCustomers consulta via gateway central as contas Google Ads e subcontas MCC acessiveis.
func (h *AdminGoogleAdsAccounts) ListCustomers(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	res, err := h.client.ListCustomers(r.Context(), a)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if pic, _ := res["oauth_picture_url"].(string); pic != "" && a.OAuthPictureURL == "" {
		_ = h.accounts.UpdateOAuthPicture(r.Context(), a.ID, pic)
	}
	writeJSON(w, http.StatusOK, res)
}

// ListConversionActions consulta via gateway central as Conversion Actions da conta selecionada.
func (h *AdminGoogleAdsAccounts) ListConversionActions(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	cid := r.URL.Query().Get("customer_id")
	lcid := r.URL.Query().Get("login_customer_id")
	res, err := h.client.ListConversionActions(r.Context(), a, cid, lcid)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminGoogleAdsAccounts) loadAuthorizedAccount(w http.ResponseWriter, r *http.Request) (*domain.GoogleAdsAccount, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return nil, false
	}
	a, err := h.accounts.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conta nao encontrada")
			return nil, false
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return nil, false
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && a.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return nil, false
	}
	return a, true
}

// Disconnect apaga refresh_token + email da conta.
func (h *AdminGoogleAdsAccounts) Disconnect(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	if err := h.accounts.ClearTokens(r.Context(), a.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao desconectar")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"disconnected": true})
}

func (h *AdminGoogleAdsAccounts) frontResultURL(status, message string, accountID int64) string {
	host := strings.TrimRight(h.frontHost, "/")
	q := "?google_ads_oauth=" + status
	if accountID > 0 {
		q += "&account_id=" + strconv.FormatInt(accountID, 10)
	}
	if message != "" {
		q += "&message=" + urlEncode(message)
	}
	return host + "/integracoes/google-ads" + q
}

func urlEncode(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			out = append(out, c)
			continue
		}
		out = append(out, '%')
		out = append(out, hexChar(c>>4))
		out = append(out, hexChar(c&0x0f))
	}
	return string(out)
}

func hexChar(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + n - 10
}

func sanitizeConversionActions(in []domain.GoogleAdsConversionAction) []domain.GoogleAdsConversionAction {
	out := make([]domain.GoogleAdsConversionAction, 0, len(in))
	for _, ca := range in {
		ev := strings.TrimSpace(ca.EventType)
		id := strings.TrimSpace(ca.ConversionActionID)
		if ev == "" || id == "" {
			continue
		}
		out = append(out, domain.GoogleAdsConversionAction{
			EventType:          ev,
			ConversionActionID: id,
			Name:               strings.TrimSpace(ca.Name),
			Enabled:            ca.Enabled,
		})
	}
	return out
}
