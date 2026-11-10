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
	metaAccs  *repo.MetaAccountRepo
	metaOAuth *metaoauth.Client
	qcache    *cache.QueryCache
}

func NewAdminDashboardMeta(pr *repo.PixelRepo) *AdminDashboardMeta {
	return &AdminDashboardMeta{pixelRepo: pr}
}

func (h *AdminDashboardMeta) SetQueryCache(qc *cache.QueryCache) {
	h.qcache = qc
}

func (h *AdminDashboardMeta) SetMetaOAuth(metaAccs *repo.MetaAccountRepo, metaOAuth *metaoauth.Client) {
	h.metaAccs = metaAccs
	h.metaOAuth = metaOAuth
}

type resolvedMetaSource struct {
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
				return &resolvedMetaSource{adAccountID: adAccID, oauthAcc: acc}, nil
			}
		}
	}

	if p.MetaPermanentToken != "" && p.MetaAdAccountID != "" {
		return &resolvedMetaSource{
			adAccountID: p.MetaAdAccountID,
			legacyToken: p.MetaPermanentToken,
		}, nil
	}
	return nil, errors.New("pixel sem Conta Meta Ads OAuth ou Ad Account ID configurado")
}

func getTimeRange(r *http.Request) string {
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromRaw == "" || toRaw == "" {
		return ""
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
	tr := map[string]string{"since": fromDate, "until": toDate}
	b, _ := json.Marshal(tr)
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

func (h *AdminDashboardMeta) LiveInsights(w http.ResponseWriter, r *http.Request) {
	src, err := h.resolveMetaSource(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	timeRange := getTimeRange(r)
	if timeRange == "" {
		writeError(w, http.StatusBadRequest, "missing from and to dates")
		return
	}
	key := cache.HashQueryKey("meta:spend", src.adAccountID, timeRange)
	raw, err := h.qcache.GetOrLoadBytes(r.Context(), key, 3*time.Minute, func(ctx context.Context) ([]byte, error) {
		var (
			spend float64
			ferr  error
		)
		if src.oauthAcc != nil {
			spend, ferr = h.metaOAuth.FetchLiveSpend(ctx, src.oauthAcc, src.adAccountID, timeRange)
		} else {
			spend, ferr = meta.FetchLiveSpend(ctx, src.adAccountID, src.legacyToken, timeRange)
		}
		if ferr != nil {
			return nil, ferr
		}
		return json.Marshal(map[string]any{"spend": spend})
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "meta api error: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}

func (h *AdminDashboardMeta) CreativeRanking(w http.ResponseWriter, r *http.Request) {
	src, err := h.resolveMetaSource(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	timeRange := getTimeRange(r)
	if timeRange == "" {
		writeError(w, http.StatusBadRequest, "missing from and to dates")
		return
	}
	key := cache.HashQueryKey("meta:creatives", src.adAccountID, timeRange)
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
		return json.Marshal(ranking)
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "meta api error: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}
