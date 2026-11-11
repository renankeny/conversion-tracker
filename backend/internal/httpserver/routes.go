package httpserver

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func registerRoutes(r *chi.Mux, deps Deps, h routeHandlers) {
	rl := h.rateLimiter
	rlEvents := rl.LimitByIP("events_collect", 240, 1*time.Minute)
	rlInit := rl.LimitByIP("events_init", 60, 1*time.Minute)
	rlAssets := rl.LimitByIP("assets", 600, 1*time.Minute)
	rlLogin := rl.LimitByIP("login", 10, 1*time.Minute)
	rlPublicDash := rl.LimitByIP("public_dash", 120, 1*time.Minute)
	rlAPIEvents := rl.LimitByIP("api_events", 300, 1*time.Minute)

	r.Route("/api/v1", func(r chi.Router) {
		// WebSocket e streaming de mídia ficam fora do timeout curto de 15s.
		r.Get("/ws/events", h.wsEvents.Handle)
		r.Get("/media/whatsapp/{messageID}", h.whatsappMedia.ServeHTTP)
		registerWhatsAppPublicRoutes(r, h)

		// Rotas HTTP padrão com timeout de 15s.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(15 * time.Second))

			r.Get("/health", h.health.Handler)
			r.Get("/branding", h.branding.GetPublic)
			r.Get("/setup/status", h.setup.Status)
			r.Group(func(r chi.Router) {
				r.Use(MaxBodyBytes(4 * 1024 * 1024))
				r.Use(rlLogin)
				r.Post("/setup/verify-email", h.setup.VerifyEmail)
				r.Post("/setup/init", h.setup.Init)
			})

			r.Group(func(r chi.Router) {
				r.Use(ShardSyncMiddleware(h.calibrator))
				r.Use(rlAssets)
				r.Get("/loader.js", h.tracker.Loader)
				r.Get("/tracker.js", h.tracker.Tracker)
			})
			r.Get("/internal/traefik-config", h.traefik.ServeHTTP)
			r.Post("/internal/shard-compact", ShardCompactHandler(h.calibrator))

			r.Group(func(r chi.Router) {
				r.Use(ShardSyncMiddleware(h.calibrator))
				r.Use(MaxBodyBytes(8 * 1024))
				r.Use(rlInit)
				r.Post("/events/init", h.events.Init)
			})
			r.Group(func(r chi.Router) {
				r.Use(ShardSyncMiddleware(h.calibrator))
				r.Use(MaxBodyBytes(64 * 1024))
				r.Use(rlEvents)
				r.Post("/events/collect", h.events.Send)
			})

			registerWebhookRoutes(r, h)

			r.Group(func(r chi.Router) {
				r.Use(rlPublicDash)
				r.Route("/public/dashboard/{token}", func(r chi.Router) {
					r.Get("/meta", h.publicDashboard.Meta)
					r.Get("/summary", h.publicDashboard.Summary)
					r.Get("/series", h.publicDashboard.Series)
					r.Get("/geo", h.publicDashboard.Geo)
					r.Get("/geo-by-event", h.publicDashboard.GeoByEvent)
					r.Get("/purchases", h.publicDashboard.Purchases)
					r.Get("/revenue", h.publicDashboard.Revenue)
				})
			})

			r.Group(func(r chi.Router) {
				r.Use(ShardSyncMiddleware(h.calibrator))
				r.Use(MaxBodyBytes(4 * 1024))
				r.Use(rlLogin)
				r.Post("/admin/login", h.adminAuth.Login)
			})

			registerOAuthPublicRoutes(r, h)

			r.Group(func(r chi.Router) {
				r.Use(MaxBodyBytes(256 * 1024))
				r.Use(rlAPIEvents)
				r.Post("/send-event", h.apiEvents.SendEvent)
				r.Post("/send-events-batch", h.apiEvents.SendEventsBatch)
			})
		})

		registerAdminRoutes(r, deps, h)
	})
}

