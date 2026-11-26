package campaigns

import (
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func TestAttributionRollupAndNetProfit(t *testing.T) {
	camps := []domain.CampaignRow{
		{ID: "120214982310450123", Platform: "meta", Level: "campaign", CampaignID: "120214982310450123", Name: "Escala CTWA", Spend: 200},
		{ID: "21849201928", Platform: "google", Level: "campaign", CampaignID: "21849201928", Name: "Search Fundo Funil", Spend: 100},
	}
	adsets := []domain.CampaignRow{
		{ID: "120214982310450999", Platform: "meta", Level: "adset", CampaignID: "120214982310450123", AdSetID: "120214982310450999", Name: "Público Aberto", Spend: 200},
	}
	ads := []domain.CampaignRow{
		{ID: "120214982310459999", Platform: "meta", Level: "ad", CampaignID: "120214982310450123", AdSetID: "120214982310450999", Name: "Video 01 Hook", Spend: 200},
	}

	events := []repo.AttributionEventRow{
		// Venda via CTWA só com AdID -> deve propagar para AdSet e Campanha Meta
		{Kind: "sale", Value: 500, AdID: "120214982310459999"},
		// Venda via UTM com ID embutido no nome
		{Kind: "sale", Value: 500, LastUTMCampaign: "Escala CTWA|120214982310450123"},
		// Reembolso de uma venda CTWA -> deve abater receita líquida e gerar 50% taxa de reembolso
		{Kind: "refund", Value: 500, AdID: "120214982310459999"},
		// Conversa qualificada pela IA
		{Kind: "conversation", AIQualified: true, AdID: "120214982310459999"},
		// Venda via Google Ads por nome normalizado de UTM
		{Kind: "sale", Value: 400, LastUTMCampaign: "search fundo funil"},
		// Venda orgânica sem UTM -> vai para linha unattributed
		{Kind: "sale", Value: 150},
	}

	unattr, hasUnattr := attributeEvents(camps, adsets, ads, events, "last_click")
	if !hasUnattr || unattr.GrossSales != 1 || unattr.GrossRevenue != 150 {
		t.Fatalf("unexpected unattributed bucket: has=%v row=%+v", hasUnattr, unattr)
	}

	fin := domain.PixelFinancialConfig{
		GatewayFeePct:   5.0,
		GatewayFixedFee: 2.0,
		TaxPct:          10.0,
		ProductCost:     50.0,
	}
	for i := range camps {
		computeRowFinancials(&camps[i], fin)
	}
	for i := range ads {
		computeRowFinancials(&ads[i], fin)
	}

	metaCamp := camps[0]
	if metaCamp.GrossSales != 2 || metaCamp.RefundCount != 1 || metaCamp.NetSales != 1 {
		t.Fatalf("unexpected meta sales/refunds: %+v", metaCamp)
	}
	if metaCamp.NetRevenue != 500 || metaCamp.RefundRate != 50 {
		t.Fatalf("expected NetRevenue=500 RefundRate=50, got %.2f / %.2f", metaCamp.NetRevenue, metaCamp.RefundRate)
	}
	if metaCamp.Conversations != 1 || metaCamp.AIQualified != 1 {
		t.Fatalf("expected 1 conversation + 1 AI qualified rolled up from ad, got %d/%d", metaCamp.Conversations, metaCamp.AIQualified)
	}
	// OpCosts = 500*5% (25) + 1*2 (2) + 500*10% (50) + 1*50 (50) = 127.00
	// NetProfit = 500 - 127 - 200 = 173.00
	if metaCamp.OperationalCosts != 127 || metaCamp.NetProfit != 173 {
		t.Fatalf("expected OpCosts=127 NetProfit=173, got %.2f / %.2f", metaCamp.OperationalCosts, metaCamp.NetProfit)
	}
	if camps[1].GrossSales != 1 || camps[1].NetRevenue != 400 {
		t.Fatalf("expected Google campaign matched by UTM name, got %+v", camps[1])
	}
}
