package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// AdminTrackedUsers expõe rotas administrativas de listagem, busca, jornada,
// edição e exclusão de visitantes/leads rastreados.
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

// List GET /admin/tracked-users?pixel=&segment=&search=&limit=&offset=
func (h *AdminTrackedUsers) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repo.TrackedUserFilter{
		PixelName: strings.TrimSpace(q.Get("pixel")),
		Segment:   strings.TrimSpace(q.Get("segment")),
		Search:    strings.TrimSpace(q.Get("search")),
	}
	scope := TenantFromContext(r.Context())
	f.AccountID = scope.AccountID
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parâmetro 'limit' inválido")
			return
		}
		f.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parâmetro 'offset' inválido")
			return
		}
		f.Offset = n
	}

	list, total, err := h.repo.List(r.Context(), f)
	if err != nil {
		slog.Error("admin tracked_users list failed", "account_id", f.AccountID, "error", err)
		writeError(w, http.StatusInternalServerError, "erro ao listar usuários rastreados")
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
	lookupAccID := scope.AccountID

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
		user, err = h.repo.FindRawByID(r.Context(), id)

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
		slog.Error("admin tracked_users lookup failed", "account_id", lookupAccID, "error", err)
		writeError(w, http.StatusInternalServerError, "erro ao buscar usuário")
		return
	}

	if !canEdit(r, user) {
		writeError(w, http.StatusForbidden, "sem acesso a este usuário")
		return
	}

	_ = h.repo.EnrichLTV(r.Context(), user)
	writeJSON(w, http.StatusOK, user)
}

// Get GET /admin/tracked-users/{id} carrega 1 tracked_user.
func (h *AdminTrackedUsers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	user, err := h.repo.FindRawByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		slog.Error("admin tracked_users get failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "erro ao carregar usuário")
		return
	}
	if !canEdit(r, user) {
		writeError(w, http.StatusForbidden, "sem acesso a este usuário")
		return
	}
	_ = h.repo.EnrichLTV(r.Context(), user)
	writeJSON(w, http.StatusOK, user)
}
