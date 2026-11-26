package campaigns

import (
	"regexp"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

var rxPlatformNumericID = regexp.MustCompile(`\b(\d{9,20})\b`)

func normalizeKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if idx := strings.Index(s, "|"); idx > 0 {
		s = strings.TrimSpace(s[:idx])
	}
	return s
}

func extractNumericID(candidates ...string) string {
	for _, raw := range candidates {
		if m := rxPlatformNumericID.FindStringSubmatch(strings.TrimSpace(raw)); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

type hierarchyIndex struct {
	campByID   map[string]int
	campByName map[string]int
	adSetByID  map[string]int
	adSetByKey map[string]int
	adByID     map[string]int
	adByKey    map[string]int
}

func indexRows(rows []domain.CampaignRow) (map[string]int, map[string]int) {
	byID := make(map[string]int, len(rows))
	byName := make(map[string]int, len(rows))
	for i := range rows {
		if rows[i].ID != "" {
			byID[rows[i].ID] = i
		}
		if k := normalizeKey(rows[i].Name); k != "" {
			if _, exists := byName[k]; !exists {
				byName[k] = i
			}
		}
	}
	return byID, byName
}

func buildHierarchyIndex(camps, adsets, ads []domain.CampaignRow) hierarchyIndex {
	cID, cName := indexRows(camps)
	sID, sName := indexRows(adsets)
	aID, aName := indexRows(ads)
	return hierarchyIndex{
		campByID: cID, campByName: cName,
		adSetByID: sID, adSetByKey: sName,
		adByID: aID, adByKey: aName,
	}
}

func applyAttributionEvent(row *domain.CampaignRow, ev repo.AttributionEventRow, src string) {
	if row.MatchSource == "" || src == "id" {
		row.MatchSource = src
	}
	switch ev.Kind {
	case "sale":
		row.GrossSales++
		row.GrossRevenue += ev.Value
	case "refund":
		row.RefundCount++
		row.RefundAmount += ev.Value
	case "chargeback":
		row.ChargebackCount++
		row.RefundAmount += ev.Value
	case "lead":
		row.Leads++
	case "conversation":
		row.Conversations++
		if ev.AIQualified {
			row.AIQualified++
		}
	}
}

// attributeEvents cruza todos os eventos reais (vendas, reembolsos, leads e conversas WA)
// nos 3 níveis hierárquicos (Campanha, Conjunto/Grupo e Anúncio), propagando automaticamente
// matches de Anúncio/Conjunto para os pais e separando eventos sem match numa linha de conciliação.
func attributeEvents(
	camps, adsets, ads []domain.CampaignRow,
	events []repo.AttributionEventRow,
	attributionMode string,
) (unattributedCamp domain.CampaignRow, hasUnattributed bool) {
	idx := buildHierarchyIndex(camps, adsets, ads)
	unattributedCamp = domain.CampaignRow{
		ID:            "unattributed",
		Platform:      "unattributed",
		Level:         "campaign",
		EntitySubtype: "unattributed",
		CampaignID:    "unattributed",
		CampaignName:  "Orgânico / Sem Campanha Identificada",
		Name:          "Orgânico / Sem Campanha Identificada",
		Status:        "ACTIVE",
		MatchSource:   "unattributed",
	}

	for _, ev := range events {
		utmCamp, utmMed, utmCont, utmTerm := pickUTMs(ev, attributionMode)

		adIdx := -1
		adSetIdx := -1
		campIdx := -1
		matchSrc := ""

		// 1) Match exato por ID de plataforma (CTWA ou ID embutido na UTM ex: "Criativo|120214...")
		if id := extractNumericID(ev.AdID, utmCont, utmTerm); id != "" {
			if i, ok := idx.adByID[id]; ok {
				adIdx, matchSrc = i, "id"
			}
		}
		if id := extractNumericID(ev.AdSetID, utmMed, utmCont); id != "" {
			if i, ok := idx.adSetByID[id]; ok {
				adSetIdx = i
				if matchSrc == "" {
					matchSrc = "id"
				}
			}
		}
		if id := extractNumericID(ev.CampaignID, utmCamp); id != "" {
			if i, ok := idx.campByID[id]; ok {
				campIdx = i
				if matchSrc == "" {
					matchSrc = "id"
				}
			}
		}

		// 2) Propagação hierárquica bottom-up (Anúncio -> Conjunto/Grupo -> Campanha)
		if adIdx >= 0 {
			if adSetIdx < 0 && ads[adIdx].AdSetID != "" {
				if i, ok := idx.adSetByID[ads[adIdx].AdSetID]; ok {
					adSetIdx = i
				}
			}
			if campIdx < 0 && ads[adIdx].CampaignID != "" {
				if i, ok := idx.campByID[ads[adIdx].CampaignID]; ok {
					campIdx = i
				}
			}
		}
		if adSetIdx >= 0 && campIdx < 0 && adsets[adSetIdx].CampaignID != "" {
			if i, ok := idx.campByID[adsets[adSetIdx].CampaignID]; ok {
				campIdx = i
			}
		}

		// 3) Fallback determinístico por Nome Normalizado da UTM
		if campIdx < 0 {
			if k := normalizeKey(utmCamp); k != "" {
				if i, ok := idx.campByName[k]; ok {
					campIdx = i
					if matchSrc == "" {
						matchSrc = "utm"
					}
				}
			}
		}
		if adSetIdx < 0 {
			for _, cand := range []string{utmMed, utmCont, utmTerm} {
				if k := normalizeKey(cand); k != "" {
					if i, ok := idx.adSetByKey[k]; ok {
						adSetIdx = i
						if campIdx < 0 && adsets[i].CampaignID != "" {
							if ci, ok := idx.campByID[adsets[i].CampaignID]; ok {
								campIdx = ci
							}
						}
						if matchSrc == "" {
							matchSrc = "utm"
						}
						break
					}
				}
			}
		}
		if adIdx < 0 {
			for _, cand := range []string{utmCont, utmTerm} {
				if k := normalizeKey(cand); k != "" {
					if i, ok := idx.adByKey[k]; ok {
						adIdx = i
						if adSetIdx < 0 && ads[i].AdSetID != "" {
							if si, ok := idx.adSetByID[ads[i].AdSetID]; ok {
								adSetIdx = si
							}
						}
						if campIdx < 0 && ads[i].CampaignID != "" {
							if ci, ok := idx.campByID[ads[i].CampaignID]; ok {
								campIdx = ci
							}
						}
						if matchSrc == "" {
							matchSrc = "utm"
						}
						break
					}
				}
			}
		}

		if campIdx >= 0 {
			applyAttributionEvent(&camps[campIdx], ev, matchSrc)
		} else {
			applyAttributionEvent(&unattributedCamp, ev, "unattributed")
			hasUnattributed = true
		}
		if adSetIdx >= 0 {
			applyAttributionEvent(&adsets[adSetIdx], ev, matchSrc)
		}
		if adIdx >= 0 {
			applyAttributionEvent(&ads[adIdx], ev, matchSrc)
		}
	}
	return unattributedCamp, hasUnattributed
}

func pickUTMs(ev repo.AttributionEventRow, mode string) (camp, med, cont, term string) {
	if mode == "first_click" {
		return firstNonEmpty(ev.FirstUTMCampaign, ev.LastUTMCampaign),
			firstNonEmpty(ev.FirstUTMMedium, ev.LastUTMMedium),
			firstNonEmpty(ev.FirstUTMContent, ev.LastUTMContent),
			firstNonEmpty(ev.FirstUTMTerm, ev.LastUTMTerm)
	}
	return firstNonEmpty(ev.LastUTMCampaign, ev.FirstUTMCampaign),
		firstNonEmpty(ev.LastUTMMedium, ev.FirstUTMMedium),
		firstNonEmpty(ev.LastUTMContent, ev.FirstUTMContent),
		firstNonEmpty(ev.LastUTMTerm, ev.FirstUTMTerm)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
