package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/campaigns"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/handlers"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func attachCampaignsHandlers(
	rh *routeHandlers,
	deps Deps,
	rdb *redis.Client,
	pixels *repo.PixelRepo,
	metaAcc *repo.MetaAccountRepo,
	gadsAcc *repo.GoogleAdsAccountRepo,
	metaClient *metaoauth.Client,
	gadsClient *googleads.Client,
) {
	refundRepo := repo.NewRefundRepo(deps.Pool)
	if rh.webhooks != nil {
		rh.webhooks.SetRefundRepo(refundRepo)
	}
	attrRepo := repo.NewCampaignsAttributionRepo(deps.Pool)
	svc := campaigns.NewService(pixels, metaAcc, gadsAcc, metaClient, gadsClient, attrRepo, rdb)
	rh.adminCampaigns = handlers.NewAdminCampaigns(svc, pixels)
}

func registerCampaignsFastRoutes(r chi.Router, h routeHandlers) {
	if h.adminCampaigns == nil {
		return
	}
	r.Put("/admin/pixels/{id}/financials", h.adminCampaigns.UpdatePixelFinancials)
}

func registerCampaignsLongRoutes(r chi.Router, h routeHandlers) {
	if h.adminCampaigns == nil {
		return
	}
	r.Get("/admin/campaigns/overview", h.adminCampaigns.Overview)
	r.Post("/admin/campaigns/status", h.adminCampaigns.MutateStatus)
	r.Post("/admin/campaigns/budget", h.adminCampaigns.MutateBudget)
}
