package handlers

import (
	"context"
	"net/http"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// upsertTrackedUserAPI persiste dados do usuário para eventos via API.
func (h *APIEvents) upsertTrackedUserAPI(
	ctx context.Context,
	req apiEventRequest,
	pixelName string,
	accountID int64,
	ip, ua, mappedURL, country, state, city, postal, email, phone string,
) *domain.TrackedUser {
	switch req.EventName {
	case domain.EventPageView, domain.EventViewContent:
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
			FirstUTMSource:   req.UTMSource,
			FirstUTMMedium:   req.UTMMedium,
			FirstUTMCampaign: req.UTMCampaign,
			FirstUTMContent:  req.UTMContent,
			FirstUTMTerm:     req.UTMTerm,
		}
		_ = h.users.Upsert(ctx, tu)
		if phone != "" || email != "" {
			_, _ = h.users.ResolveAndLink(ctx, accountID, tu.ExternalID, phone, email)
		}
		return tu

	case domain.EventLead, domain.EventInitiateCheckout, domain.EventPurchase,
		domain.EventContact, domain.EventCompleteRegistration, domain.EventAddPaymentInfo,
		domain.EventAddShippingInfo, domain.EventAddToCart:
		tu := &domain.TrackedUser{
			ExternalID:      req.ExternalID,
			IDWpp:           req.IDWpp,
			PixelName:       pixelName,
			AccountID:       accountID,
			FBP:             req.FBP,
			FBC:             req.FBC,
			GCLID:           req.GCLID,
			WBraid:          req.WBraid,
			GBraid:          req.GBraid,
			TTClid:          req.TTClid,
			TTP:             req.TTP,
			GA4ClientID:     req.GA4ClientID,
			GA4SessionID:    req.GA4SessionID,
			ClientIP:        ip,
			ClientUserAgent: ua,
			FirstName:       req.FirstName,
			LastName:        req.LastName,
			Email:           email,
			Phone:           phone,
			URL:             mappedURL,
		}
		_ = h.users.Upsert(ctx, tu)
		if phone != "" || email != "" {
			_, _ = h.users.ResolveAndLink(ctx, accountID, tu.ExternalID, phone, email)
		}
		return tu
	}
	return nil
}

// extractClientIP pega o IP real do client (considerando proxies/CDNs).
func extractClientIP(r *http.Request) string {
	return clientIP(r)
}
