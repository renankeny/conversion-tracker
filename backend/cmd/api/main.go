package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/renankeny/conversion-tracker/backend/internal/auth"

	"github.com/renankeny/conversion-tracker/backend/internal/config"
	"github.com/renankeny/conversion-tracker/backend/internal/db"
	"github.com/renankeny/conversion-tracker/backend/internal/geoip"
	"github.com/renankeny/conversion-tracker/backend/internal/handlers"
	"github.com/renankeny/conversion-tracker/backend/internal/httpserver"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func main() {
	// Carrega .env se existir (ignora erro se não existir, pois pode estar em produção com env vars)
	_ = godotenv.Load()
	
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	config.MustSetTimezone()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := util.SetEncryptionKey(cfg.EncryptionKey); err != nil {
		slog.Error("failed to set encryption key", "err", err)
		os.Exit(1)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, "/app/migrations"); err != nil {
		// fallback pra path local em dev
		if err2 := db.Migrate(ctx, pool, "./migrations"); err2 != nil {
			slog.Error("migrate", "err", err, "fallback_err", err2)
			os.Exit(1)
		}
	}

	if os.Getenv("AUTO_BOOTSTRAP_ADMIN") == "true" {
		if err := handlers.EnsureBootstrapAdmin(ctx, pool, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			slog.Error("bootstrap admin", "err", err)
		}
	}

	geo, err := geoip.New(cfg.GeoIPDBPath)
	if err != nil {
		slog.Warn("geoip init error", "err", err)
	}
	if !geo.Available() && cfg.GeoIPDBPath != "" {
		// Mesmo que o New não tenha conseguido abrir, tenta carregar
		// (caso o arquivo tenha sido escrito no volume entre verificações).
		if rerr := geo.Reload(cfg.GeoIPDBPath); rerr != nil {
			slog.Warn("geoip disabled", "err", rerr, "path", cfg.GeoIPDBPath)
		} else {
			slog.Info("geoip loaded", "path", cfg.GeoIPDBPath)
		}
	} else if geo.Available() {
		slog.Info("geoip loaded", "path", cfg.GeoIPDBPath)
	}
	defer geo.Close()

	enq := queue.NewEnqueuer(queue.BuildRedisClientOpt(cfg))
	defer enq.Close()

	rdb := queue.BuildRedisClient(cfg)
	defer rdb.Close()

	mediaStorage, err := storage.NewMediaStorage(cfg)
	if err != nil {
		slog.Warn("minio storage init warning", "err", err)
	} else if mediaStorage != nil {
		if berr := mediaStorage.EnsureBucket(ctx, cfg.WhatsAppMediaRetentionDays); berr != nil {
			slog.Warn("minio bucket check warning", "err", berr, "bucket", cfg.MinIOBucket)
		} else {
			slog.Info("minio storage ready", "endpoint", cfg.MinIOEndpoint, "bucket", cfg.MinIOBucket)
		}
	} else {
		slog.Info("minio storage disabled (MINIO_ENDPOINT/ACCESS_KEY/SECRET_KEY not set)")
	}

	urlRepo := repo.NewURLMappingRepo(pool)
	pixelRepo := repo.NewPixelRepo(pool)

	urls := util.NewURLMapper()
	loadURLMappings(ctx, urlRepo, urls)

	allowedDomains := util.NewDomainSet()
	loadAllowedDomains(ctx, pixelRepo, allowedDomains)

	// Atualiza ambos os caches a cada 30s do banco.
	go refreshCachesLoop(ctx, urlRepo, pixelRepo, urls, allowedDomains, 30*time.Second)

	srv, err := httpserver.New(httpserver.Deps{
		Cfg:            cfg,
		Pool:           pool,
		GeoIP:          geo,
		Enqueuer:       enq,
		JWT:            auth.NewJWT(cfg.JWTSecret, cfg.JWTExpHours),
		URLs:           urls,
		AllowedDomains: allowedDomains,
		RedisOpt:       queue.BuildRedisClientOpt(cfg),
		Secrets:        handlers.GatewaySecrets{},
		Rdb:            rdb,
		MediaStorage:   mediaStorage,
	})
	if err != nil {
		slog.Error("server init", "err", err)
		os.Exit(1)
	}

	slog.Info("api starting", "addr", cfg.HTTPAddr, "strict_cors", cfg.StrictCORS)
	if err := srv.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func loadURLMappings(ctx context.Context, r *repo.URLMappingRepo, m *util.URLMapper) {
	byAcc, err := r.ActiveByAccountMap(ctx)
	if err != nil {
		slog.Warn("load url_mappings", "err", err)
		return
	}
	m.SetByAccount(byAcc)
	slog.Info("url_mappings loaded", "accounts", len(byAcc))
}

func loadAllowedDomains(ctx context.Context, r *repo.PixelRepo, set *util.DomainSet) {
	domains, err := r.ActiveDomains(ctx)
	if err != nil {
		slog.Warn("load allowed_domains", "err", err)
		return
	}
	set.Set(domains)
	slog.Info("allowed_domains loaded", "count", len(domains))
}

func refreshCachesLoop(
	ctx context.Context,
	urlRepo *repo.URLMappingRepo,
	pixelRepo *repo.PixelRepo,
	urls *util.URLMapper,
	domains *util.DomainSet,
	every time.Duration,
) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			loadURLMappings(ctx, urlRepo, urls)
			loadAllowedDomains(ctx, pixelRepo, domains)
		}
	}
}
