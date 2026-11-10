package handlers

import (
	"net/http"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminWhatsAppStats expõe as métricas agregadas do dashboard de WhatsApp
// (conversas, leads, vendas e heatmap de atividade).
type AdminWhatsAppStats struct {
	repo *repo.WhatsAppRepo
}

func NewAdminWhatsAppStats(r *repo.WhatsAppRepo) *AdminWhatsAppStats {
	return &AdminWhatsAppStats{repo: r}
}

func (h *AdminWhatsAppStats) parseFilter(r *http.Request) (repo.WhatsAppStatsFilter, error) {
	q := r.URL.Query()
	f := repo.WhatsAppStatsFilter{PixelName: q.Get("pixel_name")}

	f.AccountID = TenantFromContext(r.Context()).AccountID

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, err
		}
		f.From = t
	} else {
		f.From = time.Now().AddDate(0, 0, -30)
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

	overview, err := h.repo.Overview(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "whatsapp stats error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

// Heatmap GET /admin/whatsapp-stats/heatmap?from=&to=&pixel_name=
func (h *AdminWhatsAppStats) Heatmap(w http.ResponseWriter, r *http.Request) {
	f, err := h.parseFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid 'from'/'to' (RFC3339)")
		return
	}

	buckets, err := h.repo.Heatmap(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "whatsapp heatmap error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"buckets":     buckets,
		"window_days": 7, // limitado pela retenção de whatsapp_messages (prune diário)
	})
}

// CTWALeads GET /admin/whatsapp-stats/ctwa-leads?from=&to=&pixel_name=
func (h *AdminWhatsAppStats) CTWALeads(w http.ResponseWriter, r *http.Request) {
	f, err := h.parseFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid 'from'/'to' (RFC3339)")
		return
	}

	leads, err := h.repo.CTWALeads(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "whatsapp ctwa leads error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, leads)
}
