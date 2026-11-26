package httpserver

import "github.com/go-chi/chi/v5"

// registerHumanFastAdminRoutes concentra todas as rotas administrativas sensíveis
// que exigem obrigatoriamente sessão humana (`RequireHumanAuth`) e retornam 403 para chaves `trk_*`.
func registerHumanFastAdminRoutes(r chi.Router, h routeHandlers) {
	r.Post("/admin/logout", h.adminAuth.Logout)
	r.Post("/admin/ws-ticket", h.adminAuth.IssueWSTicket)
	r.Get("/admin/me", h.adminAuth.Me)
	r.Put("/admin/me", h.adminUsers.UpdateMyProfile)
	r.Post("/admin/me/password", h.adminUsers.ChangeMyPassword)
	r.Put("/admin/me/preferences", h.adminUsers.UpdateMyPreferences)
	r.Put("/admin/branding", h.branding.UpdateAdmin)
	r.Get("/admin/system/health-alerts", h.adminAlerts.GetHealthAlerts)
	r.Get("/admin/system/easypanel", h.setup.GetEasyPanelStatus)

	r.Delete("/admin/pixels/{id}", h.adminPixels.Delete)
	r.Post("/admin/pixels/{id}/verify-dns", h.adminPixels.VerifyDNS)
	r.Post("/admin/pixels/{id}/regenerate-token", h.adminPixels.RegenerateToken)
	r.Put("/admin/pixels/{id}/dashboard-config", h.adminPixels.UpdateDashboardConfig)
	r.Post("/admin/pixels/{id}/dashboard-share-link", h.adminPixels.GenerateDashboardShareLink)
	r.Delete("/admin/pixels/{id}/dashboard-share-link", h.adminPixels.RevokeDashboardShareLink)

	registerOAuthAdminRoutes(r, h)

	r.Get("/admin/geoip/status", h.adminGeoIP.Status)
	r.Get("/admin/maintenance/status", h.adminMaintenance.Status)
	r.With(RequireManager).Put("/admin/maintenance/retention", h.adminMaintenance.UpdateRetention)

	r.Get("/admin/integrations/platforms/summary", h.adminPlatformTokens.Summary)
	r.Get("/admin/integrations/platforms/tokens/{platform}", h.adminPlatformTokens.GetToken)
	r.Put("/admin/integrations/platforms/tokens/{platform}", h.adminPlatformTokens.SetToken)

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
	r.With(RequireManager).Put("/admin/events-filter", h.adminAccounts.UpdateEventsFilter)

	r.With(RequireManager).Delete("/admin/sent-events", h.adminSent.BulkDelete)
	r.With(RequireManager).Delete("/admin/tracked-users/{id}", h.adminTrackedUsers.Delete)

	r.Get("/admin/api-keys", h.adminAPIKeys.List)
	r.Post("/admin/api-keys", h.adminAPIKeys.Create)
	r.Put("/admin/api-keys/{id}", h.adminAPIKeys.Update)
	r.Post("/admin/api-keys/{id}/regenerate", h.adminAPIKeys.Regenerate)
	r.Delete("/admin/api-keys/{id}", h.adminAPIKeys.Revoke)

	registerNotificationAdminRoutes(r, h)
	registerWhatsAppHumanRoutes(r, h)
	registerCampaignsFastRoutes(r, h)
}
