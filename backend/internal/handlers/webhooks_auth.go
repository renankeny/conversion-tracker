package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type webhookAuthAttemptKey struct{}

type webhookAuthAttempt struct {
	platform   string
	rawBody    []byte
	hmacSHA256 string
	hmacSHA1   string
	candidates []string
}

func constantTimeEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func constantTimeEqualFold(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(strings.ToLower(a)), []byte(strings.ToLower(b))) == 1
}

func collectRequestTokenCandidates(r *http.Request, payloadCandidates []string) []string {
	out := make([]string, 0, len(payloadCandidates)+15)
	for _, c := range payloadCandidates {
		if t := strings.TrimSpace(c); t != "" {
			out = append(out, t)
		}
	}
	if r == nil {
		return out
	}
	headers := []string{
		r.Header.Get("X-Webhook-Token"),
		r.Header.Get("X-Platform-Token"),
		r.Header.Get("X-Hotmart-Hottok"),
		r.Header.Get("X-Kiwify-Signature"),
		r.Header.Get("X-Lastlink-Token"),
		r.Header.Get("X-Hubla-Token"),
		r.Header.Get("X-Greenn-Token"),
		r.Header.Get("X-Eduzz-Token"),
		r.Header.Get("X-Pepper-Token"),
		r.Header.Get("X-Api-Key"),
		stripAuthPrefix(r.Header.Get("Authorization")),
	}
	for _, hVal := range headers {
		if t := strings.TrimSpace(hVal); t != "" {
			out = append(out, t)
		}
	}
	q := r.URL.Query()
	for _, qKey := range []string{"token", "secret", "hottok", "signature"} {
		if t := strings.TrimSpace(q.Get(qKey)); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (h *Webhooks) resolveWebhookAccountID(ctx context.Context, r *http.Request, accountID int64) int64 {
	if accountID > 0 {
		return accountID
	}
	if r != nil && h.pixels != nil {
		if pq := webhookPixelQuery(r); pq != "" {
			if p, err := h.pixels.FindByNameOrToken(ctx, pq); err == nil && p != nil {
				return p.AccountID
			}
		}
	}
	return 0
}

func matchWebhookAttemptAgainstSecret(secret string, att webhookAuthAttempt) bool {
	if secret == "" {
		return true
	}
	if att.hmacSHA256 != "" && len(att.rawBody) > 0 {
		cand := strings.TrimSpace(att.hmacSHA256)
		expectedHex := util.GenerateHMACSHA256(att.rawBody, secret)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(att.rawBody)
		expectedB64 := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		if constantTimeEqualFold(cand, expectedHex) || constantTimeEqual(cand, expectedB64) {
			return true
		}
	}

	if att.hmacSHA1 != "" && len(att.rawBody) > 0 {
		expectedSHA1 := util.GenerateHMACSHA1(att.rawBody, secret)
		if constantTimeEqualFold(strings.TrimSpace(att.hmacSHA1), expectedSHA1) {
			return true
		}
	}

	for _, cand := range att.candidates {
		if constantTimeEqual(cand, secret) {
			return true
		}
	}
	return false
}

func (h *Webhooks) verifyAttemptAgainstAccount(ctx context.Context, accountID int64, att webhookAuthAttempt) bool {
	if accountID <= 0 || h.tokens == nil {
		return true
	}
	tokenInfo, err := h.tokens.FindByPlatform(ctx, accountID, att.platform)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		// Fail-closed: erro de banco não pode desligar a autenticação.
		slog.Error("platform token lookup failed", "platform", att.platform, "account_id", accountID, "error", err)
		return false
	}
	if tokenInfo == nil || tokenInfo.TokenSecret == "" {
		return true
	}
	if matchWebhookAttemptAgainstSecret(tokenInfo.TokenSecret, att) {
		return true
	}
	slog.Warn("platform token mismatch", "platform", att.platform, "account_id", accountID)
	return false
}

func (h *Webhooks) verifyDeferredWebhookAuth(ctx context.Context, accountID int64) bool {
	att, ok := ctx.Value(webhookAuthAttemptKey{}).(webhookAuthAttempt)
	if !ok {
		return true
	}
	return h.verifyAttemptAgainstAccount(ctx, accountID, att)
}

// ValidatePlatformToken verifica em tempo constante se o token da plataforma bate com a conta.
// Se accountID == 0 e não houver ?pixel= na URL, registra a tentativa no contexto da request
// para validação obrigatória assim que o pixel/usuário for resolvido.
func (h *Webhooks) ValidatePlatformToken(ctx context.Context, r *http.Request, accountID int64, platform string, payloadCandidates ...string) bool {
	att := webhookAuthAttempt{
		platform:   platform,
		candidates: collectRequestTokenCandidates(r, payloadCandidates),
	}
	effectiveAcc := h.resolveWebhookAccountID(ctx, r, accountID)
	if effectiveAcc > 0 {
		return h.verifyAttemptAgainstAccount(ctx, effectiveAcc, att)
	}
	if r != nil {
		*r = *r.WithContext(context.WithValue(r.Context(), webhookAuthAttemptKey{}, att))
	}
	return true
}

// validateHMACOrToken valida assinaturas HMAC (SHA256 hex/base64 ou SHA1 hex) ou tokens em tempo constante.
func (h *Webhooks) validateHMACOrToken(
	ctx context.Context,
	r *http.Request,
	accountID int64,
	platform string,
	rawBody []byte,
	hmacSHA256Candidate string,
	hmacSHA1Candidate string,
	payloadTokens ...string,
) bool {
	att := webhookAuthAttempt{
		platform:   platform,
		rawBody:    rawBody,
		hmacSHA256: hmacSHA256Candidate,
		hmacSHA1:   hmacSHA1Candidate,
		candidates: collectRequestTokenCandidates(r, payloadTokens),
	}
	effectiveAcc := h.resolveWebhookAccountID(ctx, r, accountID)
	if effectiveAcc > 0 {
		return h.verifyAttemptAgainstAccount(ctx, effectiveAcc, att)
	}
	if r != nil {
		*r = *r.WithContext(context.WithValue(r.Context(), webhookAuthAttemptKey{}, att))
	}
	return true
}
