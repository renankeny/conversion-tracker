package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/config"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminDashboardMeta struct {
	pixelRepo *repo.PixelRepo
	statsRepo *repo.StatsRepo
	metaAccs  *repo.MetaAccountRepo
	metaOAuth *metaoauth.Client
	qcache    *cache.QueryCache
}

func NewAdminDashboardMeta(pr *repo.PixelRepo) *AdminDashboardMeta {
	return &AdminDashboardMeta{pixelRepo: pr}
}

func (h *AdminDashboardMeta) SetStatsRepo(sr *repo.StatsRepo) { h.statsRepo = sr }
func (h *AdminDashboardMeta) SetQueryCache(qc *cache.QueryCache) { h.qcache = qc }
func (h *AdminDashboardMeta) SetMetaOAuth(metaAccs *repo.MetaAccountRepo, metaOAuth *metaoauth.Client) {
	h.metaAccs = metaAccs
	h.metaOAuth = metaOAuth
}

type resolvedMetaSource struct {
	pixelName   string
	accountID   int64
	adAccountID string
	legacyToken string
	oauthAcc    *domain.MetaAccount
}

func (h *AdminDashboardMeta) resolveMetaSource(r *http.Request) (*resolvedMetaSource, error) {
	pixelName := strings.TrimSpace(r.URL.Query().Get("pixel_name"))
	if pixelName == "" {
		return nil, errors.New("pixel_name is required")
	}
	p, err := h.pixelRepo.FindByName(r.Context(), pixelName)
	if err != nil {
		return nil, errors.New("pixel not found")
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && p.AccountID != scope.AccountID {
		return nil, errors.New("forbidden")
	}

	if h.metaAccs != nil && h.metaOAuth != nil {
		var acc *domain.MetaAccount
		if p.MetaAccountID != nil && *p.MetaAccountID > 0 {
			acc, _ = h.metaAccs.FindByID(r.Context(), *p.MetaAccountID)
		}
		if acc == nil || !acc.IsConnected() {
			acc, _ = h.metaAccs.FindFirstConnectedByAccount(r.Context(), p.AccountID)
		}
		if acc != nil && acc.Enabled && acc.IsConnected() {
			adAccID := strings.TrimSpace(p.MetaAdAccountID)
			if adAccID == "" {
				adAccID = strings.TrimSpace(acc.AdAccountID)
			}
			if adAccID != "" {
				return &resolvedMetaSource{
					pixelName: p.Name, accountID: p.AccountID,
					adAccountID: adAccID, oauthAcc: acc,
				}, nil
			}
		}
	}

	if p.MetaPermanentToken != "" && p.MetaAdAccountID != "" {
		return &resolvedMetaSource{
			pixelName: p.Name, accountID: p.AccountID,
			adAccountID: p.MetaAdAccountID, legacyToken: p.MetaPermanentToken,
		}, nil
	}
	return nil, errors.New("pixel sem Conta Meta Ads OAuth ou Ad Account ID configurado")
}

func parseSinceUntilDates(r *http.Request) (string, string) {
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromRaw == "" || toRaw == "" {
		return "", ""
	}
	loc, err := time.LoadLocation(config.ProjectTimezone)
	if err != nil {
		loc = time.FixedZone("BRT", -3*3600)
	}
	fromDate, fromTime, hasFromTime := parseDateInLocation(fromRaw, loc)
	toDate, toTime, hasToTime := parseDateInLocation(toRaw, loc)
	if hasFromTime && hasToTime && toTime.After(fromTime) &&
		toTime.Hour() == 0 && toTime.Minute() == 0 && toTime.Second() == 0 && toTime.Nanosecond() == 0 {
		toDate = toTime.Add(-time.Second).Format("2006-01-02")
	}
	return fromDate, toDate
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

func (h *AdminDashboardMeta) fetchMetaSummary(ctx context.Context, src *resolvedMetaSource, tr string) (meta.LiveInsightsSummary, error) {
	if src.oauthAcc != nil {
		return h.metaOAuth.FetchLiveInsights(ctx, src.oauthAcc, src.adAccountID, tr)
	}
	return meta.FetchLiveInsights(ctx, src.adAccountID, src.legacyToken, tr)
}

func (h *AdminDashboardMeta) LiveInsights(w http.ResponseWriter, r *http.Request) {
	src, err := h.resolveMetaSource(r)
	if err != nil {
		writeJSON(w, http.StatusOK, meta.LiveInsightsSummary{Connected: false})
		return
	}
	since, until := parseSinceUntilDates(r)
	timeRange := marshalMetaRange(since, until)
	if timeRange == "" {
		writeError(w, http.StatusBadRequest, "missing from and to dates")
		return
	}
	prevSince, prevUntil := previousYMDWindow(since, until)
	prevRange := marshalMetaRange(prevSince, prevUntil)

	key := cache.HashQueryKey("meta:insights:v3", src.adAccountID, timeRange)
	raw, err := h.qcache.GetOrLoadBytes(r.Context(), key, 3*time.Minute, func(ctx context.Context) ([]byte, error) {
		summary, ferr := h.fetchMetaSummary(ctx, src, timeRange)
		if ferr != nil {
			return nil, ferr
		}
		if prevRange != "" {
			if prev, perr := h.fetchMetaSummary(ctx, src, prevRange); perr == nil {
				summary.PrevSpend = prev.Spend
				summary.PrevClicks = prev.Clicks
			}
		}
		summary.Connected = true
		return json.Marshal(summary)
	})
	if err != nil {
		writeJSON(w, http.StatusOK, meta.LiveInsightsSummary{Connected: false})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}

func (h *AdminDashboardMeta) CreativeRanking(w http.ResponseWriter, r *http.Request) {
	src, err := h.resolveMetaSource(r)
	if err != nil {
		writeJSON(w, http.StatusOK, []meta.CreativeInsight{})
		return
	}
	since, until := parseSinceUntilDates(r)
	timeRange := marshalMetaRange(since, until)
	if timeRange == "" {
		writeError(w, http.StatusBadRequest, "missing from and to dates")
		return
	}
	fromTime, _ := time.Parse(time.RFC3339, strings.TrimSpace(r.URL.Query().Get("from")))
	toTime, _ := time.Parse(time.RFC3339, strings.TrimSpace(r.URL.Query().Get("to")))

	key := cache.HashQueryKey("meta:creatives:v3", src.pixelName, src.adAccountID, timeRange)
	raw, err := h.qcache.GetOrLoadBytes(r.Context(), key, 3*time.Minute, func(ctx context.Context) ([]byte, error) {
		var (
			ranking []meta.CreativeInsight
			ferr    error
		)
		if src.oauthAcc != nil {
			ranking, ferr = h.metaOAuth.FetchCreativeRanking(ctx, src.oauthAcc, src.adAccountID, timeRange)
		} else {
			ranking, ferr = meta.FetchCreativeRanking(ctx, src.adAccountID, src.legacyToken, timeRange)
		}
		if ferr != nil {
			return nil, ferr
		}
		if len(ranking) > 3 {
			ranking = ranking[:3]
		}
		if h.statsRepo != nil && len(ranking) > 0 {
			h.statsRepo.EnrichCreativesWithTrackedConversions(ctx, repo.StatsFilter{
				AccountID: src.accountID, PixelName: src.pixelName, From: fromTime, To: toTime,
			}, ranking)
		}
		if ranking == nil {
			ranking = []meta.CreativeInsight{}
		}
		return json.Marshal(ranking)
	})
	if err != nil {
		writeJSON(w, http.StatusOK, []meta.CreativeInsight{})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}
