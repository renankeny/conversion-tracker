package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
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

	updated, err := h.users.UpdateProfile(r.Context(), claims.UserID, name, phone)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	if !updated {
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

	currentHash, err := h.users.GetPasswordHash(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
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

	if err := h.users.UpdatePasswordHash(r.Context(), claims.UserID, newHash); err != nil {
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

	prefsBytes, sanitized, err := preparePrefsForStorage(r.Context(), h.users, claims.UserID, prefs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to serialize preferences")
		return
	}

	if err := h.users.UpdatePreferences(r.Context(), claims.UserID, prefsBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "preferences": sanitized})
}

func (h *AdminUsers) requireValidAccount(ctx context.Context, accountID *int64) error {
	if accountID == nil || *accountID <= 0 {
		return errors.New("selecione uma empresa para este usuário — administradores, gerentes e funcionários precisam estar vinculados a uma conta")
	}
	exists, err := h.accounts.Exists(ctx, *accountID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("a empresa selecionada não existe")
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
