package httpserver

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/mcpserver"
)

func registerRoutes(root *chi.Mux, deps Deps, h routeHandlers) {
	rl := h.rateLimiter
	rlEvents := rl.LimitByIP("events_collect", 240, 1*time.Minute)
	rlInit := rl.LimitByIP("events_init", 60, 1*time.Minute)
	rlAssets := rl.LimitByIP("assets", 600, 1*time.Minute)
	rlLogin := rl.LimitByIP("login", 10, 1*time.Minute)
	rlPublicDash := rl.LimitByIP("public_dash", 120, 1*time.Minute)
	rlAPIEvents := rl.LimitByIP("api_events", 300, 1*time.Minute)

	mcpSrv := mcpserver.New(root, deps.Pool)

	root.Group(func(r chi.Router) {
		r.Use(ShardSyncMiddleware(h.calibrator))
		r.Use(middleware.Timeout(15 * time.Second))
		r.Use(rlAssets)
		r.Get("/w/{slug}", h.publicTrackableLinks.ServeRedirect)
	})

	root.Route("/api/v1", func(r chi.Router) {
		r.Get("/ws/events", h.wsEvents.Handle)
		r.Group(func(r chi.Router) {
			r.Use(ShardSyncMiddleware(h.calibrator))
			r.Get("/media/whatsapp/{messageID}", h.whatsappMedia.ServeHTTP)
			registerWhatsAppPublicRoutes(r, h)
			registerPublicDashboardRoutes(r, h, rlPublicDash)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(15 * time.Second))

			r.Get("/health", h.health.Handler)
			r.Get("/branding", h.branding.GetPublic)
			registerNotificationPublicRoutes(r, h)
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
				r.Use(ShardSyncMiddleware(h.calibrator))
				r.Use(MaxBodyBytes(4 * 1024))
				r.Use(rlLogin)
				r.Post("/admin/login", h.adminAuth.Login)
			})

			registerOAuthPublicRoutes(r, h)

			r.Group(func(r chi.Router) {
				r.Use(ShardSyncMiddleware(h.calibrator))
				r.Use(MaxBodyBytes(256 * 1024))
				r.Use(rlAPIEvents)
				r.Post("/send-event", h.apiEvents.SendEvent)
				r.Post("/send-events-batch", h.apiEvents.SendEventsBatch)
			})
		})

		registerAdminAndMCPRoutes(r, deps, h, mcpSrv)
	})
}

func registerAdminAndMCPRoutes(r chi.Router, deps Deps, h routeHandlers, mcpSrv *mcpserver.Server) {
	r.Group(func(r chi.Router) {
		r.Use(ShardSyncMiddleware(h.calibrator))
		r.Use(AuthMiddlewareWithSessions(deps.JWT, h.apiKeysRepo, h.sessions))
		r.Use(TenantMiddlewareWithSessions(deps.Pool, h.sessions))
		r.Use(MaxBodyBytes(4 * 1024 * 1024))

		// Servidor MCP Nativo Streamable HTTP (usa a mesma chave `trk_*` e respeita seus escopos)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(120 * time.Second))
			r.Post("/mcp", mcpSrv.ServeHTTP)
			r.Get("/mcp", mcpSrv.ServeHTTP)
			r.Delete("/mcp", mcpSrv.ServeHTTP)
		})

		// Operações rápidas (15s)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(15 * time.Second))
			registerS2SFastRoutes(r, h)

			r.Group(func(r chi.Router) {
				r.Use(RequireHumanAuth)
				registerHumanFastAdminRoutes(r, h)
			})
		})

		// Operações longas (120s)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(120 * time.Second))

			r.Group(func(r chi.Router) {
				r.Use(RequireAPIScope(domain.APIScopeAnalytics))
				r.Get("/admin/dashboard/meta/live-insights", h.adminDashboardMeta.LiveInsights)
				r.Get("/admin/dashboard/meta/creative-ranking", h.adminDashboardMeta.CreativeRanking)
				r.Get("/admin/dashboard/google/live-insights", h.adminDashboardGoogle.LiveInsights)
				r.Get("/admin/dashboard/google/keyword-ranking", h.adminDashboardGoogle.KeywordRanking)
			})

			r.Group(func(r chi.Router) {
				r.Use(RequireAPIScope(domain.APIScopeWhatsApp))
				registerWhatsAppLongOperationalRoutes(r, h)
			})

			r.Group(func(r chi.Router) {
				r.Use(RequireHumanAuth)
				r.Post("/admin/openai/webhook-mapping", h.adminOpenAI.GenerateMapping)
				r.Post("/admin/system/easypanel", h.setup.ConnectEasyPanel)
				r.Post("/admin/system/update-now", h.setup.TriggerSystemUpdate)
				r.Post("/admin/geoip/refresh", h.adminGeoIP.Refresh)
				r.With(RequireManager).Post("/admin/maintenance/prune", h.adminMaintenance.PruneNow)
				r.With(RequireManager).Post("/admin/maintenance/reset", h.adminMaintenance.Reset)
				r.Post("/admin/pixels/{id}/test-event", h.adminPixelTest.SendTestEvent)
				registerOAuthLongAdminRoutes(r, h)
				registerWhatsAppLongHumanRoutes(r, h)
				registerCampaignsLongRoutes(r, h)
			})
		})
	})
}
