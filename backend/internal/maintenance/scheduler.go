package maintenance

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PerAccountRetentionLister interface {
	ListRetentions(ctx context.Context) ([]PerAccountRetention, error)
}

type PerAccountRetention struct {
	AccountID                  int64
	SentEventsRetentionDays    *int
	EventCountersRetentionDays *int
}

type Scheduler struct {
	Pool                   *pgxpool.Pool
	Accounts               PerAccountRetentionLister
	MediaDeleter           ObjectBatchDeleter
	SentEventsRetention    int
	EventCountersRetention int
	WhatsAppMediaRetention int
	HourSP                 int
	MinuteSP               int
}

func (s *Scheduler) Run(ctx context.Context) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.UTC
	}

	for {
		next := nextRunAt(time.Now().In(loc), s.HourSP, s.MinuteSP)
		wait := time.Until(next)
		slog.Info("maintenance scheduled",
			"next_run", next.Format(time.RFC3339),
			"wait_seconds", int(wait.Seconds()),
			"sent_events_retention_days", s.SentEventsRetention,
			"event_counters_retention_days", s.EventCountersRetention,
		)

		select {
		case <-ctx.Done():
			slog.Info("maintenance scheduler stopped")
			return
		case <-time.After(wait):
		}

		s.runOnce(ctx)
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	if s.Accounts == nil {
		s.runOnceGlobal(ctx)
		return
	}

	retentions, err := s.Accounts.ListRetentions(ctx)
	if err != nil {
		slog.Error("maintenance list retentions failed", "err", err)
		s.runOnceGlobal(ctx)
		return
	}

	for _, r := range retentions {
		sentDays := s.SentEventsRetention
		if r.SentEventsRetentionDays != nil {
			sentDays = *r.SentEventsRetentionDays
		}
		countersDays := s.EventCountersRetention
		if r.EventCountersRetentionDays != nil {
			countersDays = *r.EventCountersRetentionDays
		}

		if sentDays > 0 {
			if pr, err := PruneOldSentEventsForAccount(ctx, s.Pool, r.AccountID, sentDays); err != nil {
				slog.Error("prune sent_events failed", "account_id", r.AccountID, "err", err)
			} else if pr.RetentionDays > 0 {
				slog.Info("prune sent_events ok", "account_id", r.AccountID, "deleted", pr.Deleted, "days", sentDays, "duration_ms", pr.Duration.Milliseconds())
			}
		}
		if countersDays > 0 {
			if pr, err := PruneOldEventCountersForAccount(ctx, s.Pool, r.AccountID, countersDays); err != nil {
				slog.Error("prune event_counters failed", "account_id", r.AccountID, "err", err)
			} else if pr.RetentionDays > 0 {
				slog.Info("prune event_counters ok", "account_id", r.AccountID, "deleted", pr.Deleted, "days", countersDays, "duration_ms", pr.Duration.Milliseconds())
			}
		}
	}

	s.runWhatsAppPrune(ctx)
}

func (s *Scheduler) runOnceGlobal(ctx context.Context) {
	if r, err := PruneOldSentEvents(ctx, s.Pool, s.SentEventsRetention); err != nil {
		slog.Error("prune sent_events failed", "err", err)
	} else if r.RetentionDays > 0 {
		slog.Info("prune sent_events ok", "deleted", r.Deleted, "threshold", r.Threshold.Format(time.RFC3339), "duration_ms", r.Duration.Milliseconds())
	}
	if r, err := PruneOldEventCounters(ctx, s.Pool, s.EventCountersRetention); err != nil {
		slog.Error("prune event_counters failed", "err", err)
	} else if r.RetentionDays > 0 {
		slog.Info("prune event_counters ok", "deleted", r.Deleted, "threshold", r.Threshold.Format(time.RFC3339), "duration_ms", r.Duration.Milliseconds())
	}

	s.runWhatsAppPrune(ctx)
}

func (s *Scheduler) runWhatsAppPrune(ctx context.Context) {
	mediaDays := s.WhatsAppMediaRetention
	if mediaDays <= 0 {
		mediaDays = 14
	}
	if r, err := ExpireOldWhatsAppMedia(ctx, s.Pool, s.MediaDeleter, mediaDays); err != nil {
		slog.Error("expire whatsapp media failed", "err", err)
	} else if r.Deleted > 0 {
		slog.Info("expire whatsapp media ok", "expired", r.Deleted, "days", mediaDays, "duration_ms", r.Duration.Milliseconds())
	}
	webhookDays := s.SentEventsRetention
	if webhookDays <= 0 {
		webhookDays = 30
	}
	if r, err := PruneOldWebhookLogs(ctx, s.Pool, webhookDays); err != nil {
		slog.Error("prune webhook_logs failed", "err", err)
	} else if r.Deleted > 0 {
		slog.Info("prune webhook_logs ok", "deleted", r.Deleted, "days", webhookDays, "duration_ms", r.Duration.Milliseconds())
	}
}

func nextRunAt(now time.Time, hour, minute int) time.Time {
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !target.After(now) {
		target = target.AddDate(0, 0, 1)
	}
	return target
}

