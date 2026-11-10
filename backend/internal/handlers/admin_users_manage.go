package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

type updateAdminInput struct {
	Email       string         `json:"email"`
	Password    string         `json:"password"`
	Name        string         `json:"name"`
	Phone       string         `json:"phone"`
	Role        string         `json:"role"`
	AccountID   *int64         `json:"account_id"`
	Permissions map[string]any `json:"permissions"`
}

// Update altera dados de um admin existente.
func (h *AdminUsers) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var in updateAdminInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if !validEmail(in.Email) {
		writeError(w, http.StatusBadRequest, "e-mail inválido")
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
		writeError(w, http.StatusForbidden, "apenas superadmin pode criar/editar administradores")
		return
	}
	if !auth.CanManageUsers(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar usuários")
		return
	}

	// Verificar se usuário tem permissão sobre o alvo
	var targetRole string
	var targetParentID *int64
	if err := h.pool.QueryRow(r.Context(), `SELECT role, parent_id FROM admin_users WHERE id = $1`, id).Scan(&targetRole, &targetParentID); err != nil {
		writeError(w, http.StatusNotFound, "usuário não encontrado")
		return
	}

	if !isSuper {
		if targetRole == "superadmin" || targetRole == "owner" || targetRole == "admin" {
			writeError(w, http.StatusForbidden, "você não tem permissão para editar este usuário")
			return
		}
		if targetParentID == nil || *targetParentID != c.UserID {
			writeError(w, http.StatusForbidden, "você só pode editar usuários criados por você")
			return
		}
	}

	if err := requireValidAccount(r.Context(), h.pool, in.AccountID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !CanAccessAccount(r.Context(), h.pool, c, *in.AccountID) {
		writeError(w, http.StatusForbidden, "você não tem permissão para vincular usuários a esta empresa")
		return
	}

	var permsBytes []byte
	if in.Permissions != nil {
		permsBytes, _ = json.Marshal(in.Permissions)
	} else {
		permsBytes = []byte("{}")
	}

	if in.Password != "" {
		if len(in.Password) < 6 {
			writeError(w, http.StatusBadRequest, "senha deve ter pelo menos 6 caracteres")
			return
		}
		hash, err := auth.HashPassword(in.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "hash error")
			return
		}
		_, err = h.pool.Exec(r.Context(),
			`UPDATE admin_users SET email = $1, name = $2, phone = $3, role = $4, account_id = $5, permissions = $6, password_hash = $7, updated_at = NOW() WHERE id = $8`,
			in.Email, in.Name, in.Phone, role, in.AccountID, permsBytes, hash, id,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db error: "+err.Error())
			return
		}
	} else {
		_, err = h.pool.Exec(r.Context(),
			`UPDATE admin_users SET email = $1, name = $2, phone = $3, role = $4, account_id = $5, permissions = $6, updated_at = NOW() WHERE id = $7`,
			in.Email, in.Name, in.Phone, role, in.AccountID, permsBytes, id,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db error: "+err.Error())
			return
		}
	}

	if in.Password != "" || role != targetRole {
		if h.sessions != nil {
			h.sessions.RevokeUserSessions(r.Context(), id)
		}
	}

	var updatedUser adminUserView
	var created, updated time.Time
	err = h.pool.QueryRow(r.Context(),
		`SELECT id, email, name, phone, COALESCE(role, 'admin'), account_id, parent_id, permissions, created_at, updated_at FROM admin_users WHERE id = $1`,
		id,
	).Scan(&updatedUser.ID, &updatedUser.Email, &updatedUser.Name, &updatedUser.Phone, &updatedUser.Role, &updatedUser.AccountID, &updatedUser.ParentID, &permsBytes, &created, &updated)

	if err == nil {
		if len(permsBytes) > 0 {
			_ = json.Unmarshal(permsBytes, &updatedUser.Permissions)
		}
		updatedUser.CreatedAt = created.Format(time.RFC3339)
		updatedUser.UpdatedAt = updated.Format(time.RFC3339)
		json.NewEncoder(w).Encode(updatedUser)
	} else {
		writeError(w, http.StatusInternalServerError, "error fetching updated user")
	}
}

// Delete remove um admin. Requer body com {"confirm": true} pra dupla checagem.
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

	c, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	if !auth.CanManageUsers(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para excluir usuários")
		return
	}

	var targetRole string
	var targetParentID *int64
	if err := h.pool.QueryRow(r.Context(), `SELECT role, parent_id FROM admin_users WHERE id = $1`, id).Scan(&targetRole, &targetParentID); err != nil {
		writeError(w, http.StatusNotFound, "usuário não encontrado")
		return
	}

	isSuper := c.IsOwner()
	if !isSuper {
		if targetRole == "superadmin" || targetRole == "owner" || targetRole == "admin" {
			writeError(w, http.StatusForbidden, "você não tem permissão para excluir este usuário")
			return
		}
		if targetParentID == nil || *targetParentID != c.UserID {
			writeError(w, http.StatusForbidden, "você só pode excluir usuários criados por você")
			return
		}
	}

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
	if h.sessions != nil {
		h.sessions.RevokeUserSessions(r.Context(), id)
	}
	w.WriteHeader(http.StatusNoContent)
}
