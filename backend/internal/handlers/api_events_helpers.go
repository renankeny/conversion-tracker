package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func isServerSideHTTPUserAgent(ua string) bool {
	low := strings.ToLower(strings.TrimSpace(ua))
	if low == "" {
		return true
	}
	serverPrefixes := []string{
		"axios/", "curl/", "python-requests", "python-urllib", "aiohttp/",
		"go-http-client", "node-fetch", "undici", "postmanruntime", "insomnia/",
		"okhttp/", "java/", "apache-httpclient", "guzzlehttp", "ruby",
	}
	for _, p := range serverPrefixes {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

// upsertTrackedUserAPI persiste dados completos (PII, GeoIP, cookies e UTMs) para todos os eventos não-internos via API.
func (h *APIEvents) upsertTrackedUserAPI(
	ctx context.Context,
	req apiEventRequest,
	pixelName string,
	accountID int64,
	ip, ua, mappedURL, country, state, city, postal, email, phone string,
) *domain.TrackedUser {
	if domain.IsInternalEvent(req.EventName) {
		return nil
	}
	tu := &domain.TrackedUser{
		ExternalID:       req.ExternalID,
		IDWpp:            req.IDWpp,
		PixelName:        pixelName,
		AccountID:        accountID,
		FBP:              req.FBP,
		FBC:              req.FBC,
		GCLID:            req.GCLID,
		WBraid:           req.WBraid,
		GBraid:           req.GBraid,
		TTClid:           req.TTClid,
		TTP:              req.TTP,
		CtwaClid:         req.CtwaClid,
		GA4ClientID:      req.GA4ClientID,
		GA4SessionID:     req.GA4SessionID,
		ClientIP:         ip,
		ClientUserAgent:  ua,
		Country:          country,
		State:            state,
		City:             city,
		PostalCode:       postal,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Email:            email,
		Phone:            phone,
		URL:              mappedURL,
		Referrer:         req.Referrer,
		FirstUTMSource:   req.UTMSource,
		FirstUTMMedium:   req.UTMMedium,
		FirstUTMCampaign: req.UTMCampaign,
		FirstUTMContent:  req.UTMContent,
		FirstUTMTerm:     req.UTMTerm,
		LastUTMSource:    req.UTMSource,
		LastUTMMedium:    req.UTMMedium,
		LastUTMCampaign:  req.UTMCampaign,
		LastUTMContent:   req.UTMContent,
		LastUTMTerm:      req.UTMTerm,
	}
	_ = h.users.Upsert(ctx, tu)
	if phone != "" || email != "" {
		_, _ = h.users.ResolveAndLink(ctx, accountID, tu.ExternalID, phone, email)
	}
	return tu
}

func extractClientIP(r *http.Request) string {
	return clientIP(r)
}
