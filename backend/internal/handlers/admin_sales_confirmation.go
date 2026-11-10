package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/service"
)

type AdminSalesConfirmation struct {
	sentEventsRepo *repo.SentEventRepo
	usersRepo      *repo.UserRepo
	enqueuer       *queue.Enqueuer
}

func NewAdminSalesConfirmation(s *repo.SentEventRepo, u *repo.UserRepo, q *queue.Enqueuer) *AdminSalesConfirmation {
	return &AdminSalesConfirmation{sentEventsRepo: s, usersRepo: u, enqueuer: q}
}

// List GET /api/v1/admin/sales-confirmations
func (h *AdminSalesConfirmation) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	pixelName := r.URL.Query().Get("pixel")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	items, total, err := h.sentEventsRepo.ListSalesForConfirmation(r.Context(), scope.AccountID, pixelName, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error fetching sales")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
	})
}

// Confirm POST /api/v1/admin/sales-confirmations/confirm
func (h *AdminSalesConfirmation) Confirm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventID   string `json:"event_id"`
		PixelName string `json:"pixel_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.EventID = strings.TrimSpace(req.EventID)
	if req.EventID == "" {
		writeError(w, http.StatusBadRequest, "event_id required")
		return
	}

	scope := TenantFromContext(r.Context())
	events, err := h.sentEventsRepo.FindByEventID(r.Context(), req.EventID, req.PixelName, scope.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error finding event")
		return
	}
	if len(events) == 0 {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	originalEvent := events[0]
	pixelName := req.PixelName
	if pixelName == "" {
		pixelName = originalEvent.PixelName
	}

	// Gera EventID dedicado para o novo disparo Purchase sem colidir com o AddPaymentInfo original
	// em GetSuccessfulChannels (que antes pulava todos os canais já com status='sent').
	purchaseEventID := BuildConfirmedPurchaseEventID(req.EventID)
	if exists, err := h.sentEventsRepo.ExistsSent(r.Context(), purchaseEventID, pixelName); err == nil && exists {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "already_confirmed": true, "event_id": purchaseEventID})
		return
	}

	user, err := h.usersRepo.FindByExternalIDAndPixel(r.Context(), originalEvent.ExternalID, pixelName)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "error finding user")
		return
	}
	effAccountID := scope.AccountID
	if user != nil && effAccountID == 0 {
		effAccountID = user.AccountID
	}
	if user == nil {
		user = &domain.TrackedUser{
			PixelName:  pixelName,
			AccountID:  effAccountID,
			ExternalID: originalEvent.ExternalID,
		}
	}

	if err := h.sentEventsRepo.MarkAsConfirmed(r.Context(), req.EventID, pixelName, scope.AccountID); err != nil {
		slog.Warn("failed to mark sales event as confirmed in db", "err", err, "event_id", req.EventID)
	}

	val := originalEvent.PurchaseValue
	input := domain.SendEventInput{
		PixelName:       pixelName,
		EventName:       domain.EventPurchase,
		EventID:         purchaseEventID,
		OrderID:         req.EventID,
		EventTime:       time.Now().Unix(),
		SourceURL:       user.URL,
		Value:           &val,
		Currency:        "BRL",
		ActionSource:    "website",
		ExternalID:      user.ExternalID,
		IDWpp:           user.IDWpp,
		ClientIP:        user.ClientIP,
		ClientUserAgent: user.ClientUserAgent,
		FBP:             user.FBP,
		FBC:             user.FBC,
		GCLID:           user.GCLID,
		WBraid:          user.WBraid,
		GBraid:          user.GBraid,
		TTClid:          user.TTClid,
		TTP:             user.TTP,
		GA4ClientID:     user.GA4ClientID,
		GA4SessionID:    user.GA4SessionID,
		Email:           user.Email,
		Phone:           user.Phone,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		Country:         user.Country,
		State:           user.State,
		City:            user.City,
		PostalCode:      user.PostalCode,
	}
	service.EnrichSendInput(r.Context(), h.usersRepo, effAccountID, &input)

	if err := h.enqueuer.EnqueueSendEvent(r.Context(), input); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue event")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "event_id": purchaseEventID})
}

// BuildConfirmedPurchaseEventID deriva um EventID determinístico distinto do AddPaymentInfo original.
func BuildConfirmedPurchaseEventID(originalEventID string) string {
	trimmed := strings.TrimSpace(originalEventID)
	if strings.HasPrefix(trimmed, "pur_") {
		return trimmed
	}
	return "pur_" + trimmed
}
