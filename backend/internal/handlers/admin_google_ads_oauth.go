package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
)

func (h *AdminGoogleAdsAccounts) OAuthStart(w http.ResponseWriter, r *http.Request) {
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
		GoogleAdsAccountID: a.ID,
		AccountID:          a.AccountID,
		Popup:              popup,
	})

	switchAccount := r.URL.Query().Get("switch_account") == "1"
	authURL, err := h.oauth.StartCentralOAuth(r.Context(), a.ID, state, switchAccount)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"url": authURL, "state": state})
}

func (h *AdminGoogleAdsAccounts) OAuthCallback(w http.ResponseWriter, r *http.Request) {
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
	if !ok || entry.GoogleAdsAccountID == 0 {
		http.Redirect(w, r, h.frontResultURL("error", "sessão OAuth inválida ou expirada", 0, popupHint), http.StatusFound)
		return
	}
	if entry.Popup {
		popupHint = true
	}

	prev, _ := h.accounts.FindByID(r.Context(), entry.GoogleAdsAccountID)
	if prev != nil && prev.OAuthEmail != "" && ticket.Email != "" && !strings.EqualFold(prev.OAuthEmail, ticket.Email) {
		_ = h.accounts.ClearTokens(r.Context(), entry.GoogleAdsAccountID)
	}

	if err := h.accounts.SetTokens(r.Context(), entry.GoogleAdsAccountID, ticket.VaultToken, ticket.Email, ticket.PictureURL); err != nil {
		slog.Error("save google ads vault token", "error", err, "google_ads_account_id", entry.GoogleAdsAccountID)
		http.Redirect(w, r, h.frontResultURL("error", "erro ao salvar credenciais Google Ads", entry.GoogleAdsAccountID, popupHint), http.StatusFound)
		return
	}
	h.invalidateAccountCaches(r.Context(), entry.AccountID, entry.GoogleAdsAccountID)

	http.Redirect(w, r, h.frontResultURL("success", ticket.Email, entry.GoogleAdsAccountID, popupHint), http.StatusFound)
}

func (h *AdminGoogleAdsAccounts) Disconnect(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para desconectar contas de anúncios")
		return
	}
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	if err := h.accounts.ClearTokens(r.Context(), a.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao desconectar conta Google Ads")
		return
	}
	h.invalidateAccountCaches(r.Context(), a.AccountID, a.ID)
	writeJSON(w, http.StatusOK, map[string]any{"disconnected": true})
}

func (h *AdminGoogleAdsAccounts) frontResultURL(status, message string, accountID int64, popup bool) string {
	host := strings.TrimRight(h.frontHost, "/")
	if popup {
		q := "?channel=google_ads&status=" + status
		if accountID > 0 {
			q += "&account_id=" + strconv.FormatInt(accountID, 10)
		}
		if message != "" {
			q += "&message=" + urlEncode(message)
		}
		return host + "/oauth-complete" + q
	}
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
		out = append(out, '%', hexChar(c>>4), hexChar(c&0x0f))
	}
	return string(out)
}

func hexChar(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + n - 10
}
