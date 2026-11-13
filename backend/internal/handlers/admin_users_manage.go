package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
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

	targetRole, targetParentID, err := h.users.GetRoleAndParent(r.Context(), id)
	if err != nil {
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

	if err := h.requireValidAccount(r.Context(), in.AccountID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !CanAccessAccount(r.Context(), h.pool, c, *in.AccountID) {
		writeError(w, http.StatusForbidden, "você não tem permissão para vincular usuários a esta empresa")
		return
	}

	permsBytes := []byte("{}")
	if in.Permissions != nil {
		permsBytes, _ = json.Marshal(in.Permissions)
	}

	var passwordHash string
	if in.Password != "" {
		if len(in.Password) < 6 {
			writeError(w, http.StatusBadRequest, "senha deve ter pelo menos 6 caracteres")
			return
		}
		passwordHash, err = auth.HashPassword(in.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "hash error")
			return
		}
	}

	if err := h.users.Update(r.Context(), id, in.Email, in.Name, in.Phone, role, in.AccountID, permsBytes, passwordHash); err != nil {
		if isDuplicateKey(err) {
			writeError(w, http.StatusConflict, "e-mail já cadastrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error: "+err.Error())
		return
	}

	if in.Password != "" || role != targetRole {
		if h.sessions != nil {
			h.sessions.RevokeUserSessions(r.Context(), id)
		}
	}

	updatedRec, err := h.users.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error fetching updated user")
		return
	}
	writeJSON(w, http.StatusOK, recordToAdminUserView(*updatedRec))
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

	targetRole, targetParentID, err := h.users.GetRoleAndParent(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "usuário não encontrado")
		return
	}

	if !c.IsOwner() {
		if targetRole == "superadmin" || targetRole == "owner" || targetRole == "admin" {
			writeError(w, http.StatusForbidden, "você não tem permissão para excluir este usuário")
			return
		}
		if targetParentID == nil || *targetParentID != c.UserID {
			writeError(w, http.StatusForbidden, "você só pode excluir usuários criados por você")
			return
		}
	}

	total, err := h.users.Count(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "count error")
		return
	}
	if total <= 1 {
		writeError(w, http.StatusBadRequest, "não é possível remover o último admin")
		return
	}

	deleted, err := h.users.Delete(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "delete error: "+err.Error())
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "admin não encontrado")
		return
	}
	if h.sessions != nil {
		h.sessions.RevokeUserSessions(r.Context(), id)
	}
	w.WriteHeader(http.StatusNoContent)
}
