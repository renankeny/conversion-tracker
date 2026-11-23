package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/webpush"
)

type AdminNotifications struct {
	repo   *repo.NotificationRepo
	notify *notify.Service
}

func NewAdminNotifications(r *repo.NotificationRepo, n *notify.Service) *AdminNotifications {
	return &AdminNotifications{repo: r, notify: n}
}

func (h *AdminNotifications) PublicVAPIDKey(w http.ResponseWriter, r *http.Request) {
	pub := ""
	if h.notify != nil {
		pub = h.notify.PublicKey(r.Context())
	}
	writeJSON(w, http.StatusOK, map[string]string{"public_key": pub})
}

func (h *AdminNotifications) Overview(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	prefs, err := h.repo.GetPreferences(r.Context(), c.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar preferências")
		return
	}
	subs, err := h.repo.ListSubscriptionsByUser(r.Context(), c.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar dispositivos")
		return
	}
	items, unread, err := h.repo.ListRecentByUser(r.Context(), c.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao carregar histórico de notificações")
		return
	}
	pub := ""
	if h.notify != nil {
		pub = h.notify.PublicKey(r.Context())
	}
	writeJSON(w, http.StatusOK, domain.NotificationsOverview{
		VAPIDPublicKey: pub,
		UnreadCount:    unread,
		Preferences:    prefs,
		Subscriptions:  subs,
		Notifications:  items,
	})
}

func (h *AdminNotifications) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in domain.NotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	in.UserID = c.UserID
	in.AccountScopeMode = domain.NormalizeAccountScopeMode(in.AccountScopeMode)
	saved, err := h.repo.UpsertPreferences(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar preferências")
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

type pushSubscriptionInput struct {
	Endpoint    string `json:"endpoint"`
	DeviceLabel string `json:"device_label"`
	Keys        struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *AdminNotifications) UpsertSubscription(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in pushSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	endpoint := strings.TrimSpace(in.Endpoint)
	if err := webpush.ValidateEndpoint(endpoint); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(in.Keys.P256dh) == "" || strings.TrimSpace(in.Keys.Auth) == "" {
		writeError(w, http.StatusBadRequest, "chaves de criptografia do navegador ausentes")
		return
	}
	ua := strings.TrimSpace(r.Header.Get("User-Agent"))
	label := strings.TrimSpace(in.DeviceLabel)
	if label == "" {
		label = inferDeviceLabel(ua)
	}
	sub := &domain.PushSubscription{
		UserID:      c.UserID,
		Endpoint:    endpoint,
		P256dh:      strings.TrimSpace(in.Keys.P256dh),
		AuthSecret:  strings.TrimSpace(in.Keys.Auth),
		DeviceLabel: label,
		UserAgent:   ua,
	}
	if err := h.repo.UpsertSubscription(r.Context(), sub); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao registrar este dispositivo")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *AdminNotifications) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	subID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || subID <= 0 {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return
	}
	if err := h.repo.DeleteSubscription(r.Context(), c.UserID, subID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "dispositivo não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao remover dispositivo")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *AdminNotifications) MarkRead(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	notifID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || notifID <= 0 {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return
	}
	_ = h.repo.MarkRead(r.Context(), c.UserID, notifID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *AdminNotifications) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	_ = h.repo.MarkAllRead(r.Context(), c.UserID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *AdminNotifications) SendTest(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var body struct {
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !domain.IsConfigurableNotificationCategory(body.Category) {
		writeError(w, http.StatusBadRequest, "categoria inválida para teste")
		return
	}
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	item, err := h.notify.SendTest(r.Context(), c.UserID, accID, body.Category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}
