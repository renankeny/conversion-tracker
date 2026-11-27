package campaignrules

import (
	"fmt"
	"math"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func MatchesRowFilters(rule domain.CampaignRule, row domain.CampaignRow) bool {
	if row.ID == "" || row.Platform == "unattributed" {
		return false
	}
	if rule.Platform != "all" && !strings.EqualFold(rule.Platform, row.Platform) {
		return false
	}
	if rule.TargetStatus != "ALL" && !strings.EqualFold(row.Status, rule.TargetStatus) {
		return false
	}
	haystack := strings.ToLower(strings.TrimSpace(row.Name + " " + row.CampaignName + " " + row.AdSetName))
	if tokens := splitFilterTokens(rule.NameContains); len(tokens) > 0 {
		matched := false
		for _, tok := range tokens {
			if strings.Contains(haystack, tok) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, tok := range splitFilterTokens(rule.NameExcludes) {
		if strings.Contains(haystack, tok) {
			return false
		}
	}
	return true
}

func splitFilterTokens(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.ToLower(strings.TrimSpace(p)); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func EvaluateConditions(rule domain.CampaignRule, row domain.CampaignRow) (bool, string) {
	if len(rule.Conditions) == 0 {
		return true, "Agendamento programado"
	}
	reasons := make([]string, 0, len(rule.Conditions))
	for _, c := range rule.Conditions {
		actual, ok, hasDenom := ExtractMetric(row, c.Metric)
		if !ok {
			return false, ""
		}
		if !hasDenom && (c.Operator == "<" || c.Operator == "<=") {
			return false, ""
		}
		if !compareNumber(actual, c.Operator, c.Value) {
			return false, ""
		}
		reasons = append(reasons, fmt.Sprintf("%s %s %s %s", metricLabelPT(c.Metric), formatMetricVal(c.Metric, actual), c.Operator, formatMetricVal(c.Metric, c.Value)))
	}
	return true, strings.Join(reasons, " · ")
}

func compareNumber(actual float64, op string, target float64) bool {
	const eps = 1e-6
	switch op {
	case ">":
		return actual > target+eps
	case ">=":
		return actual >= target-eps
	case "<":
		return actual < target-eps
	case "<=":
		return actual <= target+eps
	case "==":
		return math.Abs(actual-target) <= eps
	case "!=":
		return math.Abs(actual-target) > eps
	default:
		return false
	}
}

// ComputeTargetBudget calcula o novo orçamento diário aplicando piso/teto configurados.
func ComputeTargetBudget(rule domain.CampaignRule, current float64) (float64, bool) {
	var next float64
	switch rule.ActionType {
	case domain.RuleActionIncreaseBudgetPct:
		if current <= 0 {
			return 0, false
		}
		next = current * (1 + rule.ActionValue/100)
	case domain.RuleActionIncreaseBudgetFixed:
		if current <= 0 {
			return 0, false
		}
		next = current + rule.ActionValue
	case domain.RuleActionDecreaseBudgetPct:
		if current <= 0 {
			return 0, false
		}
		next = current * (1 - rule.ActionValue/100)
	case domain.RuleActionDecreaseBudgetFixed:
		if current <= 0 {
			return 0, false
		}
		next = current - rule.ActionValue
	case domain.RuleActionSetBudget:
		next = rule.ActionValue
	default:
		return 0, false
	}
	if rule.MaxBudget > 0 && next > rule.MaxBudget {
		next = rule.MaxBudget
	}
	if rule.MinBudget > 0 && next < rule.MinBudget {
		next = rule.MinBudget
	}
	next = round2(next)
	if next < 1.0 {
		return 0, false
	}
	if math.Abs(next-round2(current)) < 0.01 {
		return next, false
	}
	return next, true
}
