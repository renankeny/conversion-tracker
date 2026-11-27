package campaignrules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/campaigns"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type Runner struct {
	Campaigns *campaigns.Service
	Rules     *repo.CampaignRulesRepo
	Logs      *repo.CampaignRuleLogsRepo
	Notify    *notify.Service
	WhatsApp  *repo.WhatsAppRepo
}

func NewRunner(
	campSvc *campaigns.Service,
	rulesRepo *repo.CampaignRulesRepo,
	logsRepo *repo.CampaignRuleLogsRepo,
	notifySvc *notify.Service,
	waRepo *repo.WhatsAppRepo,
) *Runner {
	return &Runner{Campaigns: campSvc, Rules: rulesRepo, Logs: logsRepo, Notify: notifySvc, WhatsApp: waRepo}
}

func (r *Runner) EvaluateRule(ctx context.Context, rule domain.CampaignRule, forceManual bool, overrideDryRun bool) (*domain.CampaignRuleRunSummary, error) {
	effectiveDryRun := rule.DryRun || overrideDryRun || rule.ActionType == domain.RuleActionNotifyOnly
	nowSP := time.Now().In(SaoPauloLocation())
	since, until, from, to := ResolvePeriodDates(rule.PeriodPreset, nowSP)
	startOfDaySP := time.Date(nowSP.Year(), nowSP.Month(), nowSP.Day(), 0, 0, 0, 0, nowSP.Location())

	summary := &domain.CampaignRuleRunSummary{
		RuleID: rule.ID, RuleName: rule.Name, DryRun: effectiveDryRun, Logs: []domain.CampaignRuleLog{},
	}

	overview, err := r.Campaigns.Overview(ctx, campaigns.OverviewParams{
		AccountID: rule.AccountID, PixelName: rule.PixelName, Platform: rule.Platform,
		AttributionMode: rule.AttributionMode, Since: since, Until: until, From: from, To: to, ForceRefresh: forceManual,
	})
	if err != nil {
		_ = r.Rules.UpdateRunTelemetry(ctx, rule.ID, 0, err.Error())
		return nil, err
	}
	summary.Warnings = overview.Warnings

	triggeredCount := 0
	var lastRowErr string

	for _, row := range selectLevelRows(overview, rule.Level) {
		if !MatchesRowFilters(rule, row) {
			continue
		}
		summary.EvaluatedCount++

		if rule.RuleKind == domain.RuleKindReactivateLateSale {
			if wasPaused, perr := r.Logs.WasPausedByRuleToday(ctx, rule.AccountID, row.Platform, row.ID, startOfDaySP); perr != nil || !wasPaused {
				continue
			}
		}

		matched, reason := EvaluateConditions(rule, row)
		if !matched {
			continue
		}
		if rule.RuleKind == domain.RuleKindSchedule && len(rule.Conditions) == 0 {
			reason = fmt.Sprintf("Agendamento programado (%s)", rule.ScheduleTime)
		}
		summary.MatchedCount++

		if !forceManual && rule.CooldownMinutes > 0 {
			cdSince := nowSP.Add(-time.Duration(rule.CooldownMinutes) * time.Minute)
			if recent, cerr := r.Logs.HasRecentExecution(ctx, rule.ID, row.ID, cdSince); cerr == nil && recent {
				summary.SkippedCount++
				continue
			}
		}

		logEntry, skipped := r.applyOrSimulate(ctx, rule, row, overview.Currency, reason, effectiveDryRun)
		if skipped {
			summary.SkippedCount++
			continue
		}
		_ = r.Logs.InsertLog(ctx, &logEntry)
		summary.Logs = append(summary.Logs, logEntry)

		if logEntry.Status == domain.RuleLogStatusFailed {
			summary.FailedCount++
			lastRowErr = logEntry.ErrorMessage
			continue
		}
		summary.ExecutedCount++
		triggeredCount++
		dispatchRuleAlerts(r.Notify, r.WhatsApp, rule, logEntry)
	}

	_ = r.Rules.UpdateRunTelemetry(ctx, rule.ID, triggeredCount, lastRowErr)
	return summary, nil
}

