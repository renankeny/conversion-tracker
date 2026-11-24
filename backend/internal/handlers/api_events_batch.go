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
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func (h *APIEvents) processEvent(
	ctx context.Context,
	req apiEventRequest,
	accountID int64,
	r *http.Request,
) (result map[string]any, httpStatus int, errorMessage string) {
	req.PixelName = strings.TrimSpace(req.PixelName)
	req.EventName = strings.TrimSpace(req.EventName)
	req.ExternalID = strings.TrimSpace(req.ExternalID)
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

	valPtr := req.Value
	if req.EventName == domain.EventPurchase {
		if (valPtr == nil || *valPtr <= 0) && len(pixel.TrackerConfig.ProductValue) > 0 && pixel.TrackerConfig.ProductValue[0] > 0 {
			pv := pixel.TrackerConfig.ProductValue[0]
			valPtr = &pv
		}
		if valPtr == nil || *valPtr <= 0 {
			return nil, http.StatusBadRequest, "value (> 0) required for Purchase"
		}
	}

	existingUser, _ := h.users.FindByExternalIDByAccount(ctx, req.ExternalID, accountID)
	if existingUser != nil {
		h.users.EnrichGroupFields(ctx, existingUser)
	}

	clientIP := strings.TrimSpace(req.ClientIP)
	clientUA := strings.TrimSpace(req.ClientUserAgent)
	if existingUser != nil {
		if clientIP == "" && existingUser.ClientIP != "" {
			clientIP = existingUser.ClientIP
		}
		if clientUA == "" && existingUser.ClientUserAgent != "" {
			clientUA = existingUser.ClientUserAgent
		}
	}
	reqUA := r.Header.Get("User-Agent")
	if !isServerSideHTTPUserAgent(reqUA) {
		if clientIP == "" {
			clientIP = extractClientIP(r)
		}
		if clientUA == "" {
			clientUA = reqUA
		}
	}

	country := strings.ToLower(strings.TrimSpace(req.Country))
	state := strings.ToLower(strings.TrimSpace(req.State))
	city := util.NormalizeCity(req.City)
	postal := strings.TrimSpace(req.PostalCode)
	if existingUser != nil {
		country = util.FirstNonEmpty(country, existingUser.Country)
		state = util.FirstNonEmpty(state, existingUser.State)
		city = util.FirstNonEmpty(city, existingUser.City)
		postal = util.FirstNonEmpty(postal, existingUser.PostalCode)
	}
	if (country == "" || state == "") && clientIP != "" {
		geo, _ := h.geo.Resolve(clientIP)
		country = util.FirstNonEmpty(country, strings.ToLower(geo.Country))
		state = util.FirstNonEmpty(state, strings.ToLower(geo.State))
		city = util.FirstNonEmpty(city, util.NormalizeCity(geo.City))
		postal = util.FirstNonEmpty(postal, geo.PostalCode)
	}

	phone := util.NormalizePhoneWithCountry(req.Phone, country)
	email := util.NormalizeEmail(req.Email)
	firstName, lastName := req.FirstName, req.LastName
	fbp, fbc := req.FBP, req.FBC
	gclid, wbraid, gbraid := req.GCLID, req.WBraid, req.GBraid
	ttclid, ttp := req.TTClid, req.TTP
	ga4ClientID, ga4SessionID := req.GA4ClientID, req.GA4SessionID
	idWpp := req.IDWpp
	ctwaClid := strings.TrimSpace(req.CtwaClid)

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
		ctwaClid = util.FirstNonEmpty(ctwaClid, existingUser.CtwaClid)
	}

	mappedURL := strings.TrimSpace(req.SourceURL)
	if mappedURL == "" && existingUser != nil {
		mappedURL = util.FirstNonEmpty(existingUser.URL, existingUser.FirstURL)
	}
	if mappedURL != "" {
		mappedURL = h.urls.ApplyForAccount(accountID, mappedURL)
	}

	defaultEvID := ""
	if req.EventName == domain.EventPurchase {
		defaultEvID = strings.TrimSpace(req.OrderID)
	}
	eventID := util.FirstNonEmpty(strings.TrimSpace(req.EventID), defaultEvID, util.GenerateID10())
	if claimed, redisOK := cache.ClaimEventDedup(ctx, h.rdb, accountID, pixel.Name, eventID, 0); redisOK {
		if !claimed {
			return map[string]any{"queued": false, "deduplicated": true, "event_id": eventID}, http.StatusOK, ""
		}
	} else if exists, err := h.sentEvents.ExistsSent(ctx, accountID, eventID, pixel.Name, ""); err == nil && exists {
		return map[string]any{"queued": false, "deduplicated": true, "event_id": eventID}, http.StatusOK, ""
	}

	tu := h.upsertTrackedUserAPI(ctx, req, pixel.Name, accountID, clientIP, clientUA, mappedURL, country, state, city, postal, email, phone)
	var tuID int64
	var tuGroupID *int64
	if tu != nil {
		tuID = tu.ID
		tuGroupID = tu.GroupID
	}
	if req.EventName == domain.EventPurchase {
		ord := strings.TrimSpace(req.OrderID)
		now := time.Now()
		dupInFlight := cache.HasInFlightMatchingPurchase(ctx, h.rdb, accountID, pixel.Name, tuGroupID, tuID, req.ExternalID, phone, email, ord, ord == "", *valPtr, now)
		dupDB := false
		if !dupInFlight && h.sentEvents != nil {
			dupDB, _ = h.sentEvents.HasDuplicateExternalPurchase(ctx, accountID, tuID, req.ExternalID, pixel.Name, ord, *valPtr, now)
		}
		if dupInFlight || dupDB {
			if h.crmRepo != nil && tuID > 0 {
				if leadID, synced := h.crmRepo.SyncOrCreateLeadOnPurchase(ctx, accountID, tuID, *valPtr, false); synced && h.rdb != nil {
					wshub.PublishCustom(ctx, h.rdb, accountID, "crm_updated", map[string]any{"action": "api_purchase_synced", "lead_id": leadID, "event": domain.EventPurchase})
				}
			}
			return map[string]any{"queued": false, "deduplicated": true, "event_id": eventID}, http.StatusOK, ""
		}
		cache.RecordInFlightPurchase(ctx, h.rdb, accountID, pixel.Name, tuGroupID, tuID, req.ExternalID, phone, email, cache.InFlightPurchase{
			EventID: eventID, OrderID: ord, IsNonOrder: ord == "", Value: *valPtr,
		})
	}
	if h.crmRepo != nil && tuID > 0 {
		switch req.EventName {
		case domain.EventPurchase:
			if leadID, synced := h.crmRepo.SyncOrCreateLeadOnPurchase(ctx, accountID, tuID, *valPtr, true); synced && h.rdb != nil {
				wshub.PublishCustom(ctx, h.rdb, accountID, "crm_updated", map[string]any{"action": "api_purchase_synced", "lead_id": leadID, "event": domain.EventPurchase})
			}
		case domain.EventLead, domain.EventLeadSubmitted, domain.EventQualifiedLead:
			if firstStage, errStage := h.crmRepo.GetFirstStage(ctx, accountID); errStage == nil && firstStage != nil {
				_ = h.crmRepo.EnsureLead(ctx, accountID, tuID, firstStage.ID)
			}
		}
	}

	eventTime := time.Now().Unix()
	if req.EventTime != nil && *req.EventTime > 0 {
		eventTime = util.ClampEventTime(*req.EventTime, time.Now())
	}
	actionSource := strings.TrimSpace(req.ActionSource)
	if actionSource == "" {
		if ctwaClid != "" && strings.TrimSpace(req.PageID) != "" {
			actionSource = "business_messaging"
		} else if mappedURL != "" {
			actionSource = "website"
		} else {
			actionSource = "system_generated"
		}
	}

	in := domain.SendEventInput{
		PixelName: pixel.Name, EventName: req.EventName, EventID: eventID, EventTime: eventTime,
		SourceURL: mappedURL, ContentID: util.FirstNonEmpty(req.ContentID, pixel.Name),
		ContentName: req.ContentName, ContentType: util.FirstNonEmpty(req.ContentType, "product"),
		Value: valPtr, Currency: domain.NormalizeWhatsAppCurrency(req.Currency), OrderID: req.OrderID, NumItems: req.NumItems,
		ActionSource: actionSource, MessagingChannel: req.MessagingChannel, CtwaClid: ctwaClid, PageID: strings.TrimSpace(req.PageID),
		ExternalID: req.ExternalID, IDWpp: idWpp, ClientIP: clientIP, ClientUserAgent: clientUA,
		FBP: fbp, FBC: fbc, Email: email, Phone: phone, FirstName: firstName, LastName: lastName,
		Country: country, State: state, City: city, PostalCode: postal,
		TTClid: ttclid, TTP: ttp, Referrer: req.Referrer,
		GCLID: gclid, WBraid: wbraid, GBraid: gbraid, GA4ClientID: ga4ClientID, GA4SessionID: ga4SessionID,
		CRMSynced: req.EventName == domain.EventPurchase,
	}
	service.EnrichSendInput(ctx, h.users, accountID, &in)

	if err := h.queue.EnqueueSendEvent(ctx, in); err != nil {
		cache.ReleaseEventDedup(ctx, h.rdb, accountID, pixel.Name, eventID)
		slog.Error("enqueue send", "err", err, "event_id", eventID)
		return nil, http.StatusInternalServerError, "queue error"
	}

	return map[string]any{"queued": true, "event_id": eventID}, http.StatusAccepted, ""
}
