package domain

import "testing"

func TestSyncPixelSalesGoalMonths(t *testing.T) {
	curYM := CurrentSaoPauloYearMonth()
	p := &Pixel{SalesGoalEnabled: true, SalesGoalTarget: 45000}
	SyncPixelSalesGoalMonths(p, nil)

	if p.SalesGoalMonths[curYM] != 45000 {
		t.Fatalf("expected current month %s initialized to 45000, got %+v", curYM, p.SalesGoalMonths)
	}

	SyncPixelSalesGoalMonths(p, map[string]float64{
		"2026-10":  30000,
		curYM:      60000,
		"invalido": 999,
		"2026-09":  -10,
	})
	if len(p.SalesGoalMonths) != 2 {
		t.Fatalf("expected 2 valid months, got %+v", p.SalesGoalMonths)
	}
	if p.SalesGoalMonths["2026-10"] != 30000 || p.SalesGoalMonths[curYM] != 60000 {
		t.Fatalf("unexpected months map: %+v", p.SalesGoalMonths)
	}
	if p.SalesGoalTarget != 60000 {
		t.Fatalf("expected SalesGoalTarget synced to current month 60000, got %v", p.SalesGoalTarget)
	}
}
