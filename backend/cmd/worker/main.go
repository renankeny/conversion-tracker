package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/config"
	"github.com/renankeny/conversion-tracker/backend/internal/db"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/maintenance"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
	"github.com/renankeny/conversion-tracker/backend/internal/tiktok"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/whatsappai"
	"github.com/renankeny/conversion-tracker/backend/internal/ga4"
)

func main() {
	// Carrega .env se existir
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

	googleAdsClient := googleads.New(
		cfg.GoogleAdsDeveloperToken,
		googleads.OAuthConfig{
			ClientID:     cfg.GoogleOAuthClientID,
			ClientSecret: cfg.GoogleOAuthClientSecret,
			RedirectURI:  cfg.GoogleOAuthRedirectURI,
			Pool:         pool,
			AppURL:       cfg.AppURL,
			PublicAPIURL: cfg.PublicAPIURL,
		},
	)

	rdb := queue.BuildRedisClient(cfg)
	defer rdb.Close()

	pixelRepo := repo.NewPixelRepo(pool)
	pixelRepo.SetRedis(rdb)
	go cache.RunInvalidationSubscriber(ctx, rdb, func(evCtx context.Context, ev cache.InvalidateEvent) {
		if ev.Kind == "pixel" && pixelRepo.PixelCache() != nil {
			pixelRepo.PixelCache().InvalidateKeys(evCtx, ev.Keys...)
		}
	})

	redisOpt := queue.BuildRedisClientOpt(cfg)
	enq := queue.NewEnqueuer(redisOpt)
	defer enq.Close()

	mediaStorage, err := storage.NewMediaStorage(cfg)
	if err != nil {
		slog.Warn("minio storage init warning in worker", "err", err)
	} else if mediaStorage != nil {
		if berr := mediaStorage.EnsureBucket(ctx, cfg.WhatsAppMediaRetentionDays); berr != nil {
			slog.Warn("minio bucket check warning in worker", "err", berr)
		}
	}

	srv := queue.NewServer(redisOpt, queue.Deps{
		Pool:               pool,
		Pixels:             pixelRepo,
		Users:              repo.NewUserRepo(pool),
		GoogleAdsAcc:       repo.NewGoogleAdsAccountRepo(pool),
		MetaAcc:            repo.NewMetaAccountRepo(pool),
		WhatsApp:           repo.NewWhatsAppRepo(pool),
		CRM:                repo.NewCRMRepo(pool),
		Storage:            mediaStorage,
		Enqueuer:           enq,
		Meta:               meta.New(cfg.MetaAPIVersion, cfg.MetaPartnerAgent),
		MetaOAuth:          metaoauth.NewClient(metaoauth.Config{Pool: pool, AppURL: cfg.AppURL, PublicAPIURL: cfg.PublicAPIURL}),
		TikTok:             tiktok.New(cfg.TikTokPartnerName),
		GoogleAds:          googleAdsClient,
		GA4:                ga4.NewClient(),
		Counters:           repo.NewEventCounterRepo(pool),
		SentEvents:         repo.NewSentEventRepo(pool),
		Redis:              rdb,
		PublicAPIURL:       cfg.PublicAPIURL,
		JWTSecret:          cfg.JWTSecret,
		MediaRetentionDays: cfg.WhatsAppMediaRetentionDays,
	}, cfg.WorkerConcurrency)

	go func() {
		<-ctx.Done()
		srv.Shutdown()
	}()

	// Scheduler de manutenção: roda diariamente às 03:00 (horário São Paulo)
	// removendo registros antigos de sent_events e event_counters conforme
	// SENT_EVENTS_RETENTION_DAYS e EVENT_COUNTERS_RETENTION_DAYS.
	//
	// O scheduler iterates contas e usa o override de cada uma quando definido,
	// caindo no env global apenas quando a conta não tem override.
	accountRepo := repo.NewAccountRepo(pool)
	scheduler := &maintenance.Scheduler{
		Pool:                   pool,
		Accounts:               &accountRetentionAdapter{repo: accountRepo},
		MediaDeleter:           mediaStorage,
		SentEventsRetention:    cfg.SentEventsRetentionDays,
		EventCountersRetention: cfg.EventCountersRetentionDays,
		WhatsAppMediaRetention: cfg.WhatsAppMediaRetentionDays,
		HourSP:                 3,
		MinuteSP:               0,
	}
	go scheduler.Run(ctx)

	aiScheduler := &whatsappai.Scheduler{
		Analyzer: &whatsappai.Analyzer{
			AI:       repo.NewWhatsAppAIRepo(pool),
			WhatsApp: repo.NewWhatsAppRepo(pool),
			Pixels:   pixelRepo,
			Users:    repo.NewUserRepo(pool),
			CRM:      repo.NewCRMRepo(pool),
			MetaAcc:  repo.NewMetaAccountRepo(pool),
			Storage:  mediaStorage,
			Enqueuer: enq,
			Redis:    rdb,
		},
		Interval: 15 * time.Minute,
	}
	go aiScheduler.Run(ctx)

	slog.Info("worker starting", "redis", cfg.RedisAddr)
	if err := srv.Run(); err != nil {
		slog.Error("worker stopped", "err", err)
		os.Exit(1)
	}
}


// accountRetentionAdapter adapta repo.AccountRepo à interface
// maintenance.PerAccountRetentionLister sem criar dependência do package
// maintenance no package repo.
type accountRetentionAdapter struct {
	repo *repo.AccountRepo
}

func (a *accountRetentionAdapter) ListRetentions(ctx context.Context) ([]maintenance.PerAccountRetention, error) {
	rows, err := a.repo.ListRetentions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]maintenance.PerAccountRetention, len(rows))
	for i, r := range rows {
		out[i] = maintenance.PerAccountRetention{
			AccountID:                  r.AccountID,
			SentEventsRetentionDays:    r.SentEventsRetentionDays,
			EventCountersRetentionDays: r.EventCountersRetentionDays,
		}
	}
	return out, nil
}
