package httpserver

import (
	"time"

	"github.com/go-chi/chi/v5"
)

func registerWebhookRoutes(r chi.Router, h routeHandlers) {
	rlWebhooks := h.rateLimiter.LimitWebhooks(3000, 1*time.Minute)

	r.Group(func(r chi.Router) {
		r.Use(MaxBodyBytes(2 * 1024 * 1024))
		r.Use(rlWebhooks)

		r.Post("/webhooks/appmax", h.webhooks.Appmax)
		r.Post("/webhooks/braip", h.webhooks.Braip)
		r.Post("/webhooks/cakto", h.webhooks.Cakto)
		r.Post("/webhooks/cartpanda", h.webhooks.Cartpanda)
		r.Post("/webhooks/doppus", h.webhooks.Doppus)
		r.Post("/webhooks/eduzz", h.webhooks.Eduzz)
		r.Post("/webhooks/greenn", h.webhooks.Greenn)
		r.Post("/webhooks/guru", h.webhooks.Guru)
		r.Post("/webhooks/hotmart", h.webhooks.Hotmart)
		r.Post("/webhooks/hubla", h.webhooks.Hubla)
		r.Post("/webhooks/kirvano", h.webhooks.Kirvano)
		r.Post("/webhooks/kiwify", h.webhooks.Kiwify)
		r.Post("/webhooks/lastlink", h.webhooks.Lastlink)
		r.Post("/webhooks/monetizze", h.webhooks.Monetizze)
		r.Post("/webhooks/pagtrust", h.webhooks.PagTrust)
		r.Post("/webhooks/pepper", h.webhooks.Pepper)
		r.Post("/webhooks/perfectpay", h.webhooks.PerfectPay)
		r.Post("/webhooks/ticto", h.webhooks.Ticto)
		r.Post("/webhooks/vega", h.webhooks.Vega)
		r.Post("/webhooks/yampi", h.webhooks.Yampi)
		r.Post("/webhooks/custom/{id}", h.customWebhook.Handle)
	})

	// Webhooks de WhatsApp (Evolution Go, API Premium e Uazapi):
	// aceitamos até 25 MB e removemos thumbnails/base64 pesados antes de enfileirar no Redis.
	r.Group(func(r chi.Router) {
		r.Use(MaxBodyBytes(25 * 1024 * 1024))
		r.Use(rlWebhooks)
		r.Post("/webhooks/evolution/{secret}", h.webhookEvolution.HandleBySecret)
		r.Post("/webhooks/evolution-go/{instanceName}", h.webhookEvolution.HandleLegacyByName)
		r.Post("/webhooks/whatsapp/{provider}/{secret}", h.webhookEvolution.HandleUazapiBySecret)
		r.Post("/webhooks/whatsapp/{provider}/{secret}/*", h.webhookEvolution.HandleUazapiBySecret)
		r.Post("/webhooks/api-premium/{secret}", h.webhookEvolution.HandleUazapiBySecret)
		r.Post("/webhooks/api-premium/{secret}/*", h.webhookEvolution.HandleUazapiBySecret)
		r.Post("/webhooks/uazapi/{secret}", h.webhookEvolution.HandleUazapiBySecret)
		r.Post("/webhooks/uazapi/{secret}/*", h.webhookEvolution.HandleUazapiBySecret)
	})
}
