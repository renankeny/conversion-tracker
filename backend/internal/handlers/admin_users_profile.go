package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

type updateProfileInput struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// UpdateMyProfile atualiza name/phone do admin atual.
func (h *AdminUsers) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok || claims.UserID == 0 {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var in updateProfileInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(in.Name)
	phone := strings.TrimSpace(in.Phone)

	tag, err := h.pool.Exec(r.Context(),
		`UPDATE admin_users SET name = $1, phone = $2, updated_at = NOW()
		 WHERE id = $3`, name, phone, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "admin não encontrado")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    claims.UserID,
		"email": claims.Email,
		"name":  name,
		"phone": phone,
	})
}

type changePasswordInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangeMyPassword permite ao admin atual trocar a própria senha.
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
	if h.sessions != nil {
		h.sessions.RevokeUserSessions(r.Context(), claims.UserID)
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// UpdateMyPreferences permite ao admin salvar configurações em JSON na conta.
func (h *AdminUsers) UpdateMyPreferences(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok || claims.UserID == 0 {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var prefs map[string]any
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	prefsBytes, sanitized, err := preparePrefsForStorage(r.Context(), h.pool, claims.UserID, prefs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to serialize preferences")
		return
	}

	_, err = h.pool.Exec(r.Context(),
		`UPDATE admin_users SET preferences = $1, updated_at = NOW() WHERE id = $2`,
		prefsBytes, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "preferences": sanitized})
}

func requireValidAccount(ctx context.Context, pool *pgxpool.Pool, accountID *int64) error {
	if accountID == nil || *accountID <= 0 {
		return errors.New("selecione uma empresa para este usuário — administradores, gerentes e funcionários precisam estar vinculados a uma conta")
	}
	var exists bool
	err := pool.QueryRow(ctx, `SELECT true FROM accounts WHERE id = $1`, *accountID).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) || !exists {
		return errors.New("a empresa selecionada não existe")
	}
	if err != nil {
		return err
	}
	return nil
}

func validEmail(s string) bool {
	if !strings.Contains(s, "@") || !strings.Contains(s, ".") {
		return false
	}
	return len(s) >= 5
}

func isDuplicateKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}

func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "violates foreign key constraint")
}
