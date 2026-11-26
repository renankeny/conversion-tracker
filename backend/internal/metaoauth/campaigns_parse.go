package metaoauth

import (
	"sort"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func applyMetaInsightRow(r *domain.CampaignRow, ins map[string]any) {
	r.Spend += metaFloat(ins["spend"])
	r.Impressions += metaInt64(ins["impressions"])
	r.Clicks += metaInt64(ins["clicks"])
}

func finalizeMetaTrafficRates(rows []domain.CampaignRow) {
	for i := range rows {
		if rows[i].Impressions > 0 {
			rows[i].CTR = (float64(rows[i].Clicks) / float64(rows[i].Impressions)) * 100.0
			rows[i].CPM = (rows[i].Spend / float64(rows[i].Impressions)) * 1000.0
		}
		if rows[i].Clicks > 0 {
			rows[i].CPC = rows[i].Spend / float64(rows[i].Clicks)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Spend != rows[j].Spend {
			return rows[i].Spend > rows[j].Spend
		}
		if (rows[i].Status == "ACTIVE") != (rows[j].Status == "ACTIVE") {
			return rows[i].Status == "ACTIVE"
		}
		return rows[i].Name < rows[j].Name
	})
}

func buildMetaHierarchyBundle(
	accRefID int64,
	act, currency string,
	campRaw, adsetRaw, adRaw, campInsRaw, adsetInsRaw, adInsRaw []map[string]any,
) MetaHierarchyBundle {
	div := metaBudgetDivisor(currency)
	campMap := make(map[string]*domain.CampaignRow, len(campRaw))
	for _, m := range campRaw {
		id := metaStr(m, "id")
		if id == "" {
			continue
		}
		daily := metaFloat(m["daily_budget"]) / div
		life := metaFloat(m["lifetime_budget"]) / div
		st := normalizeMetaStatus(metaStr(m, "status"))
		campMap[id] = &domain.CampaignRow{
			ID: id, Platform: "meta", Level: "campaign", EntitySubtype: "campaign",
			AccountRefID: accRefID, AdAccountID: act, Currency: currency,
			CampaignID: id, CampaignName: metaStr(m, "name"), Name: metaStr(m, "name"),
			Status: st, EffectiveStatus: metaStr(m, "effective_status"), Objective: metaStr(m, "objective"),
			DailyBudget: daily, LifetimeBudget: life,
			BudgetLevel:   budgetLevelIfSet(daily, life, "campaign"),
			CanEditStatus: st != "ARCHIVED", CanEditBudget: daily > 0 && st != "ARCHIVED",
		}
	}
	for _, ins := range campInsRaw {
		id := metaStr(ins, "campaign_id")
		if id == "" {
			continue
		}
		r, ok := campMap[id]
		if !ok {
			name := metaStr(ins, "campaign_name")
			r = &domain.CampaignRow{
				ID: id, Platform: "meta", Level: "campaign", EntitySubtype: "campaign",
				AccountRefID: accRefID, AdAccountID: act, Currency: currency,
				CampaignID: id, CampaignName: name, Name: name, Status: "ARCHIVED",
			}
			campMap[id] = r
		}
		applyMetaInsightRow(r, ins)
	}

	adsetMap := make(map[string]*domain.CampaignRow, len(adsetRaw))
	for _, m := range adsetRaw {
		id := metaStr(m, "id")
		if id == "" {
			continue
		}
		cid := metaStr(m, "campaign_id")
		cname := ""
		if c, ok := campMap[cid]; ok {
			cname = c.Name
		}
		daily := metaFloat(m["daily_budget"]) / div
		life := metaFloat(m["lifetime_budget"]) / div
		st := normalizeMetaStatus(metaStr(m, "status"))
		adsetMap[id] = &domain.CampaignRow{
			ID: id, Platform: "meta", Level: "adset", EntitySubtype: "adset",
			AccountRefID: accRefID, AdAccountID: act, Currency: currency,
			CampaignID: cid, CampaignName: cname, AdSetID: id, AdSetName: metaStr(m, "name"), Name: metaStr(m, "name"),
			Status: st, EffectiveStatus: metaStr(m, "effective_status"),
			DailyBudget: daily, LifetimeBudget: life,
			BudgetLevel:   budgetLevelIfSet(daily, life, "adset"),
			CanEditStatus: st != "ARCHIVED", CanEditBudget: daily > 0 && st != "ARCHIVED",
		}
	}
	for _, ins := range adsetInsRaw {
		id := metaStr(ins, "adset_id")
		if id == "" {
			continue
		}
		r, ok := adsetMap[id]
		if !ok {
			name := metaStr(ins, "adset_name")
			r = &domain.CampaignRow{
				ID: id, Platform: "meta", Level: "adset", EntitySubtype: "adset",
				AccountRefID: accRefID, AdAccountID: act, Currency: currency,
				CampaignID: metaStr(ins, "campaign_id"), CampaignName: metaStr(ins, "campaign_name"),
				AdSetID: id, AdSetName: name, Name: name, Status: "ARCHIVED",
			}
			adsetMap[id] = r
		}
		applyMetaInsightRow(r, ins)
	}

	adMap := make(map[string]*domain.CampaignRow, len(adRaw))
	for _, m := range adRaw {
		id := metaStr(m, "id")
		if id == "" {
			continue
		}
		cid, asid := metaStr(m, "campaign_id"), metaStr(m, "adset_id")
		cname, asname := "", ""
		if c, ok := campMap[cid]; ok {
			cname = c.Name
		}
		if as, ok := adsetMap[asid]; ok {
			asname = as.Name
		}
		thumb := ""
		if cr, ok := m["creative"].(map[string]any); ok {
			thumb = metaStr(cr, "thumbnail_url")
			if thumb == "" {
				thumb = metaStr(cr, "image_url")
			}
		}
		st := normalizeMetaStatus(metaStr(m, "status"))
		adMap[id] = &domain.CampaignRow{
			ID: id, Platform: "meta", Level: "ad", EntitySubtype: "ad",
			AccountRefID: accRefID, AdAccountID: act, Currency: currency,
			CampaignID: cid, CampaignName: cname, AdSetID: asid, AdSetName: asname,
			Name: metaStr(m, "name"), ThumbnailURL: thumb,
			Status: st, EffectiveStatus: metaStr(m, "effective_status"),
			CanEditStatus: st != "ARCHIVED",
		}
	}
	for _, ins := range adInsRaw {
		id := metaStr(ins, "ad_id")
		if id == "" {
			continue
		}
		r, ok := adMap[id]
		if !ok {
			r = &domain.CampaignRow{
				ID: id, Platform: "meta", Level: "ad", EntitySubtype: "ad",
				AccountRefID: accRefID, AdAccountID: act, Currency: currency,
				CampaignID: metaStr(ins, "campaign_id"), CampaignName: metaStr(ins, "campaign_name"),
				AdSetID: metaStr(ins, "adset_id"), AdSetName: metaStr(ins, "adset_name"),
				Name: metaStr(ins, "ad_name"), Status: "ARCHIVED",
			}
			adMap[id] = r
		}
		applyMetaInsightRow(r, ins)
	}

	bundle := MetaHierarchyBundle{
		Currency:  currency,
		Campaigns: mapToCampaignSlice(campMap),
		AdSets:    mapToCampaignSlice(adsetMap),
		Ads:       mapToCampaignSlice(adMap),
	}
	finalizeMetaTrafficRates(bundle.Campaigns)
	finalizeMetaTrafficRates(bundle.AdSets)
	finalizeMetaTrafficRates(bundle.Ads)
	return bundle
}

func budgetLevelIfSet(daily, life float64, lvl string) string {
	if daily > 0 || life > 0 {
		return lvl
	}
	return ""
}

func mapToCampaignSlice(m map[string]*domain.CampaignRow) []domain.CampaignRow {
	out := make([]domain.CampaignRow, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	return out
}
