package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
)

// AdminUsers gerencia o CRUD de admin_users (a tabela de admins do painel).
type AdminUsers struct {
	pool     *pgxpool.Pool
	sessions *cache.SessionStore
}

func NewAdminUsers(pool *pgxpool.Pool) *AdminUsers {
	return &AdminUsers{pool: pool}
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

	query := `SELECT id, email, COALESCE(name, ''), COALESCE(phone, ''),
		        COALESCE(role, 'admin'), account_id, parent_id, permissions, created_at, updated_at
		   FROM admin_users WHERE $1::boolean OR parent_id = $2 OR id = $2 ORDER BY id ASC`

	rows, err := h.pool.Query(r.Context(), query, c.IsOwner(), c.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}
	defer rows.Close()

	var out []adminUserView
	for rows.Next() {
		var v adminUserView
		var created, updated time.Time
		var permsBytes []byte
		if err := rows.Scan(&v.ID, &v.Email, &v.Name, &v.Phone, &v.Role, &v.AccountID, &v.ParentID, &permsBytes, &created, &updated); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error: "+err.Error())
			return
		}
		if len(permsBytes) > 0 {
			_ = json.Unmarshal(permsBytes, &v.Permissions)
		}
		if v.Permissions == nil {
			v.Permissions = make(map[string]any)
		}
		v.CreatedAt = created.Format(time.RFC3339)
		v.UpdatedAt = updated.Format(time.RFC3339)
		out = append(out, v)
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

	isSuper := c.IsOwner()
	if role == "admin" && !isSuper {
		writeError(w, http.StatusForbidden, "apenas superadmin pode criar administradores")
		return
	}
	if !auth.CanManageUsers(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para criar usuários")
		return
	}

	if err := requireValidAccount(r.Context(), h.pool, in.AccountID); err != nil {
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

	var permsBytes []byte
	if in.Permissions != nil {
		permsBytes, _ = json.Marshal(in.Permissions)
	} else {
		permsBytes = []byte("{}")
	}

	var id int64
	var created, updated time.Time
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO admin_users (email, password_hash, name, phone, role, account_id, parent_id, permissions)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at, updated_at`,
		in.Email, hash, strings.TrimSpace(in.Name), strings.TrimSpace(in.Phone), role, in.AccountID, c.UserID, permsBytes,
	).Scan(&id, &created, &updated)
	if err != nil {
		if isDuplicateKey(err) {
			writeError(w, http.StatusConflict, "e-mail já cadastrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "create error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, adminUserView{
		ID:          id,
		Email:       in.Email,
		Name:        strings.TrimSpace(in.Name),
		Phone:       strings.TrimSpace(in.Phone),
		Role:        role,
		AccountID:   in.AccountID,
		ParentID:    &c.UserID,
		Permissions: in.Permissions,
		CreatedAt:   created.Format(time.RFC3339),
		UpdatedAt:   updated.Format(time.RFC3339),
	})
}