func (r *Runner) applyOrSimulate(ctx context.Context, rule domain.CampaignRule, row domain.CampaignRow, fallbackCurrency, reason string, dryRun bool) (domain.CampaignRuleLog, bool) {
	ruleID := rule.ID
	currency := row.Currency
	if strings.TrimSpace(currency) == "" {
		currency = fallbackCurrency
	}
	parentID := row.AdSetID
	if row.Level == "adset" {
		parentID = row.CampaignID
	}

	entry := domain.CampaignRuleLog{
		AccountID: rule.AccountID, RuleID: &ruleID, RuleName: rule.Name,
		Platform: row.Platform, Level: row.Level, EntitySubtype: row.EntitySubtype,
		ObjectID: row.ID, ParentID: parentID, ObjectName: row.Name, PixelName: rule.PixelName,
		AccountRefID: row.AccountRefID, AdAccountID: row.AdAccountID, LoginCustomerID: row.LoginCustomerID,
		BudgetID: row.BudgetID, Currency: currency, ActionType: rule.ActionType, DryRun: dryRun,
		PreviousStatus: row.Status, PreviousBudget: round2(row.DailyBudget),
		MetricsSnapshot: BuildSnapshot(row), Reason: reason,
	}

	switch rule.ActionType {
	case domain.RuleActionPause, domain.RuleActionActivate:
		if !row.CanEditStatus {
			return entry, true
		}
		target := "PAUSED"
		if rule.ActionType == domain.RuleActionActivate {
			target = "ACTIVE"
		}
		if strings.EqualFold(row.Status, target) {
			return entry, true
		}
		entry.NewStatus = target
		if dryRun {
			entry.Status = domain.RuleLogStatusSimulated
			return entry, false
		}
		err := r.Campaigns.MutateStatus(ctx, rule.AccountID, domain.CampaignStatusMutateInput{
			Platform: row.Platform, Level: row.Level, EntitySubtype: row.EntitySubtype,
			ObjectID: row.ID, ParentID: parentID, Status: target, PixelName: rule.PixelName,
			AccountRefID: row.AccountRefID, AdAccountID: row.AdAccountID, LoginCustomerID: row.LoginCustomerID,
		})
		setMutationOutcome(&entry, err)
		return entry, false

	case domain.RuleActionIncreaseBudgetPct, domain.RuleActionIncreaseBudgetFixed,
		domain.RuleActionDecreaseBudgetPct, domain.RuleActionDecreaseBudgetFixed, domain.RuleActionSetBudget:
		if !row.CanEditBudget {
			return entry, true
		}
		nextBudget, changed := ComputeTargetBudget(rule, row.DailyBudget)
		if !changed {
			return entry, true
		}
		entry.NewBudget = nextBudget
		if dryRun {
			entry.Status = domain.RuleLogStatusSimulated
			return entry, false
		}
		err := r.Campaigns.MutateBudget(ctx, rule.AccountID, domain.CampaignBudgetMutateInput{
			Platform: row.Platform, Level: row.Level, ObjectID: row.ID, BudgetID: row.BudgetID,
			DailyBudget: nextBudget, Currency: currency, PixelName: rule.PixelName,
			AccountRefID: row.AccountRefID, AdAccountID: row.AdAccountID, LoginCustomerID: row.LoginCustomerID,
		})
		setMutationOutcome(&entry, err)
		return entry, false

	default:
		entry.DryRun = true
		entry.Status = domain.RuleLogStatusSimulated
		return entry, false
	}
}

func setMutationOutcome(entry *domain.CampaignRuleLog, err error) {
	if err != nil {
		entry.Status = domain.RuleLogStatusFailed
		entry.ErrorMessage = err.Error()
		return
	}
	entry.Status = domain.RuleLogStatusExecuted
}

func selectLevelRows(ov *domain.CampaignsOverviewResponse, level string) []domain.CampaignRow {
	switch level {
	case "campaign":
		return ov.Campaigns
	case "ad":
		return ov.Ads
	default:
		return ov.AdSets
	}
}
