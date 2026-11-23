package httpserver

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/handlers"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func attachNotificationHandlers(rh *routeHandlers, deps Deps, rdb *redis.Client) *notify.Service {
	notifRepo := repo.NewNotificationRepo(deps.Pool)
	svc := notify.NewService(notifRepo, rdb, deps.Cfg.AdminEmail, deps.Cfg.AppURL)
	_, _ = svc.EnsureKeys(context.Background())

	rh.adminNotifications = handlers.NewAdminNotifications(notifRepo, svc)
	if rh.events != nil {
		rh.events.SetNotify(svc)
	}
	if rh.adminCRM != nil {
		rh.adminCRM.SetNotify(svc)
	}
	if rh.adminMetaAds != nil {
		rh.adminMetaAds.SetNotify(svc)
	}
	if rh.adminGoogleAds != nil {
		rh.adminGoogleAds.SetNotify(svc)
	}
	if rh.adminPixels != nil {
		rh.adminPixels.SetNotify(svc)
	}
	if rh.adminAccounts != nil {
		rh.adminAccounts.SetNotify(svc)
	}
	if rh.adminAPIKeys != nil {
		rh.adminAPIKeys.SetNotify(svc)
	}
	return svc
}

func registerNotificationPublicRoutes(r chi.Router, h routeHandlers) {
	if h.adminNotifications != nil {
		r.Get("/push/vapid-public-key", h.adminNotifications.PublicVAPIDKey)
	}
}

func registerNotificationAdminRoutes(r chi.Router, h routeHandlers) {
	if h.adminNotifications == nil {
		return
	}
	r.Get("/admin/notifications", h.adminNotifications.Overview)
	r.Put("/admin/notifications/preferences", h.adminNotifications.UpdatePreferences)
	r.Post("/admin/notifications/subscriptions", h.adminNotifications.UpsertSubscription)
	r.Delete("/admin/notifications/subscriptions/{id}", h.adminNotifications.DeleteSubscription)
	r.Post("/admin/notifications/read-all", h.adminNotifications.MarkAllRead)
	r.Post("/admin/notifications/{id}/read", h.adminNotifications.MarkRead)
	r.Post("/admin/notifications/test", h.adminNotifications.SendTest)
}
