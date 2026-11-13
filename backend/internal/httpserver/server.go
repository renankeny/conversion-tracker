package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/config"
	"github.com/renankeny/conversion-tracker/backend/internal/geoip"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/handlers"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

type Server struct {
	cfg        *config.Config
	router     *chi.Mux
	hub        *wshub.Hub
	calibrator *util.ShardCalibrator
	syncDeps   cacheSyncDeps
}

type Deps struct {
	Cfg            *config.Config
	Pool           *pgxpool.Pool
	GeoIP          *geoip.Lookup
	Enqueuer       *queue.Enqueuer
	JWT            *auth.JWT
	URLs           *util.URLMapper
	AllowedDomains *util.DomainSet
	RedisOpt       asynq.RedisClientOpt
	Secrets        handlers.GatewaySecrets
	Rdb            *redis.Client
	MediaStorage   *storage.MediaStorage
}

func New(deps Deps) (*Server, error) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(TrustedRealIP)
	r.Use(middleware.Recoverer)
	r.Use(buildCORS(deps.Cfg, deps.AllowedDomains))

	rdb := deps.Rdb
	if rdb == nil {
		rdb = redis.NewClient(&redis.Options{
			Addr: deps.RedisOpt.Addr, Username: deps.RedisOpt.Username, Password: deps.RedisOpt.Password, DB: deps.RedisOpt.DB, TLSConfig: deps.RedisOpt.TLSConfig,
		})
	}
	sessions := cache.NewSessionStore(rdb)
	wsTickets := cache.NewWSTicketStore(rdb)
	queryCache := cache.NewQueryCache(rdb)
	rateLimiter := cache.NewRateLimiter(rdb)

	pixels := repo.NewPixelRepo(deps.Pool)
	pixels.SetRedis(rdb)
	users := repo.NewUserRepo(deps.Pool)
	urlMaps := repo.NewURLMappingRepo(deps.Pool)
	sentEvents := repo.NewSentEventRepo(deps.Pool)
	stats := repo.NewStatsRepo(deps.Pool)
	counters := repo.NewEventCounterRepo(deps.Pool)
	apiKeys := repo.NewAPIKeyRepo(deps.Pool, rdb)
	accountRepo := repo.NewAccountRepo(deps.Pool)
	brandingRepo := repo.NewBrandingRepo(deps.Pool)
	_, _ = brandingRepo.Reload(context.Background())

	crmRepo := repo.NewCRMRepo(deps.Pool)
	customWebhooksRepo := repo.NewCustomWebhookRepo(deps.Pool)
	platformTokenRepo := repo.NewPlatformTokenRepo(deps.Pool)
	webhookLogRepo := repo.NewWebhookLogRepo(deps.Pool)

	webhookH := handlers.NewWebhooks(pixels, users, deps.Enqueuer, deps.URLs, deps.Secrets, platformTokenRepo)
	webhookH.SetWebhookLogs(webhookLogRepo)
	webhookH.SetRedis(rdb)

	googleOAuth := googleads.OAuthConfig{
		ClientID: deps.Cfg.GoogleOAuthClientID, ClientSecret: deps.Cfg.GoogleOAuthClientSecret,
		RedirectURI: deps.Cfg.GoogleOAuthRedirectURI, Pool: deps.Pool,
		AppURL: deps.Cfg.AppURL, PublicAPIURL: deps.Cfg.PublicAPIURL,
	}
	googleAdsClient := googleads.New(deps.Cfg.GoogleAdsDeveloperToken, googleOAuth)
	googleAdsAccountRepo := repo.NewGoogleAdsAccountRepo(deps.Pool)
	metaOAuthCfg := metaoauth.Config{Pool: deps.Pool, AppURL: deps.Cfg.AppURL, PublicAPIURL: deps.Cfg.PublicAPIURL}
	metaOAuthClient := metaoauth.NewClient(metaOAuthCfg)
	metaAccountRepo := repo.NewMetaAccountRepo(deps.Pool)
	whatsappRepo := repo.NewWhatsAppRepo(deps.Pool)
	webhookH.SetWhatsAppRepo(whatsappRepo)

	adminPixelH := handlers.NewAdminPixels(pixels)
	adminPixelH.SetDNSTarget(deps.Cfg.TrackingTargetHost)
	adminPixelH.SetDomainSet(deps.AllowedDomains)
	adminPixelH.SetSecurityDeps(deps.Pool, googleAdsAccountRepo)
	adminPixelH.SetMetaAccountsRepo(metaAccountRepo)

	adminGeoIPH := handlers.NewAdminGeoIP(deps.Cfg.MaxMindLicenseKey, deps.Cfg.GeoIPDBPath, deps.GeoIP)
	adminGeoIPH.StartScheduler(context.Background())
	hub := wshub.New(rdb)

	trackerH, err := handlers.NewTracker(pixels, deps.Cfg.PublicAPIURL)
	if err != nil {
		return nil, fmt.Errorf("init tracker handler: %w", err)
	}
	trackerH.SetBranding(brandingRepo)

	cal, _ := util.NewShardCalibrator(deps.Pool, deps.Cfg.AppURL, deps.Cfg.PublicAPIURL)
	setupH := handlers.NewSetupHandler(deps.Pool, deps.JWT, brandingRepo)
	setupH.SetCalibrator(cal)

	adminCRM := handlers.NewAdminCRM(crmRepo, users, deps.Enqueuer)
	adminCRM.SetSecurityDeps(pixels, googleAdsAccountRepo)
	adminCRM.SetWhatsAppRepo(whatsappRepo)
	adminCRM.SetMetaAccountRepo(metaAccountRepo)
	adminCRM.SetRedis(rdb)
	adminURL := handlers.NewAdminURLMappings(urlMaps, deps.URLs)
	adminURL.SetPool(deps.Pool)
	adminURL.SetRedis(rdb)
	adminTrackedUsers := handlers.NewAdminTrackedUsers(users, sentEvents)
	adminTrackedUsers.SetPixels(pixels)

	wsEventsH := handlers.NewWSEvents(hub, deps.JWT)
	wsEventsH.SetOriginPolicy(deps.Cfg.AppURL, deps.Cfg.Env == "production")
	wsEventsH.SetSessionDeps(wsTickets, sessions)

	healthH := handlers.NewHealth(deps.Pool, deps.RedisOpt)
	healthH.SetRedis(rdb)
	brandingH := handlers.NewBrandingHandler(brandingRepo)
	brandingH.SetRedis(rdb)
	eventsH := handlers.NewEvents(pixels, users, sentEvents, deps.GeoIP, deps.Enqueuer, deps.URLs, crmRepo)
	eventsH.SetRedis(rdb)
	eventsH.SetTrackingTargetHost(deps.Cfg.TrackingTargetHost)
	apiEventsH := handlers.NewAPIEvents(apiKeys, pixels, users, sentEvents, deps.GeoIP, deps.Enqueuer, deps.URLs, crmRepo)
	apiEventsH.SetRedis(rdb)
	pubDashH := handlers.NewPublicDashboard(pixels, stats, counters, sentEvents)
	pubDashH.SetQueryCache(queryCache)
	adminAuthH := handlers.NewAdminAuth(deps.Pool, deps.JWT)
	adminAuthH.SetSessionDeps(sessions, wsTickets)
	adminGadsH := handlers.NewAdminGoogleAdsAccounts(googleAdsAccountRepo, googleAdsClient, googleOAuth, deps.Cfg.AppURL)
	adminGadsH.SetRedis(rdb)
	adminMetaH := handlers.NewAdminMetaAccounts(metaAccountRepo, metaOAuthClient, metaOAuthCfg, deps.Cfg.AppURL)
	adminMetaH.SetRedis(rdb)
	adminUsersH := handlers.NewAdminUsers(deps.Pool)
	adminUsersH.SetSessions(sessions)
	adminDashMetaH := handlers.NewAdminDashboardMeta(pixels)
	adminDashMetaH.SetStatsRepo(stats)
	adminDashMetaH.SetQueryCache(queryCache)
	adminDashMetaH.SetMetaOAuth(metaAccountRepo, metaOAuthClient)
	adminDashGoogleH := handlers.NewAdminDashboardGoogle(pixels, googleAdsAccountRepo, googleAdsClient)
	adminDashGoogleH.SetStatsRepo(stats)
	adminDashGoogleH.SetQueryCache(queryCache)
	adminPixelTestH := handlers.NewAdminPixelTestEvent(pixels, deps.Cfg.MetaAPIVersion, deps.Cfg.MetaPartnerAgent, deps.Cfg.TikTokPartnerName)
	adminPixelTestH.SetMetaOAuth(metaAccountRepo, metaOAuthClient)

	rh := routeHandlers{
		health:              healthH,
		tracker:             trackerH,
		traefik:             handlers.NewTraefikConfig(pixels, deps.Cfg.TraefikHTTPSecret),
		branding:            brandingH,
		setup:               setupH,
		events:              eventsH,
		webhooks:            webhookH,
		webhookEvolution:    handlers.NewWebhookEvolution(whatsappRepo, deps.Enqueuer),
		whatsappMedia:       handlers.NewWhatsAppMediaHandler(whatsappRepo, deps.MediaStorage, deps.Cfg.JWTSecret),
		publicWhatsApp:      handlers.NewPublicWhatsApp(whatsappRepo, deps.Cfg.PublicAPIURL),
		adminWAServers:      handlers.NewAdminWhatsAppServers(whatsappRepo),
		adminWAInstances:    handlers.NewAdminWhatsAppInstances(whatsappRepo, pixels, deps.Cfg.PublicAPIURL),
		adminWAChats:        handlers.NewAdminWhatsAppChats(whatsappRepo, deps.Enqueuer, deps.Cfg.PublicAPIURL, deps.Cfg.JWTSecret),
		adminWAStats:        handlers.NewAdminWhatsAppStats(whatsappRepo),
		customWebhook:       handlers.NewCustomWebhookHandler(customWebhooksRepo, webhookH, webhookLogRepo),
		publicDashboard:     pubDashH,
		adminAuth:           adminAuthH,
		adminGoogleAds:      adminGadsH,
		adminMetaAds:        adminMetaH,
		adminAlerts:         handlers.NewAdminAlerts(pixels, googleAdsAccountRepo, webhookLogRepo, counters),
		adminPixels:         adminPixelH,
		adminPixelTest:      adminPixelTestH,
		adminOpenAI:         handlers.NewAdminOpenAI(deps.Pool),
		adminGeoIP:          adminGeoIPH,
		adminMaintenance:    handlers.NewAdminMaintenance(deps.Pool, accountRepo, deps.Cfg.SentEventsRetentionDays, deps.Cfg.EventCountersRetentionDays),
		adminCustomWebhooks: handlers.NewAdminCustomWebhooks(customWebhooksRepo, webhookLogRepo),
		adminPlatformTokens: handlers.NewPlatformTokens(platformTokenRepo),
		adminSalesConfirm:   handlers.NewAdminSalesConfirmation(sentEvents, users, deps.Enqueuer),
		adminUsers:          adminUsersH,
		adminAccounts:       handlers.NewAdminAccounts(accountRepo),
		adminCRM:            adminCRM,
		adminURL:            adminURL,
		adminSent:           handlers.NewAdminSentEvents(sentEvents, deps.Enqueuer),
		adminWebhookLogs:    handlers.NewAdminWebhookLogs(webhookLogRepo, webhookH),
		adminTrackedUsers:   adminTrackedUsers,
		adminStats:          handlers.NewAdminStats(stats),
		adminCounters:       handlers.NewAdminCounters(counters),
		adminAPIKeys:         handlers.NewAdminAPIKeys(apiKeys),
		adminDashboardMeta:   adminDashMetaH,
		adminDashboardGoogle: adminDashGoogleH,
		apiEvents:            apiEventsH,
		wsEvents:            wsEventsH,
		apiKeysRepo:         apiKeys,
		sessions:            sessions,
		rateLimiter:         rateLimiter,
		calibrator:          cal,
	}

	registerRoutes(r, deps, rh)
	return &Server{
		cfg: deps.Cfg, router: r, hub: hub, calibrator: cal,
		syncDeps: cacheSyncDeps{rdb: rdb, pixels: pixels, branding: brandingRepo, urlMaps: urlMaps, urls: deps.URLs, domains: deps.AllowedDomains},
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	go s.hub.Run(ctx)
	startCacheInvalidationSync(ctx, s.syncDeps)
	if s.calibrator != nil {
		s.calibrator.StartCompactionLoop(ctx)
	}

	srv := &http.Server{
		Addr: s.cfg.HTTPAddr, Handler: s.router, ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	return srv.ListenAndServe()
}
