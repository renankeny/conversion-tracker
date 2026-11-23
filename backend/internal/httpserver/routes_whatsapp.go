package httpserver

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/handlers"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/whatsappai"
)

func attachWhatsAppHandlers(
	rh *routeHandlers,
	deps Deps,
	rdb *redis.Client,
	qcache *cache.QueryCache,
	waRepo *repo.WhatsAppRepo,
	pixels *repo.PixelRepo,
	users *repo.UserRepo,
	crmRepo *repo.CRMRepo,
	metaAcc *repo.MetaAccountRepo,
	notifySvc *notify.Service,
) {
	aiRepo := repo.NewWhatsAppAIRepo(deps.Pool)
	adminUsersRepo := repo.NewAdminUserRepo(deps.Pool)
	analyzer := &whatsappai.Analyzer{
		AI:       aiRepo,
		WhatsApp: waRepo,
		Pixels:   pixels,
		Users:    users,
		CRM:      crmRepo,
		MetaAcc:  metaAcc,
		Storage:  deps.MediaStorage,
		Enqueuer: deps.Enqueuer,
		Notify:   notifySvc,
		Redis:    rdb,
	}

	chatsH := handlers.NewAdminWhatsAppChats(waRepo, deps.Enqueuer, deps.Cfg.PublicAPIURL, deps.Cfg.JWTSecret)
	chatsH.SetAIRepo(aiRepo)
	chatsH.SetUsersRepo(users)
	chatsH.SetMetaAndCRMDeps(pixels, metaAcc, crmRepo, rdb)

	rh.webhookEvolution = handlers.NewWebhookEvolution(waRepo, deps.Enqueuer)
	rh.whatsappMedia = handlers.NewWhatsAppMediaHandler(waRepo, deps.MediaStorage, deps.Cfg.JWTSecret)
	rh.publicWhatsApp = handlers.NewPublicWhatsApp(waRepo, deps.Cfg.PublicAPIURL)
	rh.adminWAServers = handlers.NewAdminWhatsAppServers(waRepo)
	rh.adminWAInstances = handlers.NewAdminWhatsAppInstances(waRepo, pixels, deps.Cfg.PublicAPIURL)
	rh.adminWAInstances.SetNotify(notifySvc)
	rh.adminWAChats = chatsH
	waStatsH := handlers.NewAdminWhatsAppStats(waRepo)
	waStatsH.SetQueryCache(qcache)
	rh.adminWAStats = waStatsH
	rh.adminWhatsAppAI = handlers.NewAdminWhatsAppAI(aiRepo, adminUsersRepo, analyzer, rdb)

	publicLinkBaseURL := deps.Cfg.AppURL
	if publicLinkBaseURL == "" {
		publicLinkBaseURL = deps.Cfg.PublicAPIURL
	}

	linksRepo := repo.NewTrackableLinkRepo(deps.Pool)
	rh.publicTrackableLinks = handlers.NewPublicTrackableLinks(
		linksRepo, pixels, users, deps.GeoIP, deps.Enqueuer, deps.URLs,
		rdb, publicLinkBaseURL, deps.Cfg.TrackingTargetHost,
	)
	rh.adminTrackableLinks = handlers.NewAdminTrackableLinks(
		linksRepo, pixels, waRepo, qcache, rdb, publicLinkBaseURL,
	)
	if rh.publicDashboard != nil {
		rh.publicDashboard.SetTrackableLinksRepo(linksRepo, publicLinkBaseURL)
		rh.publicDashboard.SetPortalDeps(rh.adminCRM, waRepo, aiRepo, deps.Cfg.JWTSecret, deps.Cfg.PublicAPIURL)
	}
}

func registerWhatsAppPublicRoutes(r chi.Router, h routeHandlers) {
	r.Get("/w/{slug}", h.publicTrackableLinks.ServeRedirect)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(25 * time.Second))
		r.Get("/public/whatsapp/{token}/status", h.publicWhatsApp.ConnectStatus)
		r.Post("/public/whatsapp/{token}/connect", h.publicWhatsApp.InitConnect)
		r.Post("/public/whatsapp/{token}/pair", h.publicWhatsApp.PairPhone)
	})
}

