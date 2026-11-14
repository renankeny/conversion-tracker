package handlers

import (
	"net/http"
	"sync"

	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
)

func (h *PublicDashboard) SetAdsHandlers(metaH *AdminDashboardMeta, googleH *AdminDashboardGoogle) {
	h.metaH = metaH
	h.googleH = googleH
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
