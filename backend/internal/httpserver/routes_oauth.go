package httpserver

import "github.com/go-chi/chi/v5"

func registerOAuthPublicRoutes(r chi.Router, h routeHandlers) {
	r.Get("/admin/google-ads/oauth/callback", h.adminGoogleAds.OAuthCallback)
	r.Get("/admin/meta-ads/oauth/callback", h.adminMetaAds.OAuthCallback)
}

func registerOAuthAdminRoutes(r chi.Router, h routeHandlers) {
	r.Get("/admin/google-ads-accounts", h.adminGoogleAds.List)
	r.Post("/admin/google-ads-accounts", h.adminGoogleAds.Create)
	r.Get("/admin/google-ads-accounts/{id}", h.adminGoogleAds.Get)
	r.Put("/admin/google-ads-accounts/{id}", h.adminGoogleAds.Update)
	r.Delete("/admin/google-ads-accounts/{id}", h.adminGoogleAds.Delete)
	r.Get("/admin/google-ads-accounts/{id}/oauth/start", h.adminGoogleAds.OAuthStart)
	r.Post("/admin/google-ads-accounts/{id}/reuse-oauth", h.adminGoogleAds.ReuseOAuth)
	r.Get("/admin/google-ads-accounts/{id}/customers", h.adminGoogleAds.ListCustomers)
	r.Get("/admin/google-ads-accounts/{id}/conversion-actions", h.adminGoogleAds.ListConversionActions)
	r.Post("/admin/google-ads-accounts/{id}/disconnect", h.adminGoogleAds.Disconnect)

	r.Get("/admin/meta-accounts", h.adminMetaAds.List)
	r.Post("/admin/meta-accounts", h.adminMetaAds.Create)
	r.Get("/admin/meta-accounts/{id}", h.adminMetaAds.Get)
	r.Put("/admin/meta-accounts/{id}", h.adminMetaAds.Update)
	r.Delete("/admin/meta-accounts/{id}", h.adminMetaAds.Delete)
	r.Get("/admin/meta-accounts/{id}/oauth/start", h.adminMetaAds.OAuthStart)
	r.Post("/admin/meta-accounts/{id}/reuse-oauth", h.adminMetaAds.ReuseOAuth)
	r.Post("/admin/meta-accounts/{id}/disconnect", h.adminMetaAds.Disconnect)
	r.Get("/admin/meta-accounts/{id}/ad-accounts", h.adminMetaAds.ListAdAccounts)
	r.Get("/admin/meta-accounts/{id}/pages", h.adminMetaAds.ListPages)
	r.Get("/admin/meta-accounts/{id}/pixels", h.adminMetaAds.ListPixels)
}

