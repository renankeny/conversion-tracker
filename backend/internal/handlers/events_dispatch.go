package handlers

import (
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

func (h *Events) handle(w http.ResponseWriter, r *http.Request, req sendRequest) {
	if req.EventType == "" {
		writeError(w, http.StatusBadRequest, "eventType required")
		return
	}
	if !domain.IsValidEventType(req.EventType) {
		writeError(w, http.StatusBadRequest, "invalid eventType: "+req.EventType)
		return
	}

	ctx := r.Context()
	ip := clientIP(r)
	ua := r.Header.Get("User-Agent")
	geo, _ := h.geo.Resolve(ip)

	country := strings.ToLower(geo.Country)
	state := strings.ToLower(geo.State)
	city := util.NormalizeCity(geo.City)
	postal := geo.PostalCode
	if req.PostalCode != "" {
		postal = req.PostalCode
	}

	if req.EventType == domain.EventInit {
		setFirstPartyTrackingCookies(w, r, h.trackingTargetHost, "", req)
		writeJSON(w, http.StatusOK, map[string]any{
			"city":              city,
			"state":             state,
			"postal_code":       postal,
			"country":           country,
			"client_ip_address": ip,
			"client_user_agent": ua,
			"fbc":               req.FBC,
			"fbp":               req.FBP,
			"external_id":       req.UserID,
		})
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "userId required")
		return
	}
	if req.PixelName == "" {
		writeError(w, http.StatusBadRequest, "pixelName required")
		return
	}

	pixel, err := h.pixels.FindByNameOrToken(ctx, req.PixelName)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "unknown pixelName: "+req.PixelName)
			return
		}
		slog.Error("find pixel", "err", err, "pixel_name", req.PixelName)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !pixel.Enabled {
		writeError(w, http.StatusForbidden, "pixel disabled")
		return
	}

	setFirstPartyTrackingCookies(w, r, h.trackingTargetHost, pixel.TrackingHost, req)

	phone := util.NormalizePhone(req.Phone)
	email := util.NormalizeEmail(req.Email)
	mappedURL := h.urls.ApplyForAccount(pixel.AccountID, req.URL)

	if err := h.upsertTrackedUser(ctx, req, pixel.Name, pixel.AccountID, ip, ua, mappedURL, country, state, city, postal, email, phone, req.GA4ClientID, req.GA4SessionID, true); err != nil {
		slog.Error("upsert tracked user", "err", err, "external_id", req.UserID)
	}

	if req.EventType == domain.EventUpdateUserData {
		writeJSON(w, http.StatusOK, map[string]any{"updated": true})
		return
	}

	eventID := req.EventID
	if eventID == "" {
		eventID = util.GenerateID10()
	}

	claimedEv, redisOK := cache.ClaimEventDedup(ctx, h.rdb, pixel.AccountID, pixel.Name, eventID, 0)
	if redisOK {
		if !claimedEv {
			writeJSON(w, http.StatusOK, map[string]any{"queued": false, "deduplicated": true, "event_id": eventID})
			return
		}
	} else {
		if h.isDuplicateInMemory("id:"+pixel.Name+":"+eventID, 5*time.Minute) {
			writeJSON(w, http.StatusOK, map[string]any{"queued": false, "deduplicated": true, "event_id": eventID})
			return
		}
		if exists, err := h.sentEvents.ExistsSent(ctx, pixel.AccountID, eventID, pixel.Name, ""); err != nil {
			slog.Warn("dedup check failed", "err", err, "event_id", eventID)
		} else if exists {
			writeJSON(w, http.StatusOK, map[string]any{"queued": false, "deduplicated": true, "event_id": eventID})
			return
		}
	}

	if req.EventType != domain.EventPurchase {
		// content_id na chave: dois eventos legítimos diferentes (outro produto/form) não se anulam.
		actionKey := req.EventType + ":" + req.ContentID
		if claimedRecent, ok := cache.ClaimRecentAction(ctx, h.rdb, pixel.Name, req.UserID, actionKey, 3*time.Second); ok {
			if !claimedRecent {
				cache.ReleaseEventDedup(ctx, h.rdb, pixel.AccountID, pixel.Name, eventID)
				writeJSON(w, http.StatusOK, map[string]any{"queued": false, "deduplicated": true, "event_id": eventID})
				return
			}
		} else {
			if req.UserID != "" && h.isDuplicateInMemory("act:"+pixel.Name+":"+req.UserID+":"+actionKey, 2*time.Second) {
				writeJSON(w, http.StatusOK, map[string]any{"queued": false, "deduplicated": true, "event_id": eventID})
				return
			}
			if recent, err := h.sentEvents.ExistsRecentEvent(ctx, pixel.Name, req.UserID, req.EventType, 2*time.Second); err == nil && recent {
				writeJSON(w, http.StatusOK, map[string]any{"queued": false, "deduplicated": true, "event_id": eventID})
				return
			}
		}
	}

	contentID := req.ContentID
	if contentID == "" {
		contentID = pixel.Name
	}

	in := domain.SendEventInput{
		PixelName:       pixel.Name,
		EventName:       req.EventType,
		EventID:         eventID,
		EventTime:       time.Now().Unix(),
		SourceURL:       mappedURL,
		ContentID:       contentID,
		ContentName:     req.ContentName,
		ContentType:     "product",
		OrderID:         util.FirstNonEmpty(req.OrderID, req.OrderIDAlt),
		Value:           req.Price,
		Currency:        req.Currency,
		ExternalID:      req.UserID,
		IDWpp:           req.IDWpp,
		ClientIP:        ip,
		ClientUserAgent: ua,
		FBP:             req.FBP,
		FBC:             req.FBC,
		Email:           email,
		Phone:           phone,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		Country:         country,
		State:           state,
		City:            city,
		PostalCode:      postal,
		TTClid:          req.TTClid,
		TTP:             req.TTP,
		Referrer:        req.Referrer,
		GCLID:           req.GCLID,
		WBraid:          req.WBraid,
		GBraid:          req.GBraid,
		GA4ClientID:     req.GA4ClientID,
		GA4SessionID:    req.GA4SessionID,
	}
	service.EnrichSendInput(ctx, h.users, pixel.AccountID, &in)

	if err := h.queue.EnqueueSendEvent(ctx, in); err != nil {
		cache.ReleaseEventDedup(ctx, h.rdb, pixel.AccountID, pixel.Name, eventID)
		slog.Error("enqueue send", "err", err, "event_id", eventID)
		writeError(w, http.StatusInternalServerError, "queue error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"queued":   true,
		"event_id": eventID,
	})
}
