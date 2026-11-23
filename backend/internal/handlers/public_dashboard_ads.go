package handlers

import (
	"context"
	"net/http"
	"sync"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func (h *PublicDashboard) SetAdsHandlers(metaH *AdminDashboardMeta, googleH *AdminDashboardGoogle) {
	h.metaH = metaH
	h.googleH = googleH
}

func (h *PublicDashboard) SetTrackableLinksRepo(links *repo.TrackableLinkRepo, publicAPIURL string) {
	h.links = links
	h.publicAPIURL = publicAPIURL
}

func (h *PublicDashboard) TrackableLinks(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	if h.links == nil {
		writeJSON(w, http.StatusOK, domain.TrackableLinksOverview{Links: []domain.TrackableLink{}})
		return
	}
	from, to := parsePublicRange(r)
	h.writeCached(w, r, "trackable_links", p, to, func(ctx context.Context) (any, error) {
		return h.links.StatsByFilter(ctx, repo.TrackableLinkStatsFilter{
			AccountID: p.AccountID,
			PixelID:   p.ID,
			From:      &from,
			To:        &to,
		}, h.publicAPIURL)
	})
}

// AdsSummary carrega em paralelo os indicadores ao vivo de Meta Ads e Google Ads
// para o pixel compartilhado (com janela de 120s para APIs externas).
func (h *PublicDashboard) AdsSummary(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	since, until := parseSinceUntilDates(r)
	var (
		wg        sync.WaitGroup
		metaSum   meta.LiveInsightsSummary
		googleSum googleads.LiveInsightsSummary
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		if h.metaH != nil {
			metaSum = h.metaH.LoadLiveSummaryForPixel(r.Context(), p, since, until)
		}
	}()
	go func() {
		defer wg.Done()
		if h.googleH != nil {
			googleSum = h.googleH.LoadLiveSummaryForPixel(r.Context(), p, since, until)
		}
	}()
	wg.Wait()
	writeJSON(w, http.StatusOK, map[string]any{"meta": metaSum, "google": googleSum})
}

// MetaCreatives expõe o Top 3 de criativos da Meta Ads enriquecidos com conversões reais do painel.
func (h *PublicDashboard) MetaCreatives(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	if h.metaH == nil {
		writeJSON(w, http.StatusOK, []meta.CreativeInsight{})
		return
	}
	since, until := parseSinceUntilDates(r)
	fromTime, toTime := parseDashboardTimeBounds(r)
	raw := h.metaH.LoadCreativeRankingBytesForPixel(r.Context(), p, since, until, fromTime, toTime)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}

// GoogleKeywords expõe o Top 5 de palavras-chave do Google Ads enriquecidas com conversões por GCLID.
func (h *PublicDashboard) GoogleKeywords(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolvePixel(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "link inválido ou desativado")
		return
	}
	if h.googleH == nil {
		writeJSON(w, http.StatusOK, []googleads.KeywordInsight{})
		return
	}
	since, until := parseSinceUntilDates(r)
	fromTime, toTime := parseDashboardTimeBounds(r)
	raw := h.googleH.LoadKeywordRankingBytesForPixel(r.Context(), p, since, until, fromTime, toTime)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}
