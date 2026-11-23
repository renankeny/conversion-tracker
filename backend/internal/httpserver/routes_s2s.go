package httpserver

import (
	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// registerS2SFastRoutes registra os 8 módulos operacionais rápidos (15s),
// validando escopo granular e modo somente leitura por chave `trk_*`.
func registerS2SFastRoutes(r chi.Router, h routeHandlers) {
	// 1. Eventos e Auditoria
	r.Group(func(r chi.Router) {
		r.Use(RequireAPIScope(domain.APIScopeEvents))
		r.Get("/admin/sent-events", h.adminSent.List)
		r.Get("/admin/sent-events/grouped", h.adminSent.ListGrouped)
		r.Get("/admin/sent-events/by-event-id/{event_id}", h.adminSent.GetByEventID)
		r.Get("/admin/sent-events/by-event-id", h.adminSent.GetByEventID)
		r.With(RequireManagerOrAPIKey).Post("/admin/sent-events/retry", h.adminSent.Retry)
	})

	// 2. CRM e Funil 360
	r.Group(func(r chi.Router) {
		r.Use(RequireAPIScope(domain.APIScopeCRM))
		r.Get("/admin/ext-api/crm/stages", h.adminCRM.APIListStages)
		r.Post("/admin/ext-api/crm/leads", h.adminCRM.APICreateLead)
		r.Get("/admin/crm/board", h.adminCRM.GetBoard)
		r.Post("/admin/crm/stages", h.adminCRM.CreateStage)
		r.Put("/admin/crm/stages/reorder", h.adminCRM.ReorderStages)
		r.Put("/admin/crm/stages/{id}", h.adminCRM.UpdateStage)
		r.Delete("/admin/crm/stages/{id}", h.adminCRM.DeleteStage)
		r.Post("/admin/crm/leads", h.adminCRM.APICreateLead)
		r.Get("/admin/crm/leads/{id}", h.adminCRM.GetLeadDetails)
		r.Put("/admin/crm/leads/{id}", h.adminCRM.UpdateLead)
		r.Put("/admin/crm/leads/{id}/move", h.adminCRM.MoveLead)
		r.Delete("/admin/crm/leads/{id}", h.adminCRM.DeleteLead)
		r.Post("/admin/crm/leads/{id}/notes", h.adminCRM.AddNote)
		r.Delete("/admin/crm/notes/{note_id}", h.adminCRM.DeleteNote)
		r.Get("/admin/crm/tags", h.adminCRM.ListTags)
		r.Post("/admin/crm/tags", h.adminCRM.CreateTag)
		r.Delete("/admin/crm/tags/{tag_id}", h.adminCRM.DeleteTag)
		r.Post("/admin/crm/leads/{id}/tags/{tag_id}", h.adminCRM.AssignTag)
		r.Delete("/admin/crm/leads/{id}/tags/{tag_id}", h.adminCRM.RemoveTag)
	})

	// 3. Usuários Rastreáveis e Jornada
	r.Group(func(r chi.Router) {
		r.Use(RequireAPIScope(domain.APIScopeUsers))
		r.Get("/admin/tracked-users", h.adminTrackedUsers.List)
		r.Get("/admin/tracked-users/lookup", h.adminTrackedUsers.Lookup)
		r.Get("/admin/tracked-users/journey", h.adminTrackedUsers.Journey)
		r.Get("/admin/tracked-users/{id}", h.adminTrackedUsers.Get)
		r.With(RequireManagerOrAPIKey).Put("/admin/tracked-users/{id}", h.adminTrackedUsers.Update)
	})

	// 4. WhatsApp e IA Comercial
	r.Group(func(r chi.Router) {
		r.Use(RequireAPIScope(domain.APIScopeWhatsApp))
		registerWhatsAppOperationalRoutes(r, h)
	})

	// 5. Pixels e Links Rastreáveis
	r.Group(func(r chi.Router) {
		r.Use(RequireAPIScope(domain.APIScopePixels))
		r.Get("/admin/pixels", h.adminPixels.List)
		r.Get("/admin/pixels/defaults", h.adminPixels.Defaults)
		r.Get("/admin/pixels/{id}", h.adminPixels.Get)
		r.Post("/admin/pixels", h.adminPixels.Create)
		r.Put("/admin/pixels/{id}", h.adminPixels.Update)
		registerTrackableLinksOperationalRoutes(r, h)
	})

	// 6. Relatórios, Contadores e Contas Conectadas
	r.Group(func(r chi.Router) {
		r.Use(RequireAPIScope(domain.APIScopeAnalytics))
		r.Get("/admin/stats", h.adminStats.Overview)
		r.Get("/admin/stats/geo", h.adminStats.Geo)
		r.Get("/admin/stats/geo-by-event", h.adminStats.GeoByEvent)
		r.Get("/admin/counters/summary", h.adminCounters.Summary)
		r.Get("/admin/counters/series", h.adminCounters.TimeSeries)
		registerOAuthReadRoutes(r, h)
	})

	// 7. Confirmação de Vendas
	r.Group(func(r chi.Router) {
		r.Use(RequireAPIScope(domain.APIScopeSales))
		r.Get("/admin/sales-confirmations", h.adminSalesConfirm.List)
		r.Post("/admin/sales-confirmations/confirm", h.adminSalesConfirm.Confirm)
	})

	// 8. Checkouts e Mapeamento de URLs
	r.Group(func(r chi.Router) {
		r.Use(RequireAPIScope(domain.APIScopeWebhooks))
		r.Get("/admin/webhook-logs", h.adminWebhookLogs.List)
		r.Post("/admin/webhook-logs/{id}/replay", h.adminWebhookLogs.Replay)
		r.Get("/admin/custom-webhooks", h.adminCustomWebhooks.List)
		r.Post("/admin/custom-webhooks", h.adminCustomWebhooks.Create)
		r.Get("/admin/custom-webhooks/{id}", h.adminCustomWebhooks.Get)
		r.Put("/admin/custom-webhooks/{id}", h.adminCustomWebhooks.Update)
		r.Delete("/admin/custom-webhooks/{id}", h.adminCustomWebhooks.Delete)
		r.Get("/admin/custom-webhooks/{id}/logs", h.adminCustomWebhooks.Logs)
		r.Get("/admin/url-mappings", h.adminURL.List)
		r.Post("/admin/url-mappings", h.adminURL.Create)
		r.Put("/admin/url-mappings/{id}", h.adminURL.Update)
		r.Delete("/admin/url-mappings/{id}", h.adminURL.Delete)
	})
}
