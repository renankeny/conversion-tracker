package campaigns

import (
	"math"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func computeRowFinancials(r *domain.CampaignRow, fin domain.PixelFinancialConfig) {
	totalRefunds := r.RefundCount + r.ChargebackCount
	netSales := r.GrossSales - totalRefunds
	if netSales < 0 {
		netSales = 0
	}
	netRev := r.GrossRevenue - r.RefundAmount
	if netRev < 0 {
		netRev = 0
	}

	opCosts := (netRev * (fin.GatewayFeePct / 100.0)) +
		(float64(netSales) * fin.GatewayFixedFee) +
		(netRev * (fin.TaxPct / 100.0)) +
		(float64(netSales) * fin.ProductCost)

	r.NetSales = netSales
	r.GrossRevenue = round2(r.GrossRevenue)
	r.RefundAmount = round2(r.RefundAmount)
	r.NetRevenue = round2(netRev)
	r.OperationalCosts = round2(opCosts)
	r.NetProfit = round2(netRev - opCosts - r.Spend)
	r.Spend = round2(r.Spend)
	r.CTR = round2(r.CTR)
	r.CPC = round2(r.CPC)
	r.CPM = round2(r.CPM)

	if r.GrossSales > 0 {
		r.RefundRate = round2((float64(totalRefunds) / float64(r.GrossSales)) * 100.0)
	}
	if netSales > 0 && r.Spend > 0 {
		r.RealCPA = round2(r.Spend / float64(netSales))
	}
	leadBase := r.Leads
	if r.Conversations > leadBase {
		leadBase = r.Conversations
	}
	if leadBase > 0 && r.Spend > 0 {
		r.CostPerLead = round2(r.Spend / float64(leadBase))
	}
	if r.Spend > 0 {
		r.GrossROAS = round2(r.GrossRevenue / r.Spend)
		r.NetROAS = round2(r.NetRevenue / r.Spend)
		r.ROI = round2((r.NetProfit / r.Spend) * 100.0)
	}
}

func buildKpiSummary(camps []domain.CampaignRow, fin domain.PixelFinancialConfig) domain.CampaignsKpiSummary {
	var s domain.CampaignsKpiSummary
	for _, r := range camps {
		s.Spend += r.Spend
		s.Impressions += r.Impressions
		s.Clicks += r.Clicks
		s.Leads += r.Leads
		s.Conversations += r.Conversations
		s.AIQualified += r.AIQualified
		s.GrossSales += r.GrossSales
		s.RefundCount += r.RefundCount
		s.ChargebackCount += r.ChargebackCount
		s.GrossRevenue += r.GrossRevenue
		s.RefundAmount += r.RefundAmount
	}
	totalRef := s.RefundCount + s.ChargebackCount
	s.NetSales = s.GrossSales - totalRef
	if s.NetSales < 0 {
		s.NetSales = 0
	}
	s.NetRevenue = s.GrossRevenue - s.RefundAmount
	if s.NetRevenue < 0 {
		s.NetRevenue = 0
	}
	opCosts := (s.NetRevenue * (fin.GatewayFeePct / 100.0)) +
		(float64(s.NetSales) * fin.GatewayFixedFee) +
		(s.NetRevenue * (fin.TaxPct / 100.0)) +
		(float64(s.NetSales) * fin.ProductCost)

	s.Spend = round2(s.Spend)
	s.GrossRevenue = round2(s.GrossRevenue)
	s.RefundAmount = round2(s.RefundAmount)
	s.NetRevenue = round2(s.NetRevenue)
	s.OperationalCosts = round2(opCosts)
	s.NetProfit = round2(s.NetRevenue - opCosts - s.Spend)

	if s.GrossSales > 0 {
		s.RefundRate = round2((float64(totalRef) / float64(s.GrossSales)) * 100.0)
	}
	if s.NetSales > 0 && s.Spend > 0 {
		s.RealCPA = round2(s.Spend / float64(s.NetSales))
	}
	if s.Spend > 0 {
		s.GrossROAS = round2(s.GrossRevenue / s.Spend)
		s.NetROAS = round2(s.NetRevenue / s.Spend)
		s.ROI = round2((s.NetProfit / s.Spend) * 100.0)
	}
	return s
}
