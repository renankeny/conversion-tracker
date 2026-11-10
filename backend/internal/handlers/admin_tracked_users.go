package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// AdminTrackedUsers expõe o GET /admin/tracked-users pra listagem no painel
// e PUT /admin/tracked-users/{id} pra edição manual de qualquer campo.
type AdminTrackedUsers struct {
	repo       *repo.UserRepo
	sentEvents *repo.SentEventRepo
	pixels     *repo.PixelRepo
}

func NewAdminTrackedUsers(r *repo.UserRepo, s *repo.SentEventRepo) *AdminTrackedUsers {
	return &AdminTrackedUsers{repo: r, sentEvents: s}
}

func (h *AdminTrackedUsers) SetPixels(p *repo.PixelRepo) {
	h.pixels = p
}

// List GET /admin/tracked-users?pixel=&search=&limit=&offset=
//
// `search` casa parcialmente (ILIKE) em external_id, id_wpp, email e phone.
// Aplica filtro multi-tenant: members só veem tracked users da própria conta.
func (h *AdminTrackedUsers) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repo.TrackedUserFilter{
		PixelName: q.Get("pixel"),
		Search:    q.Get("search"),
	}
	scope := TenantFromContext(r.Context())
	f.AccountID = scope.AccountID
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

// Lookup GET /admin/tracked-users/lookup?id=&external_id=&email=&phone=
func (h *AdminTrackedUsers) Lookup(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	scope := TenantFromContext(r.Context())
	lookupAccID := int64(0)
	if !scope.IsOwner {
		lookupAccID = scope.AccountID
	}

	var (
		user *domain.TrackedUser
		err  error
	)

	switch {
	case q.Get("id") != "":
		id, parseErr := strconv.ParseInt(q.Get("id"), 10, 64)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "parâmetro 'id' inválido: deve ser inteiro")
			return
		}
		user, err = h.repo.FindByID(r.Context(), id)

	case q.Get("external_id") != "":
		user, err = h.repo.FindByExternalIDByAccount(r.Context(), strings.TrimSpace(q.Get("external_id")), lookupAccID)

	case q.Get("email") != "":
		normalized := util.NormalizeEmail(q.Get("email"))
		if normalized == "" {
			writeError(w, http.StatusBadRequest, "parâmetro 'email' inválido")
			return
		}
		user, err = h.repo.FindByEmailByAccount(r.Context(), normalized, lookupAccID)

	case q.Get("phone") != "":
		normalized := util.NormalizePhone(q.Get("phone"))
		if normalized == "" {
			writeError(w, http.StatusBadRequest, "parâmetro 'phone' inválido")
			return
		}
		user, err = h.repo.FindByPhoneVariantsByAccount(r.Context(), util.PhoneVariants(normalized), lookupAccID)

	default:
		writeError(w, http.StatusBadRequest, "informe ao menos um parâmetro: id, external_id, email ou phone")
		return
	}

	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro na busca: "+err.Error())
		return
	}

	if !canEdit(r, user) {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}

	_ = h.repo.EnrichLTV(r.Context(), user)
	writeJSON(w, http.StatusOK, user)
}

// Get GET /admin/tracked-users/{id} — carrega 1 tracked_user.
func (h *AdminTrackedUsers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	user, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error: "+err.Error())
		return
	}
	if !canEdit(r, user) {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	writeJSON(w, http.StatusOK, user)
}
