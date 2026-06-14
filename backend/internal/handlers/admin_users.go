package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

// AdminUsers gerencia o CRUD de admin_users (a tabela de admins do painel).
type AdminUsers struct {
	pool *pgxpool.Pool
}

func NewAdminUsers(pool *pgxpool.Pool) *AdminUsers {
	return &AdminUsers{pool: pool}
}

type adminUserView struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	AccountID *int64 `json:"account_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// List retorna todos os admins (sem hash de senha).
func (h *AdminUsers) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, email, COALESCE(role, 'owner'), account_id, created_at, updated_at
		   FROM admin_users ORDER BY id ASC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}
	defer rows.Close()

	var out []adminUserView
	for rows.Next() {
		var v adminUserView
		var created, updated time.Time
		if err := rows.Scan(&v.ID, &v.Email, &v.Role, &v.AccountID, &created, &updated); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error: "+err.Error())
			return
		}
		v.CreatedAt = created.Format(time.RFC3339)
		v.UpdatedAt = updated.Format(time.RFC3339)
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

type createAdminInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	AccountID *int64 `json:"account_id"`
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

	role := strings.TrimSpace(in.Role)
	if role == "" {
		role = "owner"
	}
	if role != "owner" && role != "member" {
		writeError(w, http.StatusBadRequest, "role inválido (use owner ou member)")
		return
	}
	if role == "member" && in.AccountID == nil {
		writeError(w, http.StatusBadRequest, "members precisam de account_id")
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash error")
		return
	}

	var id int64
	var created, updated time.Time
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO admin_users (email, password_hash, role, account_id)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`, in.Email, hash, role, in.AccountID,
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
		ID:        id,
		Email:     in.Email,
		Role:      role,
		AccountID: in.AccountID,
		CreatedAt: created.Format(time.RFC3339),
		UpdatedAt: updated.Format(time.RFC3339),
	})
}

// Delete remove um admin. Requer body com {"confirm": true} pra dupla checagem.
// Não permite deletar o último admin (anti-lockout).
func (h *AdminUsers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Confirm bool `json:"confirm"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if !body.Confirm {
		writeError(w, http.StatusConflict, "operação destrutiva: envie {\"confirm\": true}")
		return
	}

	// Conta total pra evitar deletar o último.
	var total int
	if err := h.pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM admin_users`).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "count error")
		return
	}
	if total <= 1 {
		writeError(w, http.StatusBadRequest, "não é possível remover o último admin")
		return
	}

	tag, err := h.pool.Exec(r.Context(),
		`DELETE FROM admin_users WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "delete error: "+err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "admin não encontrado")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type changePasswordInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangeMyPassword permite ao admin atual trocar a própria senha.
// Requer a senha atual pra evitar troca por sessão sequestrada.
func (h *AdminUsers) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var in changePasswordInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(in.NewPassword) < 6 {
		writeError(w, http.StatusBadRequest, "nova senha deve ter pelo menos 6 caracteres")
		return
	}

	var currentHash string
	err := h.pool.QueryRow(r.Context(),
		`SELECT password_hash FROM admin_users WHERE id = $1`, claims.UserID,
	).Scan(&currentHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "admin não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}

	if err := auth.CheckPassword(currentHash, in.CurrentPassword); err != nil {
		writeError(w, http.StatusUnauthorized, "senha atual incorreta")
		return
	}

	newHash, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash error")
		return
	}

	if _, err := h.pool.Exec(r.Context(),
		`UPDATE admin_users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		newHash, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func validEmail(s string) bool {
	if !strings.Contains(s, "@") || !strings.Contains(s, ".") {
		return false
	}
	if len(s) < 5 {
		return false
	}
	return true
}

func isDuplicateKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}