func registerTrackableLinksOperationalRoutes(r chi.Router, h routeHandlers) {
	r.Get("/admin/pixels/{id}/trackable-links", h.adminTrackableLinks.ListByPixel)
	r.Post("/admin/pixels/{id}/trackable-links", h.adminTrackableLinks.Create)
	r.Put("/admin/pixels/{id}/trackable-links/{linkID}", h.adminTrackableLinks.Update)
	r.Delete("/admin/pixels/{id}/trackable-links/{linkID}", h.adminTrackableLinks.Delete)
	r.Get("/admin/trackable-links/stats", h.adminTrackableLinks.DashboardStats)
}

func registerWhatsAppOperationalRoutes(r chi.Router, h routeHandlers) {
	r.Get("/admin/whatsapp/instances", h.adminWAInstances.List)
	r.Put("/admin/whatsapp/instances/{id}", h.adminWAInstances.Update)

	r.Get("/admin/whatsapp/chats", h.adminWAChats.ListChats)
	r.Get("/admin/whatsapp/conversations", h.adminWAChats.ListAuditConversations)
	r.Get("/admin/whatsapp/conversations/waiting-count", h.adminWAChats.WaitingCount)
	r.Get("/admin/whatsapp/chats/by-user/{userID}", h.adminWAChats.GetByTrackedUser)
	r.Get("/admin/whatsapp/chats/by-lead/{leadID}", h.adminWAChats.GetByCRMLead)
	r.Get("/admin/whatsapp/chats/{chatID}/messages", h.adminWAChats.GetMessages)
	r.Post("/admin/whatsapp/chats/{chatID}/convert", h.adminWAChats.Convert)

	r.Get("/admin/whatsapp/ai-settings", h.adminWhatsAppAI.GetSettings)
	r.Get("/admin/whatsapp-stats/overview", h.adminWAStats.Overview)
	r.Get("/admin/whatsapp-stats/heatmap", h.adminWAStats.Heatmap)
	r.Get("/admin/whatsapp-stats/ctwa-leads", h.adminWAStats.CTWALeads)
}

func registerWhatsAppHumanRoutes(r chi.Router, h routeHandlers) {
	r.Get("/admin/whatsapp/server", h.adminWAServers.Get)
	r.Put("/admin/whatsapp/server", h.adminWAServers.Upsert)
	r.Delete("/admin/whatsapp/server", h.adminWAServers.Delete)

	r.Put("/admin/whatsapp/ai-settings", h.adminWhatsAppAI.UpdateSettings)
	r.Post("/admin/whatsapp/ai-events/{id}/contest", h.adminWhatsAppAI.ContestEvent)
}

func registerWhatsAppLongOperationalRoutes(r chi.Router, h routeHandlers) {
	r.Post("/admin/whatsapp/ai-settings/run-now", h.adminWhatsAppAI.RunNow)
	r.Post("/admin/whatsapp/instances", h.adminWAInstances.Create)
	r.Post("/admin/whatsapp/instances/{id}/connect", h.adminWAInstances.Connect)
	r.Get("/admin/whatsapp/instances/{id}/qr", h.adminWAInstances.QrCode)
	r.Post("/admin/whatsapp/instances/{id}/pair", h.adminWAInstances.Pair)
	r.Get("/admin/whatsapp/instances/{id}/status", h.adminWAInstances.RefreshStatus)
	r.Post("/admin/whatsapp/instances/{id}/disconnect", h.adminWAInstances.Disconnect)
	r.Delete("/admin/whatsapp/instances/{id}", h.adminWAInstances.Delete)
}

func registerWhatsAppLongHumanRoutes(r chi.Router, h routeHandlers) {
	r.Post("/admin/whatsapp/server/test", h.adminWAServers.Test)
}
