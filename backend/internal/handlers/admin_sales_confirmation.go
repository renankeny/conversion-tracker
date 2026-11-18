package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/service"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

type AdminSalesConfirmation struct {
	sentEventsRepo *repo.SentEventRepo
	usersRepo      *repo.UserRepo
	pixelsRepo     *repo.PixelRepo
	whatsappRepo   *repo.WhatsAppRepo
	metaAccRepo    *repo.MetaAccountRepo
	crmRepo        *repo.CRMRepo
	enqueuer       *queue.Enqueuer
	rdb            *redis.Client
}

func NewAdminSalesConfirmation(s *repo.SentEventRepo, u *repo.UserRepo, q *queue.Enqueuer) *AdminSalesConfirmation {
	return &AdminSalesConfirmation{sentEventsRepo: s, usersRepo: u, enqueuer: q}
}

func (h *AdminSalesConfirmation) SetRedis(rdb *redis.Client) { h.rdb = rdb }

func (h *AdminSalesConfirmation) SetAttributionDeps(px *repo.PixelRepo, wa *repo.WhatsAppRepo, m *repo.MetaAccountRepo, crm *repo.CRMRepo) {
	h.pixelsRepo, h.whatsappRepo, h.metaAccRepo, h.crmRepo = px, wa, m, crm
}

