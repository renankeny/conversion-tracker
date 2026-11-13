package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/service"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// processEvent valida e enfileira um evento individual.
// Retorna (result, httpStatus, errorMessage). Se errorMessage != "", houve erro.
func (h *APIEvents) processEvent(
	ctx context.Context,
	req apiEventRequest,
	accountID int64,
	r *http.Request,
) (result map[string]any, httpStatus int, errorMessage string) {
	if req.PixelName == "" {
		return nil, http.StatusBadRequest, "pixel_name required"
	}
	if req.EventName == "" {
		return nil, http.StatusBadRequest, "event_name required"
	}
	if req.ExternalID == "" {
		return nil, http.StatusBadRequest, "external_id required"
	}
	if !domain.IsValidEventType(req.EventName) {
		return nil, http.StatusBadRequest, "invalid event_name: " + req.EventName
	}

	// Resolve o pixel — aceita tanto o name quanto o public_token.
	pixel, err := h.pixels.FindByNameOrToken(ctx, req.PixelName)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, http.StatusNotFound, "unknown pixel_name: " + req.PixelName
		}
		slog.Error("find pixel", "err", err, "pixel_name", req.PixelName)
		return nil, http.StatusInternalServerError, "internal error"
	}
	if pixel.AccountID != accountID {
		return nil, http.StatusForbidden, "pixel does not belong to this account"
	}
	if !pixel.Enabled {
		return nil, http.StatusForbidden, "pixel disabled"
	}

	// Tenta buscar dados do tracked_user existente pelo external_id + pixel_name
	existingUser, _ := h.users.FindByExternalIDAndPixel(ctx, req.ExternalID, pixel.Name)

	clientIP := req.ClientIP
	clientUA := req.ClientUserAgent
	if existingUser != nil {
		if clientIP == "" && existingUser.ClientIP != "" {
			clientIP = existingUser.ClientIP
		}
		if clientUA == "" && existingUser.ClientUserAgent != "" {
			clientUA = existingUser.ClientUserAgent
		}
	}
	if clientIP == "" {
		clientIP = extractClientIP(r)
	}
	if clientUA == "" {
		clientUA = r.Header.Get("User-Agent")
	}

	// GeoIP: usa dados fornecidos, ou do tracked_user, ou faz lookup pelo IP
	country := strings.ToLower(req.Country)
	state := strings.ToLower(req.State)
	city := strings.ToLower(req.City)
	postal := req.PostalCode
	if existingUser != nil {
		if country == "" && existingUser.Country != "" {
			country = existingUser.Country
		}
		if state == "" && existingUser.State != "" {
			state = existingUser.State
		}
		if city == "" && existingUser.City != "" {
			city = existingUser.City
		}
		if postal == "" && existingUser.PostalCode != "" {
			postal = existingUser.PostalCode
		}
	}
	if country == "" || state == "" {
		geo, _ := h.geo.Resolve(clientIP)
		if country == "" {
			country = strings.ToLower(geo.Country)
		}
		if state == "" {
			state = strings.ToLower(geo.State)
		}
		if city == "" {
			city = util.NormalizeCity(geo.City)
		}
		if postal == "" {
			postal = geo.PostalCode
		}
	}

	// Normaliza PII e identificadores de rastreamento
	phone := util.NormalizePhone(req.Phone)
	email := util.NormalizeEmail(req.Email)
	firstName, lastName := req.FirstName, req.LastName
	fbp, fbc := req.FBP, req.FBC
	gclid, wbraid, gbraid := req.GCLID, req.WBraid, req.GBraid
	ttclid, ttp := req.TTClid, req.TTP
	ga4ClientID, ga4SessionID := req.GA4ClientID, req.GA4SessionID
	idWpp := req.IDWpp

	if existingUser != nil {
		phone = util.FirstNonEmpty(phone, existingUser.Phone)
		email = util.FirstNonEmpty(email, existingUser.Email)
		firstName = util.FirstNonEmpty(firstName, existingUser.FirstName)
		lastName = util.FirstNonEmpty(lastName, existingUser.LastName)
		fbp = util.FirstNonEmpty(fbp, existingUser.FBP)
		fbc = util.FirstNonEmpty(fbc, existingUser.FBC)
		gclid = util.FirstNonEmpty(gclid, existingUser.GCLID)
		wbraid = util.FirstNonEmpty(wbraid, existingUser.WBraid)
		gbraid = util.FirstNonEmpty(gbraid, existingUser.GBraid)
		ttclid = util.FirstNonEmpty(ttclid, existingUser.TTClid)
		ttp = util.FirstNonEmpty(ttp, existingUser.TTP)
		ga4ClientID = util.FirstNonEmpty(ga4ClientID, existingUser.GA4ClientID)
		ga4SessionID = util.FirstNonEmpty(ga4SessionID, existingUser.GA4SessionID)
		idWpp = util.FirstNonEmpty(idWpp, existingUser.IDWpp)
	}

	mappedURL := req.SourceURL
	if mappedURL == "" && existingUser != nil && existingUser.URL != "" {
		mappedURL = existingUser.URL
	}
	if mappedURL != "" {
		mappedURL = h.urls.ApplyForAccount(accountID, mappedURL)
	}

	eventID := req.EventID
	if eventID == "" {
		eventID = util.GenerateID10()
	}

	if claimed, redisOK := cache.ClaimEventDedup(ctx, h.rdb, accountID, pixel.Name, eventID, 0); redisOK {
		if !claimed {
			return map[string]any{"queued": false, "deduplicated": true, "event_id": eventID}, http.StatusOK, ""
		}
	} else if exists, err := h.sentEvents.ExistsSent(ctx, accountID, eventID, pixel.Name, ""); err != nil {
		slog.Warn("dedup check failed", "err", err, "event_id", eventID)
	} else if exists {
		return map[string]any{"queued": false, "deduplicated": true, "event_id": eventID}, http.StatusOK, ""
	}

	tu := h.upsertTrackedUserAPI(ctx, req, pixel.Name, accountID, clientIP, clientUA, mappedURL, country, state, city, postal, email, phone)
	if tu != nil && tu.ID > 0 && req.EventName == domain.EventLead {
		if firstStage, errStage := h.crmRepo.GetFirstStage(ctx, accountID); errStage == nil && firstStage != nil {
			_ = h.crmRepo.EnsureLead(ctx, accountID, tu.ID, firstStage.ID)
		}
	}

	eventTime := time.Now().Unix()
	if req.EventTime != nil && *req.EventTime > 0 {
		eventTime = util.ClampEventTime(*req.EventTime, time.Now())
	}
	contentID := req.ContentID
	if contentID == "" {
		contentID = pixel.Name
	}
	contentType := req.ContentType
	if contentType == "" {
		contentType = "product"
	}

	in := domain.SendEventInput{
		PixelName:       pixel.Name,
		EventName:       req.EventName,
		EventID:         eventID,
		EventTime:       eventTime,
		SourceURL:       mappedURL,
		ContentID:       contentID,
		ContentName:     req.ContentName,
		ContentType:     contentType,
		Value:           req.Value,
		Currency:        req.Currency,
		OrderID:         req.OrderID,
		NumItems:        req.NumItems,
		ExternalID:      req.ExternalID,
		IDWpp:           idWpp,
		ClientIP:        clientIP,
		ClientUserAgent: clientUA,
		FBP:             fbp,
		FBC:             fbc,
		Email:           email,
		Phone:           phone,
		FirstName:       firstName,
		LastName:        lastName,
		Country:         country,
		State:           state,
		City:            city,
		PostalCode:      postal,
		TTClid:          ttclid,
		TTP:             ttp,
		Referrer:        req.Referrer,
		GCLID:           gclid,
		WBraid:          wbraid,
		GBraid:          gbraid,
		GA4ClientID:     ga4ClientID,
		GA4SessionID:    ga4SessionID,
	}
	service.EnrichSendInput(ctx, h.users, accountID, &in)

	if err := h.queue.EnqueueSendEvent(ctx, in); err != nil {
		cache.ReleaseEventDedup(ctx, h.rdb, accountID, pixel.Name, eventID)
		slog.Error("enqueue send", "err", err, "event_id", eventID)
		return nil, http.StatusInternalServerError, "queue error"
	}

	return map[string]any{
		"queued":   true,
		"event_id": eventID,
	}, http.StatusAccepted, ""
}
