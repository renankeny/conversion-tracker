package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// AdminUsers gerencia o CRUD de admin_users (a tabela de admins do painel).
type AdminUsers struct {
	pool     *pgxpool.Pool
	users    *repo.AdminUserRepo
	accounts *repo.AccountRepo
	sessions *cache.SessionStore
}

func NewAdminUsers(pool *pgxpool.Pool) *AdminUsers {
	return &AdminUsers{
		pool:     pool,
		users:    repo.NewAdminUserRepo(pool),
		accounts: repo.NewAccountRepo(pool),
	}
}

func (h *AdminUsers) SetSessions(s *cache.SessionStore) {
	h.sessions = s
}

type adminUserView struct {
	ID          int64          `json:"id"`
	Email       string         `json:"email"`
	Name        string         `json:"name"`
	Phone       string         `json:"phone"`
	Role        string         `json:"role"`
	AccountID   *int64         `json:"account_id"`
	ParentID    *int64         `json:"parent_id"`
	Permissions map[string]any `json:"permissions"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

func recordToAdminUserView(u repo.AdminUserRecord) adminUserView {
	perms := make(map[string]any)
	if len(u.Permissions) > 0 {
		_ = json.Unmarshal(u.Permissions, &perms)
	}
	if perms == nil {
		perms = make(map[string]any)
	}
	return adminUserView{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		Phone:       u.Phone,
		Role:        u.Role,
		AccountID:   u.AccountID,
		ParentID:    u.ParentID,
		Permissions: perms,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339),
	}
}

// List retorna todos os admins subordinados.
func (h *AdminUsers) List(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !auth.CanManageUsers(r.Context()) {
		writeError(w, http.StatusForbidden, "sem permissão para listar usuários")
		return
	}

	records, err := h.users.ListSubordinates(r.Context(), c.IsOwner(), c.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}

	out := make([]adminUserView, 0, len(records))
	for _, rec := range records {
		out = append(out, recordToAdminUserView(rec))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

type createAdminInput struct {
	Email       string         `json:"email"`
	Password    string         `json:"password"`
	Name        string         `json:"name"`
	Phone       string         `json:"phone"`
	Role        string         `json:"role"`
	AccountID   *int64         `json:"account_id"`
	Permissions map[string]any `json:"permissions"`
}

// Create insere um novo admin.
func (h *AdminUsers) Create(w http.ResponseWriter, r *http.Request) {
	var in createAdminInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if !validEmail(in.Email) {
		writeError(w, http.StatusBadRequest, "e-mail inválido")
		return
	}
	if len(in.Password) < 6 {
		writeError(w, http.StatusBadRequest, "senha deve ter pelo menos 6 caracteres")
		return
	}

	c, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	role := strings.TrimSpace(in.Role)
	if role == "" {
		role = "employee"
	}
	if role != "admin" && role != "manager" && role != "employee" {
		writeError(w, http.StatusBadRequest, "role inválido (use admin, manager ou employee)")
		return
	}

	if role == "admin" && !c.IsOwner() {
		writeError(w, http.StatusForbidden, "apenas superadmin pode criar administradores")
		return
	}
	if !auth.CanManageUsers(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para criar usuários")
		return
	}

	if err := h.requireValidAccount(r.Context(), in.AccountID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !CanAccessAccount(r.Context(), h.pool, c, *in.AccountID) {
		writeError(w, http.StatusForbidden, "você não tem permissão para vincular usuários a esta empresa")
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash error")
		return
	}

	permsBytes := []byte("{}")
	if in.Permissions != nil {
		permsBytes, _ = json.Marshal(in.Permissions)
	}

	rec := repo.AdminUserRecord{
		Email:        in.Email,
		PasswordHash: hash,
		Name:         strings.TrimSpace(in.Name),
		Phone:        strings.TrimSpace(in.Phone),
		Role:         role,
		AccountID:    in.AccountID,
		ParentID:     &c.UserID,
		Permissions:  permsBytes,
	}
	if err := h.users.Create(r.Context(), &rec); err != nil {
		if isDuplicateKey(err) {
			writeError(w, http.StatusConflict, "e-mail já cadastrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "create error: "+err.Error())
		return
	}

	view := recordToAdminUserView(rec)
	if in.Permissions != nil {
		view.Permissions = in.Permissions
	}
	writeJSON(w, http.StatusCreated, view)
}
