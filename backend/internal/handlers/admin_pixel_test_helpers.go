package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/tiktok"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func buildTestSendInput(r *http.Request, p *domain.Pixel, in testEventInput) (domain.SendEventInput, string) {
	eventID := "TEST_" + util.GenerateID10()
	clientUA := r.UserAgent()
	if clientUA == "" {
		clientUA = "TrackFind-TestEvent/1.0"
	}
	var valuePtr *float64
	if in.Value > 0 {
		v := in.Value
		valuePtr = &v
	}
	sourceURL := "https://" + r.Host + "/admin/test-event"
	if p.TrackingHost != "" {
		sourceURL = "https://" + p.TrackingHost + "/admin/test-event"
	}
	return domain.SendEventInput{
		PixelName: p.Name, EventName: in.EventName, EventID: eventID,
		EventTime: time.Now().Unix(), SourceURL: sourceURL,
		ContentID: p.Name, ContentName: p.Name, ContentType: "product",
		Value: valuePtr, Currency: in.Currency,
		ExternalID: "test-" + util.GenerateID10(), ClientIP: extractTestClientIP(r),
		ClientUserAgent: clientUA, Email: in.Email, Phone: in.Phone,
		FirstName: in.FirstName, LastName: in.LastName,
	}, eventID
}

func (h *AdminPixelTestEvent) resolveTestMetaOAuth(ctx context.Context, p *domain.Pixel) (*domain.MetaAccount, string) {
	if h.metaAccs == nil || h.metaOAuth == nil || p == nil {
		return nil, ""
	}
	if p.MetaAccountID != nil && *p.MetaAccountID > 0 {
		acc, err := h.metaAccs.FindByID(ctx, *p.MetaAccountID)
		if err != nil || acc == nil || acc.AccountID != p.AccountID {
			return nil, "a Conta Meta Ads vinculada a este pixel não foi encontrada"
		}
		if !acc.Enabled {
			return nil, "a Conta Meta Ads vinculada está pausada em Integrações > Meta Ads"
		}
		if !acc.IsConnected() {
			return nil, "a Conta Meta Ads vinculada está desconectada ou com token expirado; reconecte em Integrações > Meta Ads"
		}
		return acc, ""
	}
	acc, _ := h.metaAccs.FindFirstConnectedByAccount(ctx, p.AccountID)
	if acc != nil && (!acc.Enabled || !acc.IsConnected()) {
		return nil, ""
	}
	return acc, ""
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

func (h *AdminPixelTestEvent) sendTestMeta(
	ctx context.Context,
	p *domain.Pixel,
	oauthAcc *domain.MetaAccount,
	input domain.SendEventInput,
	overrideTestCode string,
) []testEventResult {
	client := meta.New(h.metaAPIVersion, h.metaPartner)
	out := make([]testEventResult, 0, len(p.PixelIDs))
	for i, pid := range p.PixelIDs {
		tok := ""
		if i < len(p.AccessTokens) {
			tok = strings.TrimSpace(p.AccessTokens[i])
		}
		testCode := overrideTestCode
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
			errStr = util.RedactSecrets(r.Err.Error())
		}
		out = append(out, testEventResult{
			Channel: "meta", PixelID: pid, StatusCode: r.StatusCode,
			UsedTestCode: testCode != "",
			Response:     util.RedactSecrets(r.Body), Error: errStr,
		})
	}
	return out
}

func (h *AdminPixelTestEvent) sendTestTikTok(
	ctx context.Context,
	p *domain.Pixel,
	input domain.SendEventInput,
	overrideTestCode string,
) []testEventResult {
	sub := *p
	codes := make([]string, len(sub.TikTokPixelIDs))
	for i := range sub.TikTokPixelIDs {
		codes[i] = overrideTestCode
	}
	sub.TikTokTestCodes = codes
	client := tiktok.New(h.tiktokPartner)
	rawResults := client.Send(ctx, &sub, input)
	out := make([]testEventResult, 0, len(rawResults))
	for i, r := range rawResults {
		errStr := ""
		if r.Err != nil {
			errStr = util.RedactSecrets(r.Err.Error())
		}
		used := i < len(codes) && codes[i] != ""
		out = append(out, testEventResult{
			Channel: "tiktok", PixelID: r.PixelID, StatusCode: r.StatusCode,
			UsedTestCode: used,
			Response:     util.RedactSecrets(r.Body), Error: errStr,
		})
	}
	return out
}
