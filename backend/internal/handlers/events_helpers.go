package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/traffic"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

// upsertTrackedUser cria/atualiza o tracked user de acordo com o eventType,
// incluindo UpdateUserData (tf('updateUser')) e eventos customizados.
func (h *Events) upsertTrackedUser(
	ctx context.Context,
	req sendRequest,
	pixelName string,
	accountID int64,
	ip, ua, mappedURL, country, state, city, postal, email, phone, ga4ClientID, ga4SessionID string,
	crmEnabled bool,
) error {
	isWebdriver := req.Webdriver != nil && *req.Webdriver
	trafficRes := traffic.Classify(ua, req.Referrer, isWebdriver)
	rulesJSON, _ := json.Marshal(trafficRes.Rules)

	pageviewInc := 0
	checkoutInc := 0
	switch req.EventType {
	case domain.EventPageView, domain.EventViewContent:
		pageviewInc = 1
	case domain.EventInitiateCheckout:
		checkoutInc = 1
	}

	tu := &domain.TrackedUser{
		ExternalID:       req.UserID,
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
		GA4ClientID:      ga4ClientID,
		GA4SessionID:     ga4SessionID,
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
		FirstURL:         mappedURL,
		FirstUTMSource:   util.StripUserIDFromUTM(req.UTMSource),
		FirstUTMMedium:   req.UTMMedium,
		FirstUTMCampaign: req.UTMCampaign,
		FirstUTMContent:  req.UTMContent,
		FirstUTMTerm:     req.UTMTerm,
		PageViewCount:    pageviewInc,
		CheckoutCount:    checkoutInc,
		Referrer:         req.Referrer,
		TrafficType:      trafficRes.Type,
		TrafficScore:     trafficRes.Score,
		TrafficCategory:  trafficRes.Category,
		TrafficRules:     json.RawMessage(rulesJSON),
	}

	err := h.users.Upsert(ctx, tu)
	h.linkIdentityBestEffort(ctx, accountID, tu)

	if err == nil && req.EventType == domain.EventLead && tu.ID > 0 && crmEnabled {
		_ = h.crmRepo.EnsureDefaultStages(ctx, accountID)
		if firstStage, errStage := h.crmRepo.GetFirstStage(ctx, accountID); errStage == nil && firstStage != nil {
			if ensureErr := h.crmRepo.EnsureLead(ctx, accountID, tu.ID, firstStage.ID); ensureErr == nil && h.rdb != nil {
				wshub.PublishCustom(ctx, h.rdb, accountID, "crm_updated", map[string]any{"action": "lead_ensured"})
			}
		}
	}
	return err
}

func (h *Events) linkIdentityBestEffort(ctx context.Context, accountID int64, tu *domain.TrackedUser) {
	if tu.Phone == "" && tu.Email == "" {
		return
	}
	if _, err := h.users.ResolveAndLink(ctx, accountID, tu.ExternalID, tu.Phone, tu.Email); err != nil {
		slog.Warn("resolve identity link failed", "err", err, "external_id", tu.ExternalID)
	}
}

func (h *Events) isDuplicateInMemory(key string, ttl time.Duration) bool {
	now := time.Now()
	if val, ok := h.recentDedup.Load(key); ok {
		if ts, valid := val.(time.Time); valid && now.Sub(ts) < ttl {
			return true
		}
	}
	h.recentDedup.Store(key, now)
	if now.UnixNano()%128 == 0 {
		h.recentDedup.Range(func(k, v any) bool {
			if ts, valid := v.(time.Time); !valid || now.Sub(ts) > 10*time.Minute {
				h.recentDedup.Delete(k)
			}
			return true
		})
	}
	return false
}

// clientIP usa r.RemoteAddr já resolvido pelo middleware TrustedRealIP
// (XFF só é aceito vindo de proxy confiável). Compatível com IPv6.
func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return strings.Trim(strings.TrimSpace(r.RemoteAddr), "[]")
}
