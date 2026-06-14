package handlers

import (
	"encoding/json"
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
	repo *repo.UserRepo
}

func NewAdminTrackedUsers(r *repo.UserRepo) *AdminTrackedUsers {
	return &AdminTrackedUsers{repo: r}
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
	if scope := TenantFromContext(r.Context()); !scope.IsOwner {
		f.AccountID = scope.AccountID
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

// trackedUserUpdateInput é o payload editável. Todos os campos são opcionais
// — só atualiza o que vier preenchido. Para limpar um campo, envie string
// vazia explícita (não null).
type trackedUserUpdateInput struct {
	IDWpp           *string `json:"id_wpp"`
	PixelName       *string `json:"pixel_name"`
	FBP             *string `json:"fbp"`
	FBC             *string `json:"fbc"`
	GCLID           *string `json:"gclid"`
	ClientIP        *string `json:"client_ip"`
	ClientUserAgent *string `json:"client_user_agent"`

	Country    *string `json:"country"`
	State      *string `json:"state"`
	City       *string `json:"city"`
	PostalCode *string `json:"postal_code"`

	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`

	URL *string `json:"url"`

	FirstUTMSource   *string `json:"first_utm_source"`
	FirstUTMMedium   *string `json:"first_utm_medium"`
	FirstUTMCampaign *string `json:"first_utm_campaign"`
	FirstUTMContent  *string `json:"first_utm_content"`
	FirstUTMTerm     *string `json:"first_utm_term"`

	LastUTMSource   *string `json:"last_utm_source"`
	LastUTMMedium   *string `json:"last_utm_medium"`
	LastUTMCampaign *string `json:"last_utm_campaign"`
	LastUTMContent  *string `json:"last_utm_content"`
	LastUTMTerm     *string `json:"last_utm_term"`
}

// Update PUT /admin/tracked-users/{id}
//
// Permite editar qualquer campo editável do tracked_user. Campos protegidos
// (id, account_id, external_id, created_at) NÃO são alterados — pra mover
// entre empresas use endpoint separado (TODO se necessário).
//
// Member só edita usuários da própria conta. Owner edita qualquer.
func (h *AdminTrackedUsers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error: "+err.Error())
		return
	}
	if !canEdit(r, existing) {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}

	var in trackedUserUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Aplica patches. Strings normalizadas (email lowercase, phone só dígitos)
	// pra manter consistência com o fluxo de ingestão.
	applyStr(&existing.IDWpp, in.IDWpp, strings.TrimSpace)
	applyStr(&existing.PixelName, in.PixelName, strings.TrimSpace)
	applyStr(&existing.FBP, in.FBP, strings.TrimSpace)
	applyStr(&existing.FBC, in.FBC, strings.TrimSpace)
	applyStr(&existing.GCLID, in.GCLID, strings.TrimSpace)
	applyStr(&existing.ClientIP, in.ClientIP, strings.TrimSpace)
	applyStr(&existing.ClientUserAgent, in.ClientUserAgent, strings.TrimSpace)
	applyStr(&existing.Country, in.Country, normalizeLowerTrim)
	applyStr(&existing.State, in.State, normalizeLowerTrim)
	applyStr(&existing.City, in.City, normalizeLowerTrim)
	applyStr(&existing.PostalCode, in.PostalCode, strings.TrimSpace)
	applyStr(&existing.FirstName, in.FirstName, strings.TrimSpace)
	applyStr(&existing.LastName, in.LastName, strings.TrimSpace)
	applyStr(&existing.Email, in.Email, util.NormalizeEmail)
	applyStr(&existing.Phone, in.Phone, util.NormalizePhone)
	applyStr(&existing.URL, in.URL, strings.TrimSpace)
	applyStr(&existing.FirstUTMSource, in.FirstUTMSource, strings.TrimSpace)
	applyStr(&existing.FirstUTMMedium, in.FirstUTMMedium, strings.TrimSpace)
	applyStr(&existing.FirstUTMCampaign, in.FirstUTMCampaign, strings.TrimSpace)
	applyStr(&existing.FirstUTMContent, in.FirstUTMContent, strings.TrimSpace)
	applyStr(&existing.FirstUTMTerm, in.FirstUTMTerm, strings.TrimSpace)
	applyStr(&existing.LastUTMSource, in.LastUTMSource, strings.TrimSpace)
	applyStr(&existing.LastUTMMedium, in.LastUTMMedium, strings.TrimSpace)
	applyStr(&existing.LastUTMCampaign, in.LastUTMCampaign, strings.TrimSpace)
	applyStr(&existing.LastUTMContent, in.LastUTMContent, strings.TrimSpace)
	applyStr(&existing.LastUTMTerm, in.LastUTMTerm, strings.TrimSpace)

	if err := h.repo.UpdateByID(r.Context(), existing); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}

	// Recarrega pra devolver com updated_at atualizado.
	fresh, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}
	writeJSON(w, http.StatusOK, fresh)
}

// canEdit aplica scope multi-tenant: member só edita da própria conta.
func canEdit(r *http.Request, u *domain.TrackedUser) bool {
	scope := TenantFromContext(r.Context())
	if scope.IsOwner {
		return true
	}
	return u.AccountID == scope.AccountID
}

// applyStr aplica patch[*string] em dst se patch != nil. Aceita normalizer
// opcional (email lowercase, phone só dígitos, etc).
func applyStr(dst *string, src *string, norm func(string) string) {
	if src == nil {
		return
	}
	v := *src
	if norm != nil {
		v = norm(v)
	}
	*dst = v
}

// normalizeLowerTrim é o normalizer usado em country/state/city — match o
// formato gravado pelo GeoIP (ASCII lowercase).
func normalizeLowerTrim(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