func registerAdminRoutes(r chi.Router, deps Deps, h routeHandlers) {
	r.Group(func(r chi.Router) {
		r.Use(ShardSyncMiddleware(h.calibrator))
		r.Use(AuthMiddlewareWithSessions(deps.JWT, h.apiKeysRepo, h.sessions))
		r.Use(TenantMiddlewareWithSessions(deps.Pool, h.sessions))
		r.Use(MaxBodyBytes(4 * 1024 * 1024))

		// Operações administrativas rápidas (15s)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(15 * time.Second))

			// Rotas compatíveis com S2S (`trk_*`) e JWT humano
			r.Get("/admin/ext-api/crm/stages", h.adminCRM.APIListStages)
			r.Post("/admin/ext-api/crm/leads", h.adminCRM.APICreateLead)
			r.Get("/admin/crm/board", h.adminCRM.GetBoard)
			r.Post("/admin/crm/leads", h.adminCRM.APICreateLead)
			r.Put("/admin/crm/leads/{id}/move", h.adminCRM.MoveLead)
			r.Put("/admin/crm/leads/{id}", h.adminCRM.UpdateLead)
			r.Get("/admin/crm/leads/{id}", h.adminCRM.GetLeadDetails)
			r.Post("/admin/crm/leads/{id}/notes", h.adminCRM.AddNote)
			r.Get("/admin/crm/tags", h.adminCRM.ListTags)
			r.Post("/admin/crm/leads/{id}/tags/{tag_id}", h.adminCRM.AssignTag)
			r.Delete("/admin/crm/leads/{id}/tags/{tag_id}", h.adminCRM.RemoveTag)

			r.Get("/admin/sent-events", h.adminSent.List)
			r.Get("/admin/sent-events/grouped", h.adminSent.ListGrouped)
			r.Get("/admin/sent-events/by-event-id/{event_id}", h.adminSent.GetByEventID)
			r.Get("/admin/webhook-logs", h.adminWebhookLogs.List)

			r.Get("/admin/tracked-users", h.adminTrackedUsers.List)
			r.Get("/admin/tracked-users/lookup", h.adminTrackedUsers.Lookup)
			r.Get("/admin/tracked-users/journey", h.adminTrackedUsers.Journey)
			r.Get("/admin/tracked-users/{id}", h.adminTrackedUsers.Get)

			r.Get("/admin/stats", h.adminStats.Overview)
			r.Get("/admin/stats/geo", h.adminStats.Geo)
			r.Get("/admin/stats/geo-by-event", h.adminStats.GeoByEvent)
			r.Get("/admin/counters/summary", h.adminCounters.Summary)
			r.Get("/admin/counters/series", h.adminCounters.TimeSeries)

			r.Group(func(r chi.Router) {
				r.Use(RequireHumanAuth)

				r.Post("/admin/logout", h.adminAuth.Logout)
				r.Post("/admin/ws-ticket", h.adminAuth.IssueWSTicket)
				r.Get("/admin/me", h.adminAuth.Me)
				r.Put("/admin/me", h.adminUsers.UpdateMyProfile)
				r.Post("/admin/me/password", h.adminUsers.ChangeMyPassword)
				r.Put("/admin/me/preferences", h.adminUsers.UpdateMyPreferences)
				r.Put("/admin/branding", h.branding.UpdateAdmin)
				r.Get("/admin/system/health-alerts", h.adminAlerts.GetHealthAlerts)

				r.Get("/admin/pixels", h.adminPixels.List)
				r.Get("/admin/pixels/defaults", h.adminPixels.Defaults)
				r.Get("/admin/pixels/{id}", h.adminPixels.Get)
				r.Post("/admin/pixels", h.adminPixels.Create)
				r.Put("/admin/pixels/{id}", h.adminPixels.Update)
				r.Delete("/admin/pixels/{id}", h.adminPixels.Delete)
				r.Post("/admin/pixels/{id}/verify-dns", h.adminPixels.VerifyDNS)
				r.Post("/admin/pixels/{id}/regenerate-token", h.adminPixels.RegenerateToken)
				r.Put("/admin/pixels/{id}/dashboard-config", h.adminPixels.UpdateDashboardConfig)
				r.Post("/admin/pixels/{id}/dashboard-share-link", h.adminPixels.GenerateDashboardShareLink)
				r.Delete("/admin/pixels/{id}/dashboard-share-link", h.adminPixels.RevokeDashboardShareLink)

				registerOAuthAdminRoutes(r, h)

				r.Get("/admin/geoip/status", h.adminGeoIP.Status)
				r.Get("/admin/maintenance/status", h.adminMaintenance.Status)
				r.Put("/admin/maintenance/retention", h.adminMaintenance.UpdateRetention)

				r.Get("/admin/custom-webhooks", h.adminCustomWebhooks.List)
				r.Post("/admin/custom-webhooks", h.adminCustomWebhooks.Create)
				r.Get("/admin/custom-webhooks/{id}", h.adminCustomWebhooks.Get)
				r.Put("/admin/custom-webhooks/{id}", h.adminCustomWebhooks.Update)
				r.Delete("/admin/custom-webhooks/{id}", h.adminCustomWebhooks.Delete)
				r.Get("/admin/custom-webhooks/{id}/logs", h.adminCustomWebhooks.Logs)

				r.Get("/admin/integrations/platforms/tokens/{platform}", h.adminPlatformTokens.GetToken)
				r.Put("/admin/integrations/platforms/tokens/{platform}", h.adminPlatformTokens.SetToken)

				r.Get("/admin/sales-confirmations", h.adminSalesConfirm.List)
				r.Post("/admin/sales-confirmations/confirm", h.adminSalesConfirm.Confirm)

				r.Get("/admin/users", h.adminUsers.List)
				r.Post("/admin/users", h.adminUsers.Create)
				r.Put("/admin/users/{id}", h.adminUsers.Update)
				r.Delete("/admin/users/{id}", h.adminUsers.Delete)

				r.Get("/admin/accounts", h.adminAccounts.List)
				r.Get("/admin/accounts/{id}", h.adminAccounts.Get)
				r.Post("/admin/accounts", h.adminAccounts.Create)
				r.Put("/admin/accounts/{id}", h.adminAccounts.Update)
				r.Put("/admin/accounts/{id}/event-labels", h.adminAccounts.UpdateEventLabels)
				r.Delete("/admin/accounts/{id}", h.adminAccounts.Delete)
				r.Get("/admin/events-filter", h.adminAccounts.GetEventsFilter)
				r.Put("/admin/events-filter", h.adminAccounts.UpdateEventsFilter)

				r.Post("/admin/crm/stages", h.adminCRM.CreateStage)
				r.Put("/admin/crm/stages/reorder", h.adminCRM.ReorderStages)
				r.Put("/admin/crm/stages/{id}", h.adminCRM.UpdateStage)
				r.Delete("/admin/crm/stages/{id}", h.adminCRM.DeleteStage)
				r.Delete("/admin/crm/leads/{id}", h.adminCRM.DeleteLead)
				r.Delete("/admin/crm/notes/{note_id}", h.adminCRM.DeleteNote)
				r.Post("/admin/crm/tags", h.adminCRM.CreateTag)
				r.Delete("/admin/crm/tags/{tag_id}", h.adminCRM.DeleteTag)

				r.Get("/admin/url-mappings", h.adminURL.List)
				r.Post("/admin/url-mappings", h.adminURL.Create)
				r.Put("/admin/url-mappings/{id}", h.adminURL.Update)
				r.Delete("/admin/url-mappings/{id}", h.adminURL.Delete)

				r.Delete("/admin/sent-events", h.adminSent.BulkDelete)
				r.Post("/admin/sent-events/retry", h.adminSent.Retry)
				r.Post("/admin/webhook-logs/{id}/replay", h.adminWebhookLogs.Replay)
				r.Put("/admin/tracked-users/{id}", h.adminTrackedUsers.Update)

				r.Get("/admin/api-keys", h.adminAPIKeys.List)
				r.Post("/admin/api-keys", h.adminAPIKeys.Create)
				r.Delete("/admin/api-keys/{id}", h.adminAPIKeys.Revoke)

				registerWhatsAppAdminRoutes(r, h)
			})
		})

		// Operações longas com janela estendida de 120s (OpenAI, GeoIP download, Prune em lote, Meta Live API, Testes)
		r.Group(func(r chi.Router) {
			r.Use(RequireHumanAuth)
			r.Use(middleware.Timeout(120 * time.Second))

			r.Post("/admin/openai/webhook-mapping", h.adminOpenAI.GenerateMapping)
			r.Post("/admin/geoip/refresh", h.adminGeoIP.Refresh)
			r.Post("/admin/maintenance/prune", h.adminMaintenance.PruneNow)
			r.Post("/admin/maintenance/reset", h.adminMaintenance.Reset)
			r.Post("/admin/pixels/{id}/test-event", h.adminPixelTest.SendTestEvent)
			r.Post("/admin/google-ads-accounts/{id}/test-event", h.adminGoogleAds.SendTestEvent)
			r.Get("/admin/dashboard/meta/live-insights", h.adminDashboardMeta.LiveInsights)
			r.Get("/admin/dashboard/meta/creative-ranking", h.adminDashboardMeta.CreativeRanking)
			r.Get("/admin/dashboard/google/live-insights", h.adminDashboardGoogle.LiveInsights)
			r.Get("/admin/dashboard/google/keyword-ranking", h.adminDashboardGoogle.KeywordRanking)
			registerWhatsAppLongAdminRoutes(r, h)
		})
	})
}
