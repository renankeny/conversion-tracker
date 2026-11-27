package campaignrules

import (
	"fmt"
	"math"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func CostPerQualified(row domain.CampaignRow) float64 {
	if row.AIQualified <= 0 {
		return 0
	}
	return round2(row.Spend / float64(row.AIQualified))
}

func BuildSnapshot(row domain.CampaignRow) domain.CampaignRuleMetricsSnapshot {
	return domain.CampaignRuleMetricsSnapshot{
		Spend:            round2(row.Spend),
		Impressions:      row.Impressions,
		Clicks:           row.Clicks,
		CTR:              round2(row.CTR),
		CPC:              round2(row.CPC),
		Leads:            row.Leads,
		Conversations:    row.Conversations,
		AIQualified:      row.AIQualified,
		GrossSales:       row.GrossSales,
		RefundCount:      row.RefundCount,
		NetSales:         row.NetSales,
		NetRevenue:       round2(row.NetRevenue),
		NetProfit:        round2(row.NetProfit),
		RefundRate:       round2(row.RefundRate),
		RealCPA:          round2(row.RealCPA),
		CostPerLead:      round2(row.CostPerLead),
		CostPerQualified: CostPerQualified(row),
		NetROAS:          round2(row.NetROAS),
		ROI:              round2(row.ROI),
		DailyBudget:      round2(row.DailyBudget),
	}
}

// ExtractMetric lê uma métrica normalizada do CampaignRow.
// hasDenominator protege métricas de custo unitário contra falsos positivos em `<`/`<=` quando denominador == 0.
func ExtractMetric(row domain.CampaignRow, metric string) (val float64, ok bool, hasDenominator bool) {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "spend":
		return row.Spend, true, true
	case "impressions":
		return float64(row.Impressions), true, true
	case "clicks":
		return float64(row.Clicks), true, true
	case "ctr":
		return row.CTR, true, row.Impressions > 0
	case "cpc":
		return row.CPC, true, row.Clicks > 0
	case "cpm":
		return row.CPM, true, row.Impressions > 0
	case "leads":
		return float64(row.Leads), true, true
	case "conversations":
		return float64(row.Conversations), true, true
	case "ai_qualified":
		return float64(row.AIQualified), true, true
	case "gross_sales":
		return float64(row.GrossSales), true, true
	case "refund_count":
		return float64(row.RefundCount), true, true
	case "chargeback_count":
		return float64(row.ChargebackCount), true, true
	case "net_sales":
		return float64(row.NetSales), true, true
	case "gross_revenue":
		return row.GrossRevenue, true, true
	case "refund_amount":
		return row.RefundAmount, true, true
	case "net_revenue":
		return row.NetRevenue, true, true
	case "net_profit":
		return row.NetProfit, true, true
	case "refund_rate":
		return row.RefundRate, true, row.GrossSales > 0
	case "real_cpa":
		return row.RealCPA, true, row.NetSales > 0
	case "cost_per_lead":
		return row.CostPerLead, true, row.Leads > 0
	case "cost_per_qualified":
		return CostPerQualified(row), true, row.AIQualified > 0
	case "gross_roas":
		return row.GrossROAS, true, row.Spend > 0
	case "net_roas":
		return row.NetROAS, true, row.Spend > 0
	case "roi":
		return row.ROI, true, row.Spend > 0
	case "daily_budget":
		return row.DailyBudget, true, true
	default:
		return 0, false, false
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func metricLabelPT(metric string) string {
	labels := map[string]string{
		"spend": "Gasto", "impressions": "Impressões", "clicks": "Cliques",
		"ctr": "CTR", "cpc": "CPC", "cpm": "CPM",
		"leads": "Leads", "conversations": "Conversas", "ai_qualified": "Qualificados IA",
		"gross_sales": "Vendas brutas", "refund_count": "Reembolsos", "net_sales": "Vendas líquidas",
		"net_revenue": "Receita líquida", "net_profit": "Lucro líquido", "refund_rate": "Taxa reembolso",
		"real_cpa": "CPA real", "cost_per_lead": "Custo/Lead", "cost_per_qualified": "Custo/Qualif. IA",
		"gross_roas": "ROAS bruto", "net_roas": "ROAS líquido", "roi": "ROI", "daily_budget": "Orçamento diário",
	}
	if l, ok := labels[metric]; ok {
		return l
	}
	return metric
}

func formatMetricVal(metric string, v float64) string {
	switch metric {
	case "spend", "cpc", "cpm", "gross_revenue", "refund_amount", "net_revenue", "net_profit",
		"real_cpa", "cost_per_lead", "cost_per_qualified", "daily_budget":
		return fmt.Sprintf("R$ %.2f", v)
	case "ctr", "refund_rate", "roi":
		return fmt.Sprintf("%.1f%%", v)
	case "gross_roas", "net_roas":
		return fmt.Sprintf("%.2fx", v)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}
