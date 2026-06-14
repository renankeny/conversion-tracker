package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminSentEvents expõe a auditoria de eventos enviados à Meta CAPI.
// Usado pelo painel pra mostrar histórico, filtrar por status, debugar
// erros, etc.
type AdminSentEvents struct {
	repo *repo.SentEventRepo
}

func NewAdminSentEvents(r *repo.SentEventRepo) *AdminSentEvents {
	return &AdminSentEvents{repo: r}
}

// List GET /admin/sent-events
//
// Query params (todos opcionais):
//
//	pixel       — filtra por pixel_name exato
//	event       — filtra por event_name exato (PageView, Purchase, ...)
//	status      — queued | sent | failed
//	from / to   — RFC3339 timestamp (created_at >= from, < to)
//	limit       — default 50, max 500
//	offset      — default 0
func (h *AdminSentEvents) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repo.ListFilter{
		PixelName: q.Get("pixel"),
		EventName: q.Get("event"),
		Status:    q.Get("status"),
	}
	if scope := TenantFromContext(r.Context()); !scope.IsOwner {
		f.AccountID = scope.AccountID
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' (RFC3339): "+err.Error())
			return
		}
		f.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' (RFC3339): "+err.Error())
			return
		}
		f.To = t
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'limit'")
			return
		}
		f.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'offset'")
			return
		}
		f.Offset = n
	}

	list, total, err := h.repo.List(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  list,
		"total": total,
		"limit": f.Limit,
		"offset": f.Offset,
	})
}
