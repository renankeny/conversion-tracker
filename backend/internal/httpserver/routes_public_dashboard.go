package httpserver

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func registerPublicDashboardRoutes(r chi.Router, h routeHandlers, rlPublicDash func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(rlPublicDash)
		r.Route("/public/dashboard/{token}", func(r chi.Router) {
			// Consultas locais rápidas (Postgres + Redis) com timeout de 15s
			r.Group(func(r chi.Router) {
				r.Use(middleware.Timeout(15 * time.Second))
				r.Get("/meta", h.publicDashboard.Meta)
				r.Get("/summary", h.publicDashboard.Summary)
				r.Get("/series", h.publicDashboard.Series)
				r.Get("/geo", h.publicDashboard.Geo)
				r.Get("/geo-by-event", h.publicDashboard.GeoByEvent)
				r.Get("/purchases", h.publicDashboard.Purchases)
				r.Get("/revenue", h.publicDashboard.Revenue)
				r.Get("/trackable-links", h.publicDashboard.TrackableLinks)

				// Portal do Cliente Whitelabel: Conversas & Chat somente-leitura
				r.Get("/conversations", h.publicDashboard.Conversations)
				r.Get("/chats/{chatID}/messages", h.publicDashboard.ConversationMessages)

				// Portal do Cliente Whitelabel: CRM (leitura + mutações quando permitido no Pixel)
				r.Get("/crm/board", h.publicDashboard.CRMBoard)
				r.Get("/crm/leads/{leadID}", h.publicDashboard.CRMLeadDetails)
				r.Get("/crm/leads/{leadID}/chat", h.publicDashboard.CRMLeadChat)

				r.Group(func(r chi.Router) {
					r.Use(MaxBodyBytes(64 * 1024))
					r.Post("/crm/leads", h.publicDashboard.CRMCreateLead)
					r.Put("/crm/leads/{leadID}", h.publicDashboard.CRMUpdateLead)
					r.Put("/crm/leads/{leadID}/move", h.publicDashboard.CRMMoveLead)
					r.Post("/crm/leads/{leadID}/notes", h.publicDashboard.CRMAddNote)
					r.Delete("/crm/leads/{leadID}/notes/{noteID}", h.publicDashboard.CRMDeleteNote)
					r.Post("/crm/tags", h.publicDashboard.CRMCreateTag)
					r.Post("/crm/leads/{leadID}/tags/{tagID}", h.publicDashboard.CRMAssignTag)
					r.Delete("/crm/leads/{leadID}/tags/{tagID}", h.publicDashboard.CRMRemoveTag)
				})
			})

			// Consultas externas ao vivo (Meta Graph API + Google Ads API) com timeout estendido de 120s
			r.Group(func(r chi.Router) {
				r.Use(middleware.Timeout(120 * time.Second))
				r.Get("/ads-summary", h.publicDashboard.AdsSummary)
				r.Get("/meta-creatives", h.publicDashboard.MetaCreatives)
				r.Get("/google-keywords", h.publicDashboard.GoogleKeywords)
			})
		})
	})
}
