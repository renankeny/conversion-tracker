package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/config"
)

func projectLocation() *time.Location {
	loc, err := time.LoadLocation(config.ProjectTimezone)
	if err != nil {
		return time.FixedZone("BRT", -3*3600)
	}
	return loc
}

func parseSinceUntilDates(r *http.Request) (string, string) {
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromRaw == "" || toRaw == "" {
		return "", ""
	}
	loc := projectLocation()
	fromDate, fromTime, hasFromTime := parseDateInLocation(fromRaw, loc)
	toDate, toTime, hasToTime := parseDateInLocation(toRaw, loc)
	if hasFromTime && hasToTime && toTime.After(fromTime) &&
		toTime.Hour() == 0 && toTime.Minute() == 0 && toTime.Second() == 0 && toTime.Nanosecond() == 0 {
		toDate = toTime.Add(-time.Second).Format("2006-01-02")
	}
	return fromDate, toDate
}

// parseDashboardTimeBounds converte os parâmetros `from` e `to` (RFC3339 ou YYYY-MM-DD)
// em timestamps exatos na timezone oficial do projeto (America/Sao_Paulo).
func parseDashboardTimeBounds(r *http.Request) (time.Time, time.Time) {
	loc := projectLocation()
	from := parseFlexibleBound(strings.TrimSpace(r.URL.Query().Get("from")), loc, false)
	to := parseFlexibleBound(strings.TrimSpace(r.URL.Query().Get("to")), loc, true)
	return from, to
}

func parseFlexibleBound(raw string, loc *time.Location, isUpperBound bool) time.Time {
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	if len(raw) >= 10 {
		if d, err := time.ParseInLocation("2006-01-02", raw[:10], loc); err == nil {
			if isUpperBound {
				return d.AddDate(0, 0, 1)
			}
			return d
		}
	}
	return time.Time{}
}

func previousYMDWindow(since, until string) (string, string) {
	s, err1 := time.Parse("2006-01-02", since)
	u, err2 := time.Parse("2006-01-02", until)
	if err1 != nil || err2 != nil || u.Before(s) {
		return "", ""
	}
	days := int(u.Sub(s)/(24*time.Hour)) + 1
	return s.AddDate(0, 0, -days).Format("2006-01-02"), s.AddDate(0, 0, -1).Format("2006-01-02")
}

func marshalMetaRange(since, until string) string {
	if since == "" || until == "" {
		return ""
	}
	b, _ := json.Marshal(map[string]string{"since": since, "until": until})
	return string(b)
}

func parseDateInLocation(raw string, loc *time.Location) (string, time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			inLoc := parsed.In(loc)
			return inLoc.Format("2006-01-02"), inLoc, true
		}
	}
	if len(raw) >= 10 {
		return raw[:10], time.Time{}, false
	}
	return raw, time.Time{}, false
}

func dashboardCacheTTL(to time.Time, liveTTL, closedTTL time.Duration) time.Duration {
	if to.IsZero() {
		return liveTTL
	}
	loc := projectLocation()
	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if to.After(todayStart) {
		return liveTTL
	}
	return closedTTL
}

func writeDashboardCachedJSON(
	w http.ResponseWriter,
	r *http.Request,
	qc *cache.QueryCache,
	accountID int64,
	prefix string,
	ttl time.Duration,
	errPrefix string,
	loader func(context.Context) (any, error),
) {
	if qc == nil {
		data, err := loader(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, errPrefix+": "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, data)
		return
	}
	ver := qc.DashboardVersion(r.Context(), accountID)
	key := cache.HashQueryKey("dash:q2:"+prefix, strconv.FormatInt(accountID, 10), ver, r.URL.RawQuery)
	raw, err := qc.GetOrLoadBytes(r.Context(), key, ttl, func(ctx context.Context) ([]byte, error) {
		data, lerr := loader(ctx)
		if lerr != nil {
			return nil, lerr
		}
		return json.Marshal(data)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, errPrefix+": "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}
