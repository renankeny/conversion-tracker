package campaignrules

import (
	"context"
	"errors"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// UndoLog reverte uma ação real executada por regra (status ou orçamento diário)
// diretamente na API da Meta ou Google Ads e marca o registro como desfeito.
func (r *Runner) UndoLog(ctx context.Context, accountID, logID int64) (*domain.CampaignRuleLog, error) {
	entry, err := r.Logs.GetLogByID(ctx, accountID, logID)
	if err != nil {
		return nil, err
	}
	if entry.DryRun || entry.Status != domain.RuleLogStatusExecuted {
		return nil, errors.New("apenas ações reais executadas podem ser desfeitas")
	}

	switch entry.ActionType {
	case domain.RuleActionPause, domain.RuleActionActivate:
		if entry.PreviousStatus == "" {
			return nil, errors.New("status anterior indisponível para reversão")
		}
		if err := r.Campaigns.MutateStatus(ctx, accountID, domain.CampaignStatusMutateInput{
			Platform:        entry.Platform,
			Level:           entry.Level,
			EntitySubtype:   entry.EntitySubtype,
			ObjectID:        entry.ObjectID,
			ParentID:        entry.ParentID,
			Status:          entry.PreviousStatus,
			PixelName:       entry.PixelName,
			AccountRefID:    entry.AccountRefID,
			AdAccountID:     entry.AdAccountID,
			LoginCustomerID: entry.LoginCustomerID,
		}); err != nil {
			return nil, err
		}
	case domain.RuleActionIncreaseBudgetPct, domain.RuleActionIncreaseBudgetFixed,
		domain.RuleActionDecreaseBudgetPct, domain.RuleActionDecreaseBudgetFixed,
		domain.RuleActionSetBudget:
		if entry.PreviousBudget <= 0 {
			return nil, errors.New("orçamento anterior indisponível para reversão")
		}
		if err := r.Campaigns.MutateBudget(ctx, accountID, domain.CampaignBudgetMutateInput{
			Platform:        entry.Platform,
			Level:           entry.Level,
			ObjectID:        entry.ObjectID,
			BudgetID:        entry.BudgetID,
			DailyBudget:     entry.PreviousBudget,
			Currency:        entry.Currency,
			PixelName:       entry.PixelName,
			AccountRefID:    entry.AccountRefID,
			AdAccountID:     entry.AdAccountID,
			LoginCustomerID: entry.LoginCustomerID,
		}); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("este tipo de ação não suporta reversão")
	}

	if err := r.Logs.MarkLogUndone(ctx, accountID, logID); err != nil {
		return nil, err
	}
	now := time.Now()
	entry.Status = domain.RuleLogStatusUndone
	entry.UndoneAt = &now
	return entry, nil
}
