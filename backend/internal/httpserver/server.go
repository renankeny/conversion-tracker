package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/config"
	"github.com/renankeny/conversion-tracker/backend/internal/geoip"
	"github.com/renankeny/conversion-tracker/backend/internal/handlers"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type Server struct {
	cfg    *config.Config
	router *chi.Mux
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
}

func New(deps Deps) (*Server, error) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	corsHandler := buildCORS(deps.Cfg, deps.AllowedDomains)
	r.Use(corsHandler)

	pixels := repo.NewPixelRepo(deps.Pool)
	users := repo.NewUserRepo(deps.Pool)
	urlMaps := repo.NewURLMappingRepo(deps.Pool)
	sentEvents := repo.NewSentEventRepo(deps.Pool)
	stats := repo.NewStatsRepo(deps.Pool)
	counters := repo.NewEventCounterRepo(deps.Pool)
	apiKeys := repo.NewAPIKeyRepo(deps.Pool)
	accountRepo := repo.NewAccountRepo(deps.Pool)

	eventH := handlers.NewEvents(pixels, users, sentEvents, deps.GeoIP, deps.Enqueuer, deps.URLs)
	webhookH := handlers.NewWebhooks(pixels, users, deps.Enqueuer, deps.URLs, deps.Secrets)
	healthH := handlers.NewHealth(deps.Pool, deps.RedisOpt)
	adminAuthH := handlers.NewAdminAuth(deps.Pool, deps.JWT)
	adminPixelH := handlers.NewAdminPixels(pixels)
	adminPixelH.SetDNSTarget(deps.Cfg.TrackingTargetHost)
	adminPixelTestH := handlers.NewAdminPixelTestEvent(pixels, deps.Cfg.MetaAPIVersion, deps.Cfg.MetaPartnerAgent)
	adminGeoIPH := handlers.NewAdminGeoIP(deps.Cfg.MaxMindLicenseKey, deps.Cfg.GeoIPDBPath, deps.GeoIP)
	adminMaintenanceH := handlers.NewAdminMaintenance(deps.Pool, accountRepo, deps.Cfg.SentEventsRetentionDays, deps.Cfg.EventCountersRetentionDays)
	adminUsersH := handlers.NewAdminUsers(deps.Pool)
	adminAccountsH := handlers.NewAdminAccounts(accountRepo)
	adminURLH := handlers.NewAdminURLMappings(urlMaps, deps.URLs)
	adminSentH := handlers.NewAdminSentEvents(sentEvents)
	adminTUH := handlers.NewAdminTrackedUsers(users)
	adminStatsH := handlers.NewAdminStats(stats)
	adminCountersH := handlers.NewAdminCounters(counters)
	adminAPIKeysH := handlers.NewAdminAPIKeys(apiKeys)

	trackerH, err := handlers.NewTracker(pixels, deps.Cfg.PublicAPIURL)
	if err != nil {
		return nil, fmt.Errorf("init tracker handler: %w", err)
	}

	// Rate limiters por IP — evitam abuso dos endpoints públicos.
	// Janela de 1 min com limite por categoria.
	rlEvents := httprate.LimitByIP(120, 1*time.Minute)  // /events/send (até 120/min/IP é generoso pra um single user)
	rlInit := httprate.LimitByIP(60, 1*time.Minute)     // /events/init
	rlAssets := httprate.LimitByIP(600, 1*time.Minute)  // /loader.js, /tracker.js (cacheado, então raramente bate)
	rlLogin := httprate.LimitByIP(10, 1*time.Minute)    // /admin/login (anti brute-force)

	r.Route("/api/v1", func(r chi.Router) {
		// Health: sem rate limit (probes do orquestrador hammer aqui).
		r.Get("/health", healthH.Handler)

		// Assets do tracker.
		r.Group(func(r chi.Router) {
			r.Use(rlAssets)
			r.Get("/loader.js", trackerH.Loader)
			r.Get("/tracker.js", trackerH.Tracker)
		})

		// Eventos públicos. Body limit estrito.
		r.Group(func(r chi.Router) {
			r.Use(MaxBodyBytes(8 * 1024)) // 8 KB pra Init
			r.Use(rlInit)
			r.Post("/events/init", eventH.Init)
		})
		r.Group(func(r chi.Router) {
			r.Use(MaxBodyBytes(64 * 1024)) // 64 KB pra send (PII + UTMs caso tenha)
			r.Use(rlEvents)
			r.Post("/events/send", eventH.Send)
		})

		// Webhooks de gateway: payloads grandes, sem rate limit (o gateway controla).
		r.Group(func(r chi.Router) {
			r.Use(MaxBodyBytes(2 * 1024 * 1024)) // 2 MB
			r.Post("/webhooks/kiwify", webhookH.Kiwify)
			r.Post("/webhooks/hotmart", webhookH.Hotmart)
		})

		// Admin login: separado pra aplicar rate limit anti brute-force.
		r.Group(func(r chi.Router) {
			r.Use(MaxBodyBytes(4 * 1024))
			r.Use(rlLogin)
			r.Post("/admin/login", adminAuthH.Login)
		})

		// Admin protegido por JWT ou API key (trk_*).
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(deps.JWT, apiKeys))
			r.Use(MaxBodyBytes(256 * 1024))

			r.Get("/admin/me", adminAuthH.Me)

			// Pixels
			r.Get("/admin/pixels", adminPixelH.List)
			r.Get("/admin/pixels/defaults", adminPixelH.Defaults)
			r.Get("/admin/pixels/{id}", adminPixelH.Get)
			r.Post("/admin/pixels", adminPixelH.Create)
			r.Put("/admin/pixels/{id}", adminPixelH.Update)
			r.Delete("/admin/pixels/{id}", adminPixelH.Delete)
			r.Post("/admin/pixels/{id}/verify-dns", adminPixelH.VerifyDNS)
			r.Post("/admin/pixels/{id}/test-event", adminPixelTestH.SendTestEvent)

			// GeoIP
			r.Get("/admin/geoip/status", adminGeoIPH.Status)
			r.Post("/admin/geoip/refresh", adminGeoIPH.Refresh)

			// Maintenance (retention)
			r.Get("/admin/maintenance/status", adminMaintenanceH.Status)
			r.Post("/admin/maintenance/prune", adminMaintenanceH.PruneNow)
			r.Put("/admin/maintenance/retention", adminMaintenanceH.UpdateRetention)
			r.Post("/admin/maintenance/reset", adminMaintenanceH.Reset)

			// Admin users (CRUD + change password)
			r.Get("/admin/users", adminUsersH.List)
			r.Post("/admin/users", adminUsersH.Create)
			r.Delete("/admin/users/{id}", adminUsersH.Delete)
			r.Post("/admin/me/password", adminUsersH.ChangeMyPassword)

			// Accounts (multi-tenant)
			r.Get("/admin/accounts", adminAccountsH.List)
			r.Get("/admin/accounts/{id}", adminAccountsH.Get)
			r.Post("/admin/accounts", adminAccountsH.Create)
			r.Put("/admin/accounts/{id}", adminAccountsH.Update)
			r.Delete("/admin/accounts/{id}", adminAccountsH.Delete)

			// URL mappings (black -> white)
			r.Get("/admin/url-mappings", adminURLH.List)
			r.Post("/admin/url-mappings", adminURLH.Create)
			r.Put("/admin/url-mappings/{id}", adminURLH.Update)
			r.Delete("/admin/url-mappings/{id}", adminURLH.Delete)

			// Auditoria de eventos enviados pra Meta CAPI.
			r.Get("/admin/sent-events", adminSentH.List)

			// Tracked users (visitantes/leads).
			r.Get("/admin/tracked-users", adminTUH.List)

			// Stats agregadas pra dashboard.
			r.Get("/admin/stats", adminStatsH.Overview)

			// Counters agregados (PageView, ViewContent, scroll, etc).
			r.Get("/admin/counters/summary", adminCountersH.Summary)
			r.Get("/admin/counters/series", adminCountersH.TimeSeries)

			// API keys (server-to-server, scoped por conta)
			r.Get("/admin/api-keys", adminAPIKeysH.List)
			r.Post("/admin/api-keys", adminAPIKeysH.Create)
			r.Delete("/admin/api-keys/{id}", adminAPIKeysH.Revoke)
		})
	})

	return &Server{cfg: deps.Cfg, router: r}, nil
}

