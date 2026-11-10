package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func (h *AdminPixelTestEvent) resolveTestMetaOAuth(ctx context.Context, p *domain.Pixel) *domain.MetaAccount {
	if h.metaAccs == nil || h.metaOAuth == nil || p == nil {
		return nil
	}
	var acc *domain.MetaAccount
	if p.MetaAccountID != nil && *p.MetaAccountID > 0 {
		acc, _ = h.metaAccs.FindByID(ctx, *p.MetaAccountID)
	}
	if acc == nil || !acc.IsConnected() {
		acc, _ = h.metaAccs.FindFirstConnectedByAccount(ctx, p.AccountID)
	}
	if acc != nil && (!acc.Enabled || !acc.IsConnected()) {
		return nil
	}
	return acc
}

func hasNonEmptyToken(tokens []string) bool {
	for _, t := range tokens {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}

func extractTestClientIP(r *http.Request) string {
	ip := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	if ip == "" {
		ip = strings.TrimSpace(r.Header.Get("X-Real-IP"))
	}
	if ip == "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			return host
		}
		return r.RemoteAddr
	}
	return ip
}

func (h *AdminPixelTestEvent) sendTestMeta(ctx context.Context, p *domain.Pixel, oauthAcc *domain.MetaAccount, input domain.SendEventInput) []testEventResult {
	client := meta.New(h.metaAPIVersion, h.metaPartner)
	out := make([]testEventResult, 0, len(p.PixelIDs))
	for i, pid := range p.PixelIDs {
		tok, testCode := "", ""
		if i < len(p.AccessTokens) {
			tok = strings.TrimSpace(p.AccessTokens[i])
		}
		if i < len(p.TestCodes) {
			testCode = p.TestCodes[i]
		}
		var r meta.SendResult
		if tok != "" || oauthAcc == nil {
			sub := *p
			sub.PixelIDs, sub.AccessTokens, sub.TestCodes = []string{pid}, []string{tok}, []string{testCode}
			if sent := client.Send(ctx, &sub, input); len(sent) > 0 {
				r = sent[0]
			}
		} else if rawBody, berr := meta.BuildCAPIPayload(input, testCode, h.metaPartner); berr != nil {
			r = meta.SendResult{PixelID: pid, Err: berr}
		} else {
			code, body, perr := h.metaOAuth.SendCAPIEvent(ctx, oauthAcc, pid, rawBody)
			r = meta.SendResult{PixelID: pid, StatusCode: code, Body: body, Err: perr}
			if perr == nil && code >= 300 {
				r.Err = fmt.Errorf("meta oauth capi returned %d: %s", code, body)
			}
		}
		errStr := ""
		if r.Err != nil {
			errStr = r.Err.Error()
		}
		out = append(out, testEventResult{
			Channel: "meta", PixelID: pid, StatusCode: r.StatusCode,
			Response: util.RedactSecrets(r.Body), Error: errStr,
		})
	}
	return out
}

