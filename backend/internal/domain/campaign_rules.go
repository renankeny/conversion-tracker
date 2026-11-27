package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	RuleKindMetric             = "metric"
	RuleKindSchedule           = "schedule"
	RuleKindReactivateLateSale = "reactivate_late_sale"

	RuleActionPause               = "pause"
	RuleActionActivate            = "activate"
	RuleActionIncreaseBudgetPct   = "increase_budget_pct"
	RuleActionIncreaseBudgetFixed = "increase_budget_fixed"
	RuleActionDecreaseBudgetPct   = "decrease_budget_pct"
	RuleActionDecreaseBudgetFixed = "decrease_budget_fixed"
	RuleActionSetBudget           = "set_budget"
	RuleActionNotifyOnly          = "notify_only"

	RuleLogStatusExecuted  = "executed"
	RuleLogStatusSimulated = "simulated"
	RuleLogStatusFailed    = "failed"
	RuleLogStatusUndone    = "undone"
)

// CampaignRuleCondition representa uma condição numérica avaliada sobre uma linha de campanha.
type CampaignRuleCondition struct {
	Metric   string  `json:"metric"`   // spend, net_sales, net_roas, real_cpa, conversations, ai_qualified, etc.
	Operator string  `json:"operator"` // >, >=, <, <=, ==, !=
	Value    float64 `json:"value"`
}

// CampaignRule representa uma regra automática configurada pela empresa.
type CampaignRule struct {
	ID               int64                   `json:"id"`
	AccountID        int64                   `json:"account_id"`
	Name             string                  `json:"name"`
	Description      string                  `json:"description"`
	TemplateKey      string                  `json:"template_key,omitempty"`
	Enabled          bool                    `json:"enabled"`
	DryRun           bool                    `json:"dry_run"`
	Platform         string                  `json:"platform"`         // all | meta | google
	PixelName        string                  `json:"pixel_name"`       // opcional
	Level            string                  `json:"level"`            // campaign | adset | ad
	AttributionMode  string                  `json:"attribution_mode"` // last_click | first_click
	RuleKind         string                  `json:"rule_kind"`        // metric | schedule | reactivate_late_sale
	PeriodPreset     string                  `json:"period_preset"`    // today | yesterday | 3d | 7d | 14d | 30d
	Conditions       []CampaignRuleCondition `json:"conditions"`
	ActionType       string                  `json:"action_type"`
	ActionValue      float64                 `json:"action_value"`
	MinBudget        float64                 `json:"min_budget"`
	MaxBudget        float64                 `json:"max_budget"`
	NameContains     string                  `json:"name_contains"`
	NameExcludes     string                  `json:"name_excludes"`
	TargetStatus     string                  `json:"target_status"` // ACTIVE | PAUSED | ALL
	FrequencyMinutes int                     `json:"frequency_minutes"`
	MinHourSP        int                     `json:"min_hour_sp"`
	ScheduleTime     string                  `json:"schedule_time"` // HH:MM (America/Sao_Paulo)
	ScheduleDays     []int                   `json:"schedule_days"` // 0=Dom..6=Sáb (vazio = todos os dias)
	CooldownMinutes  int                     `json:"cooldown_minutes"`
	NotifyPush       bool                    `json:"notify_push"`
	NotifyWhatsApp   bool                    `json:"notify_whatsapp"`
	WhatsAppPhone    string                  `json:"whatsapp_phone"`
	LastRunAt        *time.Time              `json:"last_run_at,omitempty"`
	LastTriggeredAt  *time.Time              `json:"last_triggered_at,omitempty"`
	LastError        string                  `json:"last_error,omitempty"`
	TriggerCount     int                     `json:"trigger_count"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}

// CampaignRuleMetricsSnapshot preserva o retrato exato das métricas no momento do disparo.
type CampaignRuleMetricsSnapshot struct {
	Spend            float64 `json:"spend"`
	Impressions      int64   `json:"impressions"`
	Clicks           int64   `json:"clicks"`
	CTR              float64 `json:"ctr"`
	CPC              float64 `json:"cpc"`
	Leads            int64   `json:"leads"`
	Conversations    int64   `json:"conversations"`
	AIQualified      int64   `json:"ai_qualified"`
	GrossSales       int64   `json:"gross_sales"`
	RefundCount      int64   `json:"refund_count"`
	NetSales         int64   `json:"net_sales"`
	NetRevenue       float64 `json:"net_revenue"`
	NetProfit        float64 `json:"net_profit"`
	RefundRate       float64 `json:"refund_rate"`
	RealCPA          float64 `json:"real_cpa"`
	CostPerLead      float64 `json:"cost_per_lead"`
	CostPerQualified float64 `json:"cost_per_qualified"`
	NetROAS          float64 `json:"net_roas"`
	ROI              float64 `json:"roi"`
	DailyBudget      float64 `json:"daily_budget"`
}

// CampaignRuleLog registra cada execução, simulação, falha ou reversão de regra.
type CampaignRuleLog struct {
	ID              int64                       `json:"id"`
	AccountID       int64                       `json:"account_id"`
	RuleID          *int64                      `json:"rule_id,omitempty"`
	RuleName        string                      `json:"rule_name"`
	Platform        string                      `json:"platform"`
	Level           string                      `json:"level"`
	EntitySubtype   string                      `json:"entity_subtype,omitempty"`
	ObjectID        string                      `json:"object_id"`
	ParentID        string                      `json:"parent_id,omitempty"`
	ObjectName      string                      `json:"object_name"`
	PixelName       string                      `json:"pixel_name,omitempty"`
	AccountRefID    int64                       `json:"account_ref_id,omitempty"`
	AdAccountID     string                      `json:"ad_account_id,omitempty"`
	LoginCustomerID string                      `json:"login_customer_id,omitempty"`
	BudgetID        string                      `json:"budget_id,omitempty"`
	Currency        string                      `json:"currency"`
	ActionType      string                      `json:"action_type"`
	DryRun          bool                        `json:"dry_run"`
	Status          string                      `json:"status"`
	PreviousStatus  string                      `json:"previous_status,omitempty"`
	NewStatus       string                      `json:"new_status,omitempty"`
	PreviousBudget  float64                     `json:"previous_budget"`
	NewBudget       float64                     `json:"new_budget"`
	MetricsSnapshot CampaignRuleMetricsSnapshot `json:"metrics_snapshot"`
	Reason          string                      `json:"reason"`
	ErrorMessage    string                      `json:"error_message,omitempty"`
	UndoneAt        *time.Time                  `json:"undone_at,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
}

