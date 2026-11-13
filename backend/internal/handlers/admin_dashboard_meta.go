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

func (h *AdminDashboardMeta) SetStatsRepo(sr *repo.StatsRepo)     { h.statsRepo = sr }
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

func (s *resolvedMetaSource) credKey() string {
	if s.oauthAcc != nil {
		return "oauth:" + strconv.FormatInt(s.oauthAcc.ID, 10)
	}
	if len(s.legacyToken) >= 8 {
		return "leg:" + s.legacyToken[len(s.legacyToken)-8:]
	}
	return "leg"
}

func (h *AdminDashboardMeta) resolveMetaSourceForPixel(ctx context.Context, p *domain.Pixel) (*resolvedMetaSource, error) {
	if h.metaAccs != nil && h.metaOAuth != nil {
		var acc *domain.MetaAccount
		if p.MetaAccountID != nil && *p.MetaAccountID > 0 {
			acc, _ = h.metaAccs.FindByID(ctx, *p.MetaAccountID)
		}
		if acc == nil || !acc.IsConnected() {
			acc, _ = h.metaAccs.FindFirstConnectedByAccount(ctx, p.AccountID)
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
	return h.resolveMetaSourceForPixel(r.Context(), p)
}

func (h *AdminDashboardMeta) fetchMetaSummary(ctx context.Context, src *resolvedMetaSource, tr string) (meta.LiveInsightsSummary, error) {
	if src.oauthAcc != nil {
		return h.metaOAuth.FetchLiveInsights(ctx, src.oauthAcc, src.adAccountID, tr)
	}
	return meta.FetchLiveInsights(ctx, src.adAccountID, src.legacyToken, tr)
}

func (h *AdminDashboardMeta) loadCachedSummary(ctx context.Context, src *resolvedMetaSource, since, until string) ([]byte, error) {
	timeRange := marshalMetaRange(since, until)
	prevSince, prevUntil := previousYMDWindow(since, until)
	prevRange := marshalMetaRange(prevSince, prevUntil)

	key := cache.HashQueryKey("meta:insights:v4", strconv.FormatInt(src.accountID, 10), src.credKey(), src.adAccountID, timeRange)
	return h.qcache.GetOrLoadWithNegCache(ctx, key, 3*time.Minute, 20*time.Second, []byte(`{"connected":false}`), func(lctx context.Context) ([]byte, error) {
		summary, ferr := h.fetchMetaSummary(lctx, src, timeRange)
		if ferr != nil {
			return nil, ferr
		}
		if prevRange != "" {
			if prev, perr := h.fetchMetaSummary(lctx, src, prevRange); perr == nil {
				summary.PrevSpend = prev.Spend
				summary.PrevClicks = prev.Clicks
			}
		}
		summary.Connected = true
		return json.Marshal(summary)
	})
}

func (h *AdminDashboardMeta) LoadLiveSummaryForPixel(ctx context.Context, p *domain.Pixel, since, until string) meta.LiveInsightsSummary {
	if since == "" || until == "" {
		return meta.LiveInsightsSummary{Connected: false}
	}
	src, err := h.resolveMetaSourceForPixel(ctx, p)
	if err != nil {
		return meta.LiveInsightsSummary{Connected: false}
	}
	raw, err := h.loadCachedSummary(ctx, src, since, until)
	if err != nil {
		return meta.LiveInsightsSummary{Connected: false}
	}
	var out meta.LiveInsightsSummary
	if json.Unmarshal(raw, &out) != nil {
		return meta.LiveInsightsSummary{Connected: false}
	}
	return out
}

func (h *AdminDashboardMeta) LiveInsights(w http.ResponseWriter, r *http.Request) {
	src, err := h.resolveMetaSource(r)
	if err != nil {
		writeJSON(w, http.StatusOK, meta.LiveInsightsSummary{Connected: false})
		return
	}
	since, until := parseSinceUntilDates(r)
	if since == "" || until == "" {
		writeError(w, http.StatusBadRequest, "missing from and to dates")
		return
	}
	raw, err := h.loadCachedSummary(r.Context(), src, since, until)
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
	fromTime, toTime := parseDashboardTimeBounds(r)
	ver := h.qcache.DashboardVersion(r.Context(), src.accountID)

	key := cache.HashQueryKey("meta:creatives:v4", strconv.FormatInt(src.accountID, 10), ver, src.credKey(), src.pixelName, src.adAccountID, timeRange)
	raw, err := h.qcache.GetOrLoadWithNegCache(r.Context(), key, 3*time.Minute, 20*time.Second, []byte("[]"), func(ctx context.Context) ([]byte, error) {
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
