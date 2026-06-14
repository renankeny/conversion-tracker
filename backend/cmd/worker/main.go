package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/renankeny/conversion-tracker/backend/internal/config"
	"github.com/renankeny/conversion-tracker/backend/internal/db"
	"github.com/renankeny/conversion-tracker/backend/internal/maintenance"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	config.MustSetTimezone()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	srv := queue.NewServer(queue.BuildRedisClientOpt(cfg), queue.Deps{
		Pool:     pool,
		Pixels:   repo.NewPixelRepo(pool),
		Meta:     meta.New(cfg.MetaAPIVersion, cfg.MetaPartnerAgent),
		Counters: repo.NewEventCounterRepo(pool),
	})

	go func() {
		<-ctx.Done()
		srv.Shutdown()
	}()

	// Scheduler de manutenção: roda diariamente às 03:00 (horário São Paulo)
	// removendo registros antigos de sent_events e event_counters conforme
	// SENT_EVENTS_RETENTION_DAYS e EVENT_COUNTERS_RETENTION_DAYS.
	scheduler := &maintenance.Scheduler{
		Pool:                   pool,
		SentEventsRetention:    cfg.SentEventsRetentionDays,
		EventCountersRetention: cfg.EventCountersRetentionDays,
		HourSP:                 3,
		MinuteSP:               0,
	}
	go scheduler.Run(ctx)

	slog.Info("worker starting", "redis", cfg.RedisAddr)
	if err := srv.Run(); err != nil {
		slog.Error("worker stopped", "err", err)
		os.Exit(1)
	}
}
