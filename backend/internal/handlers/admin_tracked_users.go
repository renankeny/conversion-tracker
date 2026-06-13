package handlers

import (
	"net/http"
	"strconv"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminTrackedUsers expõe o GET /admin/tracked-users pra listagem no painel.
type AdminTrackedUsers struct {
	repo *repo.UserRepo
}

func NewAdminTrackedUsers(r *repo.UserRepo) *AdminTrackedUsers {
	return &AdminTrackedUsers{repo: r}
}

// List GET /admin/tracked-users?pixel=&search=&limit=&offset=
//
// `search` casa parcialmente (ILIKE) em external_id, id_wpp, email e phone.
func (h *AdminTrackedUsers) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repo.TrackedUserFilter{
		PixelName: q.Get("pixel"),
		Search:    q.Get("search"),
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
		"data":   list,
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	})
}