func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.cfg.HTTPAddr,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	return srv.ListenAndServe()
}

// buildCORS monta o middleware CORS. Em modo strict, usa AllowOriginFunc pra
// validar dinamicamente contra os domínios cadastrados em pixels.domains.
// Em modo permissivo, aceita qualquer origem (útil em dev).
//
// Independente do modo, o APP_URL (painel admin) é SEMPRE permitido.
func buildCORS(cfg *config.Config, set *util.DomainSet) func(http.Handler) http.Handler {
	// Origin do painel admin (ex: "https://app.trackfind.com.br") -> SEMPRE permitido.
	adminOrigin := normalizeOrigin(cfg.AppURL)

	if !cfg.StrictCORS {
		return cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Authorization", "Content-Type"},
			AllowCredentials: false,
			MaxAge:           300,
		})
	}
	return cors.Handler(cors.Options{
		AllowOriginFunc: func(_ *http.Request, origin string) bool {
			// Painel admin sempre passa, mesmo sem estar cadastrado em pixels.domains.
			if adminOrigin != "" && strings.EqualFold(origin, adminOrigin) {
				return true
			}
			if set == nil {
				return false
			}
			return set.AllowOrigin(origin)
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}

// normalizeOrigin extrai apenas scheme + host (+ port) de uma URL completa.
// Ex: "https://app.trackfind.com.br/login" -> "https://app.trackfind.com.br"
// Retorna "" se a URL for inválida ou vazia.
func normalizeOrigin(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
