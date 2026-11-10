package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// UpdateEventLabels atualiza os nomes customizados de eventos (cards/funil
// da dashboard) da empresa.
func (h *AdminAccounts) UpdateEventLabels(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role == "employee" {
		writeError(w, http.StatusForbidden, "apenas administradores ou gerentes podem editar empresas")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if !h.repo.CanAccess(r.Context(), c, id) {
		writeError(w, http.StatusForbidden, "sem permissão para editar esta empresa")
		return
	}

	var body struct {
		EventLabelOverrides map[string]string `json:"event_label_overrides"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := h.repo.UpdateEventLabelOverrides(r.Context(), id, body.EventLabelOverrides); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "event_label_overrides": body.EventLabelOverrides})
}

// GetEventsFilter GET /admin/events-filter
func (h *AdminAccounts) GetEventsFilter(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	accountID := scope.EffectiveAccountID(0)

	acc, err := h.repo.FindByID(r.Context(), accountID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events_filter": acc.EventsFilter})
}

// UpdateEventsFilter PUT /admin/events-filter
func (h *AdminAccounts) UpdateEventsFilter(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Pixel  string   `json:"pixel"`
		Events []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	scope := TenantFromContext(r.Context())
	accountID := scope.EffectiveAccountID(0)

	if err := h.repo.UpdateEventsFilter(r.Context(), accountID, body.Pixel, body.Events); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pixel": body.Pixel, "events": body.Events})
}

func validSlug(s string) bool {
	if len(s) < 2 {
		return false
	}
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}
	return true
}

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	out := make([]rune, 0, len(s))
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else if c == ' ' || c == '_' {
			out = append(out, '-')
		}
	}
	result := string(out)
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}
