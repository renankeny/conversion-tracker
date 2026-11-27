package campaignrules

import (
	"testing"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestEvaluateConditions_StopLossNoSales(t *testing.T) {
	rule := domain.CampaignRule{
		RuleKind: domain.RuleKindMetric,
		Conditions: []domain.CampaignRuleCondition{
			{Metric: "spend", Operator: ">", Value: 150},
			{Metric: "net_sales", Operator: "==", Value: 0},
		},
	}
	rowHit := domain.CampaignRow{Spend: 185.50, NetSales: 0}
	ok, reason := EvaluateConditions(rule, rowHit)
	if !ok {
		t.Fatalf("esperava match para gasto > 150 e 0 vendas")
	}
	if reason == "" {
		t.Fatalf("esperava motivo formatado em português")
	}

	rowMiss := domain.CampaignRow{Spend: 185.50, NetSales: 1}
	if ok, _ := EvaluateConditions(rule, rowMiss); ok {
		t.Fatalf("não deveria disparar quando já existe 1 venda líquida")
	}
}

func TestEvaluateConditions_ZeroDenominatorProtection(t *testing.T) {
	rule := domain.CampaignRule{
		RuleKind: domain.RuleKindMetric,
		Conditions: []domain.CampaignRuleCondition{
			{Metric: "real_cpa", Operator: "<=", Value: 60},
		},
	}
	// Sem vendas líquidas, real_cpa é 0 — NÃO deve satisfazer "<= 60"
	rowNoSales := domain.CampaignRow{Spend: 50, NetSales: 0, RealCPA: 0}
	if ok, _ := EvaluateConditions(rule, rowNoSales); ok {
		t.Fatalf("real_cpa <= 60 não pode dar match com 0 vendas")
	}

	rowHealthy := domain.CampaignRow{Spend: 90, NetSales: 2, RealCPA: 45}
	if ok, _ := EvaluateConditions(rule, rowHealthy); !ok {
		t.Fatalf("esperava match quando real_cpa = 45 e net_sales = 2")
	}
}

func TestEvaluateConditions_WhatsAppAIQualified(t *testing.T) {
	rule := domain.CampaignRule{
		RuleKind: domain.RuleKindMetric,
		Conditions: []domain.CampaignRuleCondition{
			{Metric: "conversations", Operator: ">=", Value: 15},
			{Metric: "ai_qualified", Operator: "==", Value: 0},
		},
	}
	rowCurious := domain.CampaignRow{Spend: 120, Conversations: 18, AIQualified: 0}
	if ok, _ := EvaluateConditions(rule, rowCurious); !ok {
		t.Fatalf("esperava match para filtro de curiosos via IA")
	}

	ruleCPQ := domain.CampaignRule{
		RuleKind: domain.RuleKindMetric,
		Conditions: []domain.CampaignRuleCondition{
			{Metric: "cost_per_qualified", Operator: ">", Value: 40},
		},
	}
	rowExpensiveQ := domain.CampaignRow{Spend: 150, AIQualified: 3}
	if ok, _ := EvaluateConditions(ruleCPQ, rowExpensiveQ); !ok {
		t.Fatalf("esperava match para custo por qualificado R$ 50 > R$ 40")
	}
}

func TestComputeTargetBudget_CeilingAndFloor(t *testing.T) {
	scaleRule := domain.CampaignRule{
		ActionType:  domain.RuleActionIncreaseBudgetPct,
		ActionValue: 20,
		MaxBudget:   200,
	}
	next, changed := ComputeTargetBudget(scaleRule, 180)
	if !changed || next != 200 {
		t.Fatalf("esperava teto R$ 200, obteve %.2f (changed=%v)", next, changed)
	}
	// Já está no teto -> no-op
	if _, changed := ComputeTargetBudget(scaleRule, 200); changed {
		t.Fatalf("não deveria alterar orçamento quando já atingiu o teto")
	}

	cutRule := domain.CampaignRule{
		ActionType:  domain.RuleActionDecreaseBudgetPct,
		ActionValue: 30,
		MinBudget:   50,
	}
	next, changed = ComputeTargetBudget(cutRule, 60)
	if !changed || next != 50 {
		t.Fatalf("esperava piso R$ 50, obteve %.2f (changed=%v)", next, changed)
	}

	resetRule := domain.CampaignRule{
		ActionType:  domain.RuleActionSetBudget,
		ActionValue: 100,
	}
	next, changed = ComputeTargetBudget(resetRule, 250)
	if !changed || next != 100 {
		t.Fatalf("esperava reset fixo R$ 100, obteve %.2f", next)
	}
}

func TestMatchesRowFilters_NameIncludeExclude(t *testing.T) {
	rule := domain.CampaignRule{
		Platform:     "meta",
		TargetStatus: "ACTIVE",
		NameContains: "[escala], winner",
		NameExcludes: "[fixo], institucional",
	}
	okRow := domain.CampaignRow{ID: "1", Platform: "meta", Status: "ACTIVE", Name: "CBO [ESCALA] Video 01"}
	if !MatchesRowFilters(rule, okRow) {
		t.Fatalf("esperava match para linha elegível")
	}
	excludedRow := domain.CampaignRow{ID: "2", Platform: "meta", Status: "ACTIVE", Name: "CBO [ESCALA] [FIXO]"}
	if MatchesRowFilters(rule, excludedRow) {
		t.Fatalf("linha com tag [FIXO] deveria ser ignorada")
	}
	wrongPlat := domain.CampaignRow{ID: "3", Platform: "google", Status: "ACTIVE", Name: "Winner Search"}
	if MatchesRowFilters(rule, wrongPlat) {
		t.Fatalf("plataforma divergente deveria ser ignorada")
	}
}

func TestIsRuleDue_ScheduleAndMinHour(t *testing.T) {
	loc := SaoPauloLocation()
	// Quarta-feira 08:05 BRT (weekday = 3)
	now := time.Date(2026, 11, 25, 8, 5, 0, 0, loc)

	daypart := domain.CampaignRule{
		Enabled:      true,
		RuleKind:     domain.RuleKindSchedule,
		ScheduleTime: "08:00",
		ScheduleDays: []int{1, 2, 3, 4, 5},
	}
	if !IsRuleDue(daypart, now) {
		t.Fatalf("regra agendada das 08:00 deveria disparar às 08:05")
	}
	alreadyRan := time.Date(2026, 11, 25, 8, 1, 0, 0, loc)
	daypart.LastRunAt = &alreadyRan
	if IsRuleDue(daypart, now) {
		t.Fatalf("regra agendada já executada hoje às 08:01 não deve repetir")
	}

	metricRule := domain.CampaignRule{
		Enabled:          true,
		RuleKind:         domain.RuleKindMetric,
		FrequencyMinutes: 30,
		MinHourSP:        11,
	}
	if IsRuleDue(metricRule, now) {
		t.Fatalf("regra com min_hour_sp=11 não deve rodar às 08:05")
	}
	noon := time.Date(2026, 11, 25, 11, 10, 0, 0, loc)
	if !IsRuleDue(metricRule, noon) {
		t.Fatalf("regra com min_hour_sp=11 deve rodar às 11:10")
	}
}
