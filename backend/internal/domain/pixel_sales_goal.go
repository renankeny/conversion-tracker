package domain

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
)

var rxYearMonth = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func CurrentSaoPauloYearMonth() string {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil || loc == nil {
		loc = time.FixedZone("BRT", -3*3600)
	}
	return time.Now().In(loc).Format("2006-01")
}

func NormalizeSalesGoalMonths(in map[string]float64) map[string]float64 {
	out := make(map[string]float64)
	for k, v := range in {
		ym := strings.TrimSpace(k)
		if !rxYearMonth.MatchString(ym) || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			continue
		}
		out[ym] = math.Round(v*100) / 100
	}
	return out
}

// SyncPixelSalesGoalMonths sincroniza o histórico mensal de metas com a meta vigente do mês atual.
func SyncPixelSalesGoalMonths(p *Pixel, incoming map[string]float64) {
	if p == nil {
		return
	}
	curYM := CurrentSaoPauloYearMonth()
	if incoming != nil {
		p.SalesGoalMonths = NormalizeSalesGoalMonths(incoming)
		if curVal, ok := p.SalesGoalMonths[curYM]; ok && curVal > 0 {
			p.SalesGoalTarget = curVal
		} else if p.SalesGoalTarget <= 0 && len(p.SalesGoalMonths) > 0 {
			keys := make([]string, 0, len(p.SalesGoalMonths))
			for k := range p.SalesGoalMonths {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			p.SalesGoalTarget = p.SalesGoalMonths[keys[len(keys)-1]]
		}
		return
	}

	if p.SalesGoalMonths == nil {
		p.SalesGoalMonths = make(map[string]float64)
	} else {
		p.SalesGoalMonths = NormalizeSalesGoalMonths(p.SalesGoalMonths)
	}
	if p.SalesGoalTarget > 0 {
		p.SalesGoalMonths[curYM] = math.Round(p.SalesGoalTarget*100) / 100
	}
}
