package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminWhatsAppStats expõe as métricas agregadas do dashboard de WhatsApp.
type AdminWhatsAppStats struct {
	repo   *repo.WhatsAppRepo
	qcache *cache.QueryCache
}

func NewAdminWhatsAppStats(r *repo.WhatsAppRepo) *AdminWhatsAppStats {
	return &AdminWhatsAppStats{repo: r}
}

func (h *AdminWhatsAppStats) SetQueryCache(qc *cache.QueryCache) {
	h.qcache = qc
}

func (h *AdminWhatsAppStats) parseFilter(r *http.Request) (repo.WhatsAppStatsFilter, error) {
	q := r.URL.Query()
	now := time.Now()
	f := repo.WhatsAppStatsFilter{
		PixelName: q.Get("pixel_name"),
		AccountID: TenantFromContext(r.Context()).AccountID,
		From:      now.AddDate(0, 0, -30),
		To:        now,
	}

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, err
		}
		f.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, err
		}
		f.To = t
	}
	return f, nil
}

// Overview GET /admin/whatsapp-stats/overview?from=&to=&pixel_name=
func (h *AdminWhatsAppStats) Overview(w http.ResponseWriter, r *http.Request) {
	f, err := h.parseFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid 'from'/'to' (RFC3339)")
		return
	}
	ttl := dashboardCacheTTL(f.To, 30*time.Second, 5*time.Minute)
	writeDashboardCachedJSON(w, r, h.qcache, f.AccountID, "wa_overview", ttl, "whatsapp stats error", func(ctx context.Context) (any, error) {
		return h.repo.Overview(ctx, f)
	})
}

// Heatmap GET /admin/whatsapp-stats/heatmap?from=&to=&pixel_name=
func (h *AdminWhatsAppStats) Heatmap(w http.ResponseWriter, r *http.Request) {
	f, err := h.parseFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid 'from'/'to' (RFC3339)")
		return
	}
	ttl := dashboardCacheTTL(f.To, 30*time.Second, 5*time.Minute)
	writeDashboardCachedJSON(w, r, h.qcache, f.AccountID, "wa_heatmap", ttl, "whatsapp heatmap error", func(ctx context.Context) (any, error) {
		buckets, lerr := h.repo.Heatmap(ctx, f)
		if lerr != nil {
			return nil, lerr
		}
		return map[string]any{
			"buckets":     buckets,
			"window_days": 7,
		}, nil
	})
}

// CTWALeads GET /admin/whatsapp-stats/ctwa-leads?from=&to=&pixel_name=
func (h *AdminWhatsAppStats) CTWALeads(w http.ResponseWriter, r *http.Request) {
	f, err := h.parseFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid 'from'/'to' (RFC3339)")
		return
	}
	ttl := dashboardCacheTTL(f.To, 30*time.Second, 5*time.Minute)
	writeDashboardCachedJSON(w, r, h.qcache, f.AccountID, "wa_ctwa_leads", ttl, "whatsapp ctwa leads error", func(ctx context.Context) (any, error) {
		return h.repo.CTWALeads(ctx, f)
	})
}
