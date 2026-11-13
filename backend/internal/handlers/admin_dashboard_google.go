package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminDashboardGoogle struct {
	pixelRepo *repo.PixelRepo
	statsRepo *repo.StatsRepo
	gadsAccs  *repo.GoogleAdsAccountRepo
	gadsCli   *googleads.Client
	qcache    *cache.QueryCache
}

func NewAdminDashboardGoogle(pr *repo.PixelRepo, ga *repo.GoogleAdsAccountRepo, cli *googleads.Client) *AdminDashboardGoogle {
	return &AdminDashboardGoogle{
		pixelRepo: pr,
		gadsAccs:  ga,
		gadsCli:   cli,
	}
}

func (h *AdminDashboardGoogle) SetStatsRepo(sr *repo.StatsRepo)     { h.statsRepo = sr }
func (h *AdminDashboardGoogle) SetQueryCache(qc *cache.QueryCache) { h.qcache = qc }

func (h *AdminDashboardGoogle) resolveGoogleSourceForPixel(ctx context.Context, p *domain.Pixel) (*domain.GoogleAdsAccount, error) {
	if h.gadsAccs == nil || h.gadsCli == nil {
		return nil, errors.New("google ads service not configured")
	}
	if p.GoogleAdsAccountID != nil && *p.GoogleAdsAccountID > 0 {
		if acc, ferr := h.gadsAccs.FindByID(ctx, *p.GoogleAdsAccountID); ferr == nil && acc != nil && acc.Enabled && acc.IsConnected() && acc.CustomerID != "" {
			return acc, nil
		}
	}

	list, err := h.gadsAccs.ListByAccount(ctx, p.AccountID)
	if err == nil {
		for i := range list {
			if list[i].Enabled && list[i].IsConnected() && list[i].CustomerID != "" {
				return &list[i], nil
			}
		}
	}
	return nil, errors.New("pixel sem Conta Google Ads ativa")
}

func (h *AdminDashboardGoogle) resolveGoogleSource(r *http.Request) (*domain.GoogleAdsAccount, *domain.Pixel, error) {
	pixelName := strings.TrimSpace(r.URL.Query().Get("pixel_name"))
	if pixelName == "" {
		return nil, nil, errors.New("pixel_name is required")
	}
	p, err := h.pixelRepo.FindByName(r.Context(), pixelName)
	if err != nil {
		return nil, nil, errors.New("pixel not found")
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && p.AccountID != scope.AccountID {
		return nil, nil, errors.New("forbidden")
	}
	acc, err := h.resolveGoogleSourceForPixel(r.Context(), p)
	if err != nil {
		return nil, nil, err
	}
	return acc, p, nil
}

func (h *AdminDashboardGoogle) loadCachedSummary(ctx context.Context, acc *domain.GoogleAdsAccount, since, until string) ([]byte, error) {
	prevSince, prevUntil := previousYMDWindow(since, until)
	key := cache.HashQueryKey("gads:insights:v3", strconv.FormatInt(acc.AccountID, 10), strconv.FormatInt(acc.ID, 10), acc.CustomerID, since, until)
	return h.qcache.GetOrLoadWithNegCache(ctx, key, 3*time.Minute, 20*time.Second, []byte(`{"connected":false}`), func(lctx context.Context) ([]byte, error) {
		summary, ferr := h.gadsCli.FetchLiveInsights(lctx, acc, since, until)
		if ferr != nil {
			return nil, ferr
		}
		if prevSince != "" && prevUntil != "" {
			if prev, perr := h.gadsCli.FetchLiveInsights(lctx, acc, prevSince, prevUntil); perr == nil {
				summary.PrevSpend = prev.Spend
				summary.PrevClicks = prev.Clicks
			}
		}
		summary.Connected = true
		return json.Marshal(summary)
	})
}

func (h *AdminDashboardGoogle) LoadLiveSummaryForPixel(ctx context.Context, p *domain.Pixel, since, until string) googleads.LiveInsightsSummary {
	if since == "" || until == "" {
		return googleads.LiveInsightsSummary{Connected: false}
	}
	acc, err := h.resolveGoogleSourceForPixel(ctx, p)
	if err != nil {
		return googleads.LiveInsightsSummary{Connected: false}
	}
	raw, err := h.loadCachedSummary(ctx, acc, since, until)
	if err != nil {
		return googleads.LiveInsightsSummary{Connected: false}
	}
	var out googleads.LiveInsightsSummary
	if json.Unmarshal(raw, &out) != nil {
		return googleads.LiveInsightsSummary{Connected: false}
	}
	return out
}

func (h *AdminDashboardGoogle) LiveInsights(w http.ResponseWriter, r *http.Request) {
	acc, _, err := h.resolveGoogleSource(r)
	if err != nil {
		writeJSON(w, http.StatusOK, googleads.LiveInsightsSummary{Connected: false})
		return
	}
	since, until := parseSinceUntilDates(r)
	if since == "" || until == "" {
		writeError(w, http.StatusBadRequest, "missing from and to dates")
		return
	}
	raw, err := h.loadCachedSummary(r.Context(), acc, since, until)
	if err != nil {
		writeJSON(w, http.StatusOK, googleads.LiveInsightsSummary{Connected: false})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}

func (h *AdminDashboardGoogle) KeywordRanking(w http.ResponseWriter, r *http.Request) {
	acc, px, err := h.resolveGoogleSource(r)
	if err != nil {
		writeJSON(w, http.StatusOK, []googleads.KeywordInsight{})
		return
	}
	since, until := parseSinceUntilDates(r)
	if since == "" || until == "" {
		writeError(w, http.StatusBadRequest, "missing from and to dates")
		return
	}
	fromTime, toTime := parseDashboardTimeBounds(r)
	ver := h.qcache.DashboardVersion(r.Context(), px.AccountID)

	key := cache.HashQueryKey("gads:keywords:v3", strconv.FormatInt(px.AccountID, 10), ver, strconv.FormatInt(acc.ID, 10), px.Name, acc.CustomerID, since, until)
	raw, err := h.qcache.GetOrLoadWithNegCache(r.Context(), key, 3*time.Minute, 20*time.Second, []byte("[]"), func(ctx context.Context) ([]byte, error) {
		kw, ferr := h.gadsCli.FetchKeywordRanking(ctx, acc, since, until)
		if ferr != nil {
			return nil, ferr
		}
		if len(kw) > 5 {
			kw = kw[:5]
		}
		if h.statsRepo != nil && len(kw) > 0 {
			h.statsRepo.EnrichGoogleKeywordsWithTrackedConversions(ctx, repo.StatsFilter{
				AccountID: px.AccountID, PixelName: px.Name, From: fromTime, To: toTime,
			}, kw)
		}
		if kw == nil {
			kw = []googleads.KeywordInsight{}
		}
		return json.Marshal(kw)
	})
	if err != nil {
		writeJSON(w, http.StatusOK, []googleads.KeywordInsight{})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}
