package campaignrules

import (
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func SaoPauloLocation() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil || loc == nil {
		return time.FixedZone("BRT", -3*3600)
	}
	return loc
}

// ResolvePeriodDates converte o preset da regra em janela fechada de datas no fuso America/Sao_Paulo.
func ResolvePeriodDates(preset string, now time.Time) (since, until string, from, to time.Time) {
	loc := SaoPauloLocation()
	localNow := now.In(loc)
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)

	var tFrom, tTo time.Time
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "yesterday":
		tFrom = todayStart.AddDate(0, 0, -1)
		tTo = tFrom
	case "3d":
		tFrom = todayStart.AddDate(0, 0, -2)
		tTo = todayStart
	case "7d":
		tFrom = todayStart.AddDate(0, 0, -6)
		tTo = todayStart
	case "14d":
		tFrom = todayStart.AddDate(0, 0, -13)
		tTo = todayStart
	case "30d":
		tFrom = todayStart.AddDate(0, 0, -29)
		tTo = todayStart
	default:
		tFrom = todayStart
		tTo = todayStart
	}
	return tFrom.Format("2006-01-02"), tTo.Format("2006-01-02"), tFrom, tTo.AddDate(0, 0, 1)
}

// IsRuleDue verifica se a regra deve ser executada pelo scheduler neste instante (São Paulo).
func IsRuleDue(rule domain.CampaignRule, nowSP time.Time) bool {
	if !rule.Enabled {
		return false
	}
	if len(rule.ScheduleDays) > 0 && !containsInt(rule.ScheduleDays, int(nowSP.Weekday())) {
		return false
	}
	if rule.RuleKind == domain.RuleKindSchedule {
		hour, minute, ok := parseHHMM(rule.ScheduleTime)
		if !ok {
			return false
		}
		target := time.Date(nowSP.Year(), nowSP.Month(), nowSP.Day(), hour, minute, 0, 0, nowSP.Location())
		diff := nowSP.Sub(target)
		if diff < 0 || diff >= 20*time.Minute {
			return false
		}
		if rule.LastRunAt != nil && !rule.LastRunAt.In(nowSP.Location()).Before(target) {
			return false
		}
		return true
	}
	if rule.MinHourSP > 0 && nowSP.Hour() < rule.MinHourSP {
		return false
	}
	if rule.LastRunAt == nil {
		return true
	}
	freq := time.Duration(rule.FrequencyMinutes) * time.Minute
	if freq < 15*time.Minute {
		freq = 15 * time.Minute
	}
	return nowSP.Sub(rule.LastRunAt.In(nowSP.Location())) >= freq-30*time.Second
}

func parseHHMM(raw string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func containsInt(list []int, target int) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
