package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/campaignrules"
	"github.com/renankeny/conversion-tracker/backend/internal/campaigns"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/handlers"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
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
	notifySvc *notify.Service,
	waRepo *repo.WhatsAppRepo,
) {
	refundRepo := repo.NewRefundRepo(deps.Pool)
	if rh.webhooks != nil {
		rh.webhooks.SetRefundRepo(refundRepo)
	}
	attrRepo := repo.NewCampaignsAttributionRepo(deps.Pool)
	svc := campaigns.NewService(pixels, metaAcc, gadsAcc, metaClient, gadsClient, attrRepo, rdb)
	rh.adminCampaigns = handlers.NewAdminCampaigns(svc, pixels)

	rulesRepo := repo.NewCampaignRulesRepo(deps.Pool)
	logsRepo := repo.NewCampaignRuleLogsRepo(deps.Pool)
	runner := campaignrules.NewRunner(svc, rulesRepo, logsRepo, notifySvc, waRepo)
	rh.adminCampaignRules = handlers.NewAdminCampaignRules(rulesRepo, logsRepo, runner)
}

func registerCampaignsFastRoutes(r chi.Router, h routeHandlers) {
	if h.adminCampaigns != nil {
		r.Put("/admin/pixels/{id}/financials", h.adminCampaigns.UpdatePixelFinancials)
	}
	if h.adminCampaignRules != nil {
		r.Get("/admin/campaigns/rules", h.adminCampaignRules.List)
		r.Post("/admin/campaigns/rules", h.adminCampaignRules.Create)
		r.Put("/admin/campaigns/rules/{id}", h.adminCampaignRules.Update)
		r.Patch("/admin/campaigns/rules/{id}/toggle", h.adminCampaignRules.Toggle)
		r.Delete("/admin/campaigns/rules/{id}", h.adminCampaignRules.Delete)
		r.Get("/admin/campaigns/rules/logs", h.adminCampaignRules.ListLogs)
	}
}

func registerCampaignsLongRoutes(r chi.Router, h routeHandlers) {
	if h.adminCampaigns != nil {
		r.Get("/admin/campaigns/overview", h.adminCampaigns.Overview)
		r.Post("/admin/campaigns/status", h.adminCampaigns.MutateStatus)
		r.Post("/admin/campaigns/budget", h.adminCampaigns.MutateBudget)
	}
	if h.adminCampaignRules != nil {
		r.Post("/admin/campaigns/rules/{id}/run", h.adminCampaignRules.RunNow)
		r.Post("/admin/campaigns/rules/logs/{id}/undo", h.adminCampaignRules.UndoLog)
	}
}
