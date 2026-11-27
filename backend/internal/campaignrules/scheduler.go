package campaignrules

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type Scheduler struct {
	Runner   *Runner
	Redis    *redis.Client
	Interval time.Duration
}

func (s *Scheduler) Run(ctx context.Context) {
	if s == nil || s.Runner == nil || s.Runner.Rules == nil {
		return
	}
	interval := s.Interval
	if interval < 1*time.Minute {
		interval = 5 * time.Minute
	}

	// Pequena espera inicial para deixar o boot do worker estabilizar.
	select {
	case <-ctx.Done():
		return
	case <-time.After(45 * time.Second):
	}

	s.tick(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	if util.IsShardHalted() {
		return
	}
	rules, err := s.Runner.Rules.ListAllEnabled(ctx)
	if err != nil {
		slog.Warn("campaign_rules scheduler: list enabled failed", "error", err)
		return
	}
	nowSP := time.Now().In(SaoPauloLocation())
	for _, rule := range rules {
		if ctx.Err() != nil {
			return
		}
		if !IsRuleDue(rule, nowSP) {
			continue
		}
		if s.Redis != nil {
			lockKey := fmt.Sprintf("tf:crules:lock:%d", rule.ID)
			ok, lerr := s.Redis.SetNX(ctx, lockKey, "1", 8*time.Minute).Result()
			if lerr == nil && !ok {
				continue
			}
		}

		runCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		summary, rerr := s.Runner.EvaluateRule(runCtx, rule, false, false)
		cancel()

		if rerr != nil {
			slog.Warn("campaign_rules scheduler: rule failed",
				"rule_id", rule.ID, "account_id", rule.AccountID, "error", rerr)
			continue
		}
		if summary != nil && (summary.ExecutedCount > 0 || summary.FailedCount > 0) {
			slog.Info("campaign_rules scheduler: rule executed",
				"rule_id", rule.ID, "account_id", rule.AccountID,
				"dry_run", summary.DryRun, "executed", summary.ExecutedCount, "failed", summary.FailedCount)
		}
	}
}
