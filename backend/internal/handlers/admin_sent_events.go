package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// AdminSentEvents expõe a auditoria de eventos enviados às plataformas
// (Meta, TikTok, Google Ads).
type AdminSentEvents struct {
	repo     *repo.SentEventRepo
	enqueuer *queue.Enqueuer
}

func NewAdminSentEvents(r *repo.SentEventRepo, enqueuer *queue.Enqueuer) *AdminSentEvents {
	return &AdminSentEvents{repo: r, enqueuer: enqueuer}
}

func parseListFilter(r *http.Request) (repo.ListFilter, error) {
	q := r.URL.Query()
	f := repo.ListFilter{
		PixelName:          q.Get("pixel"),
		EventNames:         splitCSV(q.Get("event")),
		ExternalID:         q.Get("external_id"),
		Status:             q.Get("status"),
		IncludeLTV:         q.Get("include_ltv") == "true",
		IncludeUserContext: q.Get("include_user_context") == "true",
		TrafficSource:      q.Get("traffic_source"),
	}
	scope := TenantFromContext(r.Context())
	f.AccountID = scope.AccountID
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
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, err
		}
		f.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, err
		}
		f.Offset = n
	}
	return f, nil
}

// List GET /admin/sent-events (listagem flat — uma linha por canal).
func (h *AdminSentEvents) List(w http.ResponseWriter, r *http.Request) {
	f, err := parseListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid filter: "+err.Error())
		return
	}

	list, total, err := h.repo.List(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":   list,
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	})
}

// ListGrouped GET /admin/sent-events/grouped (agrupado por evento).
func (h *AdminSentEvents) ListGrouped(w http.ResponseWriter, r *http.Request) {
	f, err := parseListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid filter: "+err.Error())
		return
	}

	list, total, err := h.repo.ListGrouped(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":   list,
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	})
}

// GetByEventID retorna todas as linhas (1 por canal) com payload + response_body completos.
func (h *AdminSentEvents) GetByEventID(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "event_id")
	if eventID == "" {
		writeError(w, http.StatusBadRequest, "event_id obrigatorio na URL")
		return
	}
	pixelName := r.URL.Query().Get("pixel")
	if pixelName == "" {
		writeError(w, http.StatusBadRequest, "query param 'pixel' obrigatorio")
		return
	}

	scope := TenantFromContext(r.Context())
	rows, err := h.repo.FindByEventID(r.Context(), eventID, pixelName, scope.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load error: "+err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "evento nao encontrado")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"event_id":    eventID,
		"event_name":  rows[0].EventName,
		"pixel_name":  rows[0].PixelName,
		"external_id": rows[0].ExternalID,
		"created_at":  rows[0].CreatedAt,
		"channels":    rows,
	})
}
