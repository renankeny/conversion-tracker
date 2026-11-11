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
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminDashboardGoogle struct {
	pixelRepo *repo.PixelRepo
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

func (h *AdminDashboardGoogle) SetQueryCache(qc *cache.QueryCache) {
	h.qcache = qc
}

func (h *AdminDashboardGoogle) resolveGoogleSource(r *http.Request) (*domain.GoogleAdsAccount, error) {
	if h.gadsAccs == nil || h.gadsCli == nil {
		return nil, errors.New("google ads service not configured")
	}
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

	if p.GoogleAdsAccountID != nil && *p.GoogleAdsAccountID > 0 {
		if acc, ferr := h.gadsAccs.FindByID(r.Context(), *p.GoogleAdsAccountID); ferr == nil && acc != nil && acc.Enabled && acc.IsConnected() && acc.CustomerID != "" {
			return acc, nil
		}
	}

	list, err := h.gadsAccs.ListByAccount(r.Context(), p.AccountID)
	if err == nil {
		for i := range list {
			if list[i].Enabled && list[i].IsConnected() && list[i].CustomerID != "" {
				return &list[i], nil
			}
		}
	}
	return nil, errors.New("pixel sem Conta Google Ads ativa")
}

func parseGoogleDatesFromReq(r *http.Request) (string, string) {
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

func (h *AdminDashboardGoogle) LiveInsights(w http.ResponseWriter, r *http.Request) {
	acc, err := h.resolveGoogleSource(r)
	if err != nil {
		writeJSON(w, http.StatusOK, googleads.LiveInsightsSummary{Connected: false})
		return
	}
	since, until := parseGoogleDatesFromReq(r)
	if since == "" || until == "" {
		writeError(w, http.StatusBadRequest, "missing from and to dates")
		return
	}
	key := cache.HashQueryKey("gads:insights:v1", acc.CustomerID, since, until)
	raw, err := h.qcache.GetOrLoadBytes(r.Context(), key, 3*time.Minute, func(ctx context.Context) ([]byte, error) {
		summary, ferr := h.gadsCli.FetchLiveInsights(ctx, acc, since, until)
		if ferr != nil {
			return nil, ferr
		}
		summary.Connected = true
		return json.Marshal(summary)
	})
	if err != nil {
		writeJSON(w, http.StatusOK, googleads.LiveInsightsSummary{Connected: false})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}

func (h *AdminDashboardGoogle) KeywordRanking(w http.ResponseWriter, r *http.Request) {
	acc, err := h.resolveGoogleSource(r)
	if err != nil {
		writeJSON(w, http.StatusOK, []googleads.KeywordInsight{})
		return
	}
	since, until := parseGoogleDatesFromReq(r)
	if since == "" || until == "" {
		writeError(w, http.StatusBadRequest, "missing from and to dates")
		return
	}
	key := cache.HashQueryKey("gads:keywords:v1", acc.CustomerID, since, until)
	raw, err := h.qcache.GetOrLoadBytes(r.Context(), key, 3*time.Minute, func(ctx context.Context) ([]byte, error) {
		kw, ferr := h.gadsCli.FetchKeywordRanking(ctx, acc, since, until)
		if ferr != nil {
			return nil, ferr
		}
		if len(kw) > 5 {
			kw = kw[:5]
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

