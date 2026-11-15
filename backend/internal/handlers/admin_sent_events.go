package handlers

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

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
// (Meta, TikTok, Google Ads, GA4).
type AdminSentEvents struct {
	repo     *repo.SentEventRepo
	enqueuer *queue.Enqueuer
	rdb      *redis.Client
}

func NewAdminSentEvents(r *repo.SentEventRepo, enqueuer *queue.Enqueuer) *AdminSentEvents {
	return &AdminSentEvents{repo: r, enqueuer: enqueuer}
}

func (h *AdminSentEvents) WithRedis(rdb *redis.Client) *AdminSentEvents {
	h.rdb = rdb
	return h
}

func (h *AdminSentEvents) SetRedis(rdb *redis.Client) {
	h.rdb = rdb
}

func parseListFilter(r *http.Request) (repo.ListFilter, error) {
	q := r.URL.Query()
	f := repo.ListFilter{
		PixelName:          strings.TrimSpace(q.Get("pixel")),
		Channel:            strings.TrimSpace(q.Get("channel")),
		Search:             strings.TrimSpace(q.Get("search")),
		EventNames:         splitCSV(q.Get("event")),
		ExternalID:         strings.TrimSpace(q.Get("external_id")),
		Status:             strings.TrimSpace(q.Get("status")),
		IncludeLTV:         q.Get("include_ltv") == "true",
		IncludeBuyer:       q.Get("include_buyer") == "true" || q.Get("include_ltv") == "true",
		IncludeUserContext: q.Get("include_user_context") == "true",
		TrafficSource:      strings.TrimSpace(q.Get("traffic_source")),
	}
	scope := TenantFromContext(r.Context())
	f.AccountID = scope.AccountID
	if v := strings.TrimSpace(q.Get("from")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, err
		}
		f.From = t
	}
	if v := strings.TrimSpace(q.Get("to")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, err
		}
		if t.Second() == 0 && t.Nanosecond() == 0 {
			t = t.Add(time.Minute)
		}
		f.To = t
	}
	if v := strings.TrimSpace(q.Get("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, err
		}
		f.Limit = n
	}
	if v := strings.TrimSpace(q.Get("offset")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, err
		}
		f.Offset = n
	}
	return f, nil
}

// List GET /admin/sent-events (listagem flat: uma linha por canal).
func (h *AdminSentEvents) List(w http.ResponseWriter, r *http.Request) {
	f, err := parseListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "filtros inválidos")
		return
	}

	list, total, err := h.repo.List(r.Context(), f)
	if err != nil {
		slog.Error("list sent events failed", "err", err, "account_id", f.AccountID)
		writeError(w, http.StatusInternalServerError, "erro ao listar eventos")
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
		writeError(w, http.StatusBadRequest, "filtros inválidos")
		return
	}

	list, total, err := h.repo.ListGrouped(r.Context(), f)
	if err != nil {
		slog.Error("list grouped sent events failed", "err", err, "account_id", f.AccountID)
		writeError(w, http.StatusInternalServerError, "erro ao listar eventos")
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
	rawEventID := chi.URLParam(r, "event_id")
	if rawEventID == "" {
		rawEventID = r.URL.Query().Get("event_id")
	}
	if rawEventID == "" {
		writeError(w, http.StatusBadRequest, "event_id obrigatório")
		return
	}

	eventID := rawEventID
	if unescaped, err := url.PathUnescape(rawEventID); err == nil && unescaped != "" {
		eventID = unescaped
	}

	pixelName := strings.TrimSpace(r.URL.Query().Get("pixel"))
	scope := TenantFromContext(r.Context())

	rows, err := h.repo.FindByEventID(r.Context(), eventID, pixelName, scope.AccountID)
	if err != nil {
		slog.Error("find sent event by id failed", "err", err, "event_id", eventID, "account_id", scope.AccountID)
		writeError(w, http.StatusInternalServerError, "erro ao carregar detalhes do evento")
		return
	}
	if len(rows) == 0 && eventID != rawEventID {
		rows, _ = h.repo.FindByEventID(r.Context(), rawEventID, pixelName, scope.AccountID)
	}
	if len(rows) == 0 && pixelName != "" {
		rows, _ = h.repo.FindByEventID(r.Context(), eventID, "", scope.AccountID)
		if len(rows) == 0 && eventID != rawEventID {
			rows, _ = h.repo.FindByEventID(r.Context(), rawEventID, "", scope.AccountID)
		}
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "evento não encontrado")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"event_id":    rows[0].EventID,
		"event_name":  rows[0].EventName,
		"pixel_name":  rows[0].PixelName,
		"external_id": rows[0].ExternalID,
		"created_at":  rows[0].CreatedAt,
		"channels":    rows,
	})
}
