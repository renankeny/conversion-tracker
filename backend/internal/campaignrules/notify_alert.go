package campaignrules

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func actionLabelPT(actionType string, prevBudget, newBudget float64, dryRun bool) string {
	prefix := ""
	if dryRun {
		prefix = "[Simulação] Teria "
	}
	switch actionType {
	case domain.RuleActionPause:
		if dryRun {
			return prefix + "pausado"
		}
		return "Pausado automaticamente"
	case domain.RuleActionActivate:
		if dryRun {
			return prefix + "reativado"
		}
		return "Reativado automaticamente"
	case domain.RuleActionIncreaseBudgetPct, domain.RuleActionIncreaseBudgetFixed:
		if dryRun {
			return fmt.Sprintf("%saumentado orçamento (R$ %.2f → R$ %.2f)", prefix, prevBudget, newBudget)
		}
		return fmt.Sprintf("Orçamento aumentado (R$ %.2f → R$ %.2f)", prevBudget, newBudget)
	case domain.RuleActionDecreaseBudgetPct, domain.RuleActionDecreaseBudgetFixed:
		if dryRun {
			return fmt.Sprintf("%sreduzido orçamento (R$ %.2f → R$ %.2f)", prefix, prevBudget, newBudget)
		}
		return fmt.Sprintf("Orçamento reduzido (R$ %.2f → R$ %.2f)", prevBudget, newBudget)
	case domain.RuleActionSetBudget:
		if dryRun {
			return fmt.Sprintf("%sdefinido orçamento para R$ %.2f", prefix, newBudget)
		}
		return fmt.Sprintf("Orçamento ajustado para R$ %.2f", newBudget)
	default:
		return "Alerta de regra disparado"
	}
}

func dispatchRuleAlerts(
	notifySvc *notify.Service,
	waRepo *repo.WhatsAppRepo,
	rule domain.CampaignRule,
	logEntry domain.CampaignRuleLog,
) {
	actionText := actionLabelPT(logEntry.ActionType, logEntry.PreviousBudget, logEntry.NewBudget, logEntry.DryRun)
	title := fmt.Sprintf("Regra: %s", rule.Name)
	body := fmt.Sprintf("%s · %s (%s)", actionText, logEntry.ObjectName, strings.ToUpper(logEntry.Platform))
	if logEntry.Reason != "" {
		body += " — " + logEntry.Reason
	}

	if rule.NotifyPush && notifySvc != nil {
		notifySvc.DispatchAsync(notify.Event{
			AccountID:   rule.AccountID,
			Category:    domain.NotifCampaignRule,
			Title:       title,
			Body:        body,
			ActionURL:   "/campanhas/regras",
			CooldownKey: fmt.Sprintf("crule:%d:%s", rule.ID, logEntry.ObjectID),
			CooldownTTL: 10 * time.Minute,
		})
	}

	if rule.NotifyWhatsApp && strings.TrimSpace(rule.WhatsAppPhone) != "" && waRepo != nil {
		go sendRuleWhatsAppAlert(waRepo, rule.AccountID, rule.WhatsAppPhone, title, body)
	}
}

func sendRuleWhatsAppAlert(waRepo *repo.WhatsAppRepo, accountID int64, phone, title, body string) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	srv, err := waRepo.GetGlobalServer(ctx)
	if err != nil || srv == nil || strings.TrimSpace(srv.URL) == "" {
		return
	}
	instances, err := waRepo.ListInstances(ctx, accountID)
	if err != nil || len(instances) == 0 {
		return
	}
	var chosen *domain.WhatsAppInstance
	for i := range instances {
		inst := &instances[i]
		if !inst.Enabled || inst.IsExternalUazapi() {
			continue
		}
		if strings.EqualFold(inst.ConnectionStatus, "open") || strings.EqualFold(inst.ConnectionStatus, "connected") {
			chosen = inst
			break
		}
	}
	if chosen == nil || strings.TrimSpace(chosen.InstanceToken) == "" {
		return
	}
	client := evolution.NewClient(srv.URL, srv.APIKey)
	msg := fmt.Sprintf("*%s*\n%s", title, body)
	if err := client.SendText(ctx, chosen.InstanceToken, phone, msg); err != nil {
		slog.Warn("campaign_rules: whatsapp alert failed", "account_id", accountID, "error", err)
	}
}
