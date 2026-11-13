package whatsappai

import (
	"context"
	"log/slog"
	"time"
)

const cycleLockKey = "lock:whatsapp_ai_15m_cycle"

type Scheduler struct {
	Analyzer *Analyzer
	Interval time.Duration
}

func (s *Scheduler) Run(ctx context.Context) {
	if s == nil || s.Analyzer == nil || s.Analyzer.AI == nil {
		return
	}
	interval := s.Interval
	if interval <= 0 {
		interval = 15 * time.Minute
	}

	slog.Info("whatsapp ai scheduler started", "interval", interval.String())

	initialTimer := time.NewTimer(45 * time.Second)
	defer initialTimer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-initialTimer.C:
		s.runCycle(ctx)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runCycle(ctx)
		}
	}
}

func (s *Scheduler) runCycle(ctx context.Context) {
	if s.Analyzer.Redis != nil {
		ok, err := s.Analyzer.Redis.SetNX(ctx, cycleLockKey, "1", 12*time.Minute).Result()
		if err == nil && !ok {
			return
		}
		defer s.Analyzer.Redis.Del(context.Background(), cycleLockKey)
	}

	accountIDs, err := s.Analyzer.AI.ListEnabledAccounts(ctx)
	if err != nil {
		slog.Warn("whatsapp ai list enabled accounts failed", "error", err)
		return
	}
	for _, accID := range accountIDs {
		if ctx.Err() != nil {
			return
		}
		sum, aerr := s.Analyzer.AnalyzeAccount(ctx, accID, 0, false)
		if aerr != nil {
			slog.Warn("whatsapp ai cycle error", "account_id", accID, "error", aerr)
			continue
		}
		if sum.AnalyzedChats > 0 {
			slog.Info("whatsapp ai cycle completed",
				"account_id", accID,
				"analyzed_chats", sum.AnalyzedChats,
				"qualifications", sum.Qualifications,
				"sales", sum.Sales,
			)
		}
	}
}

