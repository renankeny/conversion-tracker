package httpserver

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/handlers"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type routeHandlers struct {
	health              *handlers.Health
	tracker             *handlers.TrackerHandler
	traefik             http.Handler
	branding            *handlers.BrandingHandler
	setup               *handlers.SetupHandler
	events              *handlers.Events
	webhooks            *handlers.Webhooks
	webhookEvolution    *handlers.WebhookEvolution
	whatsappMedia       *handlers.WhatsAppMediaHandler
	publicWhatsApp      *handlers.PublicWhatsApp
	adminWAServers      *handlers.AdminWhatsAppServers
	adminWAInstances    *handlers.AdminWhatsAppInstances
	adminWAChats        *handlers.AdminWhatsAppChats
	adminWAStats        *handlers.AdminWhatsAppStats
	customWebhook       *handlers.CustomWebhookHandler
	publicDashboard     *handlers.PublicDashboard
	adminAuth           *handlers.AdminAuth
	adminGoogleAds      *handlers.AdminGoogleAdsAccounts
	adminMetaAds        *handlers.AdminMetaAccounts
	adminAlerts         *handlers.AdminAlerts
	adminPixels         *handlers.AdminPixels
	adminPixelTest      *handlers.AdminPixelTestEvent
	adminOpenAI         *handlers.AdminOpenAI
	adminGeoIP          *handlers.AdminGeoIP
	adminMaintenance    *handlers.AdminMaintenance
	adminCustomWebhooks *handlers.AdminCustomWebhooks
	adminPlatformTokens *handlers.PlatformTokens
	adminSalesConfirm   *handlers.AdminSalesConfirmation
	adminUsers          *handlers.AdminUsers
	adminAccounts       *handlers.AdminAccounts
	adminCRM            *handlers.AdminCRM
	adminURL            *handlers.AdminURLMappings
	adminSent           *handlers.AdminSentEvents
	adminWebhookLogs    *handlers.AdminWebhookLogs
	adminTrackedUsers   *handlers.AdminTrackedUsers
	adminStats          *handlers.AdminStats
	adminCounters       *handlers.AdminCounters
	adminAPIKeys        *handlers.AdminAPIKeys
	adminDashboardMeta   *handlers.AdminDashboardMeta
	adminDashboardGoogle *handlers.AdminDashboardGoogle
	apiEvents            *handlers.APIEvents
	wsEvents            *handlers.WSEvents
	apiKeysRepo         *repo.APIKeyRepo
	sessions            *cache.SessionStore
	rateLimiter         *cache.RateLimiter
	calibrator          *util.ShardCalibrator
}

type cacheSyncDeps struct {
	rdb      *redis.Client
	pixels   *repo.PixelRepo
	branding *repo.BrandingRepo
	urlMaps  *repo.URLMappingRepo
	urls     *util.URLMapper
	domains  *util.DomainSet
}

func startCacheInvalidationSync(ctx context.Context, d cacheSyncDeps) {
	if d.rdb == nil {
		return
	}
	go cache.RunInvalidationSubscriber(ctx, d.rdb, func(evCtx context.Context, ev cache.InvalidateEvent) {
		switch ev.Kind {
		case "pixel":
			if d.pixels != nil && d.pixels.PixelCache() != nil {
				d.pixels.PixelCache().InvalidateKeys(evCtx, ev.Keys...)
			}
			if d.pixels != nil && d.domains != nil {
				if list, err := d.pixels.ActiveDomains(evCtx); err == nil {
					d.domains.Set(list)
				}
			}
		case "branding":
			if d.branding != nil {
				if _, err := d.branding.Reload(evCtx); err != nil {
					slog.Warn("branding pubsub reload failed", "err", err)
				}
			}
		case "url_mappings":
			if d.urlMaps != nil && d.urls != nil {
				if byAcc, err := d.urlMaps.ActiveByAccountMap(evCtx); err == nil {
					d.urls.SetByAccount(byAcc)
				}
			}
		}
	})
}