func (h *AdminSalesConfirmation) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	pixelName := r.URL.Query().Get("pixel")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	items, total, err := h.sentEventsRepo.ListSalesForConfirmation(r.Context(), scope.EffectiveAccountID(0), pixelName, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error fetching sales")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (h *AdminSalesConfirmation) Confirm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		EventID   string  `json:"event_id"`
		PixelName string  `json:"pixel_name"`
		Value     float64 `json:"value"`
		Currency  string  `json:"currency"`
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

	scope := TenantFromContext(ctx)
	scopedAccountID := scope.EffectiveAccountID(0)
	events, err := h.sentEventsRepo.FindByEventID(ctx, req.EventID, req.PixelName, scopedAccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error finding event")
		return
	}
	if len(events) == 0 {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	originalEvent := events[0]
	pixelName := util.FirstNonEmpty(strings.TrimSpace(req.PixelName), originalEvent.PixelName)

	purchaseEventID := BuildConfirmedPurchaseEventID(req.EventID)
	if exists, err := h.sentEventsRepo.ExistsSent(ctx, scopedAccountID, purchaseEventID, pixelName, ""); err == nil && exists {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "already_confirmed": true, "event_id": purchaseEventID})
		return
	}

	user, err := h.usersRepo.FindByExternalIDByAccount(ctx, originalEvent.ExternalID, scopedAccountID)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "error finding user")
		return
	}
	effAccountID := scopedAccountID
	if user != nil {
		if effAccountID == 0 {
			effAccountID = user.AccountID
		}
		h.usersRepo.EnrichGroupFields(ctx, user)
	} else {
		user = &domain.TrackedUser{PixelName: pixelName, AccountID: effAccountID, ExternalID: originalEvent.ExternalID}
		if effAccountID > 0 && user.ExternalID != "" {
			_ = h.usersRepo.Create(ctx, user)
		}
	}

	val := originalEvent.PurchaseValue
	if req.Value > 0 {
		val = req.Value
	}
	var pixel *domain.Pixel
	if h.pixelsRepo != nil {
		pixel, _ = h.pixelsRepo.FindByName(ctx, pixelName)
		if val <= 0 && pixel != nil && len(pixel.TrackerConfig.ProductValue) > 0 && pixel.TrackerConfig.ProductValue[0] > 0 {
			val = pixel.TrackerConfig.ProductValue[0]
		}
	}
	if val <= 0 {
		writeError(w, http.StatusBadRequest, "valor da venda deve ser maior que zero para disparar Purchase")
		return
	}

	if dup, _ := h.sentEventsRepo.HasDuplicateExternalPurchase(ctx, effAccountID, user.ID, user.ExternalID, pixelName, req.EventID, val, time.Now()); dup {
		if h.crmRepo != nil && user.ID > 0 {
			if leadID, synced := h.crmRepo.SyncOrCreateLeadOnPurchase(ctx, effAccountID, user.ID, val, false); synced && h.rdb != nil {
				wshub.PublishCustom(ctx, h.rdb, effAccountID, "crm_updated", map[string]any{"action": "sales_confirmation_synced", "lead_id": leadID, "event": domain.EventPurchase})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "already_confirmed": true, "event_id": purchaseEventID})
		return
	}

	currency := req.Currency
	if currency == "" && h.whatsappRepo != nil {
		var pxID int64
		if pixel != nil {
			pxID = pixel.ID
		}
		currency = h.whatsappRepo.ResolveCurrencyForLead(ctx, effAccountID, user.ID, pxID)
	}
	currency = domain.NormalizeWhatsAppCurrency(currency)

	sourceURL := util.FirstNonEmpty(user.URL, user.FirstURL)
	actionSource := "website"
	msgChannel := ""
	ctwaClid := ""
	pageID := ""
	if user.CtwaClid != "" && user.FBC == "" && h.whatsappRepo != nil {
		pageID = h.whatsappRepo.FindPageIDByCtwaClid(ctx, effAccountID, user.CtwaClid)
		if pageID == "" && user.ID > 0 {
			pageID = h.whatsappRepo.FindPageIDByTrackedUserID(ctx, effAccountID, user.ID)
		}
		if pageID == "" && h.metaAccRepo != nil && pixel != nil && pixel.MetaAccountID != nil && *pixel.MetaAccountID > 0 {
			if mAcc, merr := h.metaAccRepo.FindByID(ctx, *pixel.MetaAccountID); merr == nil && mAcc != nil && mAcc.AccountID == effAccountID && mAcc.Enabled && mAcc.IsConnected() {
				pageID = mAcc.DefaultPageID
			}
		}
		if pageID != "" {
			actionSource = "business_messaging"
			msgChannel = "whatsapp"
			ctwaClid = user.CtwaClid
			sourceURL = ""
		}
	}
	if actionSource == "website" && sourceURL == "" {
		actionSource = "system_generated"
	}

	input := domain.SendEventInput{
		PixelName: pixelName, EventName: domain.EventPurchase, EventID: purchaseEventID,
		OrderID: req.EventID, EventTime: time.Now().Unix(), SourceURL: sourceURL,
		Value: &val, Currency: currency, ActionSource: actionSource,
		MessagingChannel: msgChannel, CtwaClid: ctwaClid, PageID: pageID,
		ExternalID: user.ExternalID, IDWpp: user.IDWpp, ClientIP: user.ClientIP, ClientUserAgent: user.ClientUserAgent,
		FBP: user.FBP, FBC: user.FBC, GCLID: user.GCLID, WBraid: user.WBraid, GBraid: user.GBraid,
		TTClid: user.TTClid, TTP: user.TTP, GA4ClientID: user.GA4ClientID, GA4SessionID: user.GA4SessionID,
		Email: user.Email, Phone: user.Phone, FirstName: user.FirstName, LastName: user.LastName,
		Country: user.Country, State: user.State, City: user.City, PostalCode: user.PostalCode,
		CRMSynced: true,
	}
	service.EnrichSendInput(ctx, h.usersRepo, effAccountID, &input)

	if err := h.enqueuer.EnqueueSendEvent(ctx, input); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue event")
		return
	}
	if h.crmRepo != nil && user.ID > 0 {
		if leadID, synced := h.crmRepo.SyncOrCreateLeadOnPurchase(ctx, effAccountID, user.ID, val, true); synced && h.rdb != nil {
			wshub.PublishCustom(ctx, h.rdb, effAccountID, "crm_updated", map[string]any{"action": "sales_confirmation_synced", "lead_id": leadID, "event": domain.EventPurchase})
		}
	}
	cache.BumpDashboardVersion(ctx, h.rdb, effAccountID)
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "event_id": purchaseEventID})
}

func BuildConfirmedPurchaseEventID(originalEventID string) string {
	trimmed := strings.TrimSpace(originalEventID)
	if strings.HasPrefix(trimmed, "pur_") {
		return trimmed
	}
	return "pur_" + trimmed
}