// CampaignRuleRunSummary consolida o resultado de uma rodada manual ou agendada.
type CampaignRuleRunSummary struct {
	RuleID         int64             `json:"rule_id"`
	RuleName       string            `json:"rule_name"`
	DryRun         bool              `json:"dry_run"`
	EvaluatedCount int               `json:"evaluated_count"`
	MatchedCount   int               `json:"matched_count"`
	ExecutedCount  int               `json:"executed_count"`
	SkippedCount   int               `json:"skipped_count"`
	FailedCount    int               `json:"failed_count"`
	Warnings       []string          `json:"warnings,omitempty"`
	Logs           []CampaignRuleLog `json:"logs,omitempty"`
}

// NormalizeAndValidate higieniza e valida os campos da regra antes de salvar.
func (r *CampaignRule) NormalizeAndValidate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.TemplateKey = strings.TrimSpace(r.TemplateKey)
	r.PixelName = strings.TrimSpace(r.PixelName)
	r.NameContains = strings.TrimSpace(r.NameContains)
	r.NameExcludes = strings.TrimSpace(r.NameExcludes)
	r.ScheduleTime = strings.TrimSpace(r.ScheduleTime)
	r.WhatsAppPhone = strings.TrimSpace(r.WhatsAppPhone)

	if r.Name == "" {
		return fmt.Errorf("informe um nome para a regra")
	}
	switch strings.ToLower(strings.TrimSpace(r.Platform)) {
	case "meta", "google":
		r.Platform = strings.ToLower(strings.TrimSpace(r.Platform))
	default:
		r.Platform = "all"
	}
	switch strings.ToLower(strings.TrimSpace(r.Level)) {
	case "campaign", "adset", "ad":
		r.Level = strings.ToLower(strings.TrimSpace(r.Level))
	default:
		r.Level = "adset"
	}
	if strings.ToLower(strings.TrimSpace(r.AttributionMode)) == "first_click" {
		r.AttributionMode = "first_click"
	} else {
		r.AttributionMode = "last_click"
	}
	switch strings.ToLower(strings.TrimSpace(r.RuleKind)) {
	case RuleKindSchedule, RuleKindReactivateLateSale:
		r.RuleKind = strings.ToLower(strings.TrimSpace(r.RuleKind))
	default:
		r.RuleKind = RuleKindMetric
	}
	switch strings.ToLower(strings.TrimSpace(r.PeriodPreset)) {
	case "today", "yesterday", "3d", "7d", "14d", "30d":
		r.PeriodPreset = strings.ToLower(strings.TrimSpace(r.PeriodPreset))
	default:
		r.PeriodPreset = "today"
	}
	switch strings.ToUpper(strings.TrimSpace(r.TargetStatus)) {
	case "ACTIVE", "PAUSED", "ALL":
		r.TargetStatus = strings.ToUpper(strings.TrimSpace(r.TargetStatus))
	default:
		if r.RuleKind == RuleKindReactivateLateSale || r.ActionType == RuleActionActivate {
			r.TargetStatus = "PAUSED"
		} else {
			r.TargetStatus = "ACTIVE"
		}
	}
	switch r.ActionType {
	case RuleActionPause, RuleActionActivate, RuleActionNotifyOnly,
		RuleActionIncreaseBudgetPct, RuleActionIncreaseBudgetFixed,
		RuleActionDecreaseBudgetPct, RuleActionDecreaseBudgetFixed, RuleActionSetBudget:
	default:
		return fmt.Errorf("ação de regra inválida: %s", r.ActionType)
	}
	if r.IsBudgetAction() && r.Level == "ad" {
		return fmt.Errorf("alteração de orçamento só pode ser aplicada no nível de Campanha ou Conjunto/Grupo")
	}
	if r.IsBudgetAction() && r.ActionValue <= 0 {
		return fmt.Errorf("informe um valor positivo para a ação de orçamento")
	}
	if r.MinBudget < 0 || r.MaxBudget < 0 {
		return fmt.Errorf("limites de orçamento não podem ser negativos")
	}
	if r.MinBudget > 0 && r.MaxBudget > 0 && r.MinBudget > r.MaxBudget {
		return fmt.Errorf("orçamento mínimo não pode ser maior que o orçamento máximo")
	}
	if r.RuleKind == RuleKindSchedule {
		if len(r.ScheduleTime) != 5 || r.ScheduleTime[2] != ':' {
			return fmt.Errorf("informe o horário de agendamento no formato HH:MM")
		}
	} else if len(r.Conditions) == 0 {
		return fmt.Errorf("adicione ao menos uma condição para a regra")
	}
	if r.FrequencyMinutes < 15 {
		r.FrequencyMinutes = 15
	}
	if r.MinHourSP < 0 || r.MinHourSP > 23 {
		r.MinHourSP = 0
	}
	if r.CooldownMinutes < 0 {
		r.CooldownMinutes = 0
	}
	if r.Conditions == nil {
		r.Conditions = []CampaignRuleCondition{}
	}
	if r.ScheduleDays == nil {
		r.ScheduleDays = []int{}
	}
	return nil
}

func (r *CampaignRule) IsBudgetAction() bool {
	switch r.ActionType {
	case RuleActionIncreaseBudgetPct, RuleActionIncreaseBudgetFixed,
		RuleActionDecreaseBudgetPct, RuleActionDecreaseBudgetFixed, RuleActionSetBudget:
		return true
	default:
		return false
	}
}
