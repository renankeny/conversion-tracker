package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminAuth struct {
	users     *repo.AdminUserRepo
	jwt       *auth.JWT
	sessions  *cache.SessionStore
	wsTickets *cache.WSTicketStore
}

func NewAdminAuth(p *pgxpool.Pool, j *auth.JWT) *AdminAuth {
	return &AdminAuth{users: repo.NewAdminUserRepo(p), jwt: j}
}

func (h *AdminAuth) SetSessionDeps(s *cache.SessionStore, wst *cache.WSTicketStore) {
	h.sessions = s
	h.wsTickets = wst
}

type loginReq struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

func (h *AdminAuth) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password required")
		return
	}

	u, err := h.users.FindByEmailForAuth(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := auth.CheckPassword(u.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	var accID int64
	if u.AccountID != nil {
		accID = *u.AccountID
	}
	var perms map[string]any
	if len(u.Permissions) > 0 {
		_ = json.Unmarshal(u.Permissions, &perms)
	}
	if perms == nil {
		perms = make(map[string]any)
	}

	duration := time.Duration(h.jwt.ExpHours()) * time.Hour
	if req.RememberMe {
		duration = 30 * 24 * time.Hour
	}
	tok, err := h.jwt.IssueWithDuration(u.ID, req.Email, u.Role, accID, u.ParentID, perms, duration)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue error")
		return
	}
	prefs := sanitizePrefsForClient(u.Preferences)

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tok,
		"user": map[string]any{
			"id":          u.ID,
			"email":       req.Email,
			"name":        u.Name,
			"phone":       u.Phone,
			"role":        u.Role,
			"account_id":  accID,
			"preferences": prefs,
		},
	})
}

func (h *AdminAuth) Me(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var name, phone string
	var prefsBytes []byte
	if c.UserID > 0 {
		name, phone, prefsBytes, _ = h.users.GetProfileAndPrefs(r.Context(), c.UserID)
	}

	prefs := sanitizePrefsForClient(prefsBytes)

	writeJSON(w, http.StatusOK, map[string]any{
		"id":          c.UserID,
		"email":       c.Email,
		"name":        name,
		"phone":       phone,
		"role":        c.Role,
		"account_id":  c.AccountID,
		"parent_id":   c.ParentID,
		"permissions": c.Permissions,
		"preferences": prefs,
	})
}

// Logout POST /api/v1/admin/logout — revoga imediatamente o JTI atual no Redis.
func (h *AdminAuth) Logout(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if ok && c != nil && c.ID != "" && h.sessions != nil {
		ttl := 24 * time.Hour
		if c.ExpiresAt != nil {
			if rem := time.Until(c.ExpiresAt.Time); rem > 0 {
				ttl = rem
			}
		}
		h.sessions.RevokeToken(r.Context(), c.ID, ttl)
	}
	writeJSON(w, http.StatusOK, map[string]any{"logged_out": true})
}

// IssueWSTicket POST /api/v1/admin/ws-ticket — emite ticket de uso único (30s)
// já escopado pela empresa ativa (X-Account-ID validado pelo TenantMiddleware).
func (h *AdminAuth) IssueWSTicket(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil || c.IsAPIKey() {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if h.wsTickets == nil {
		writeError(w, http.StatusServiceUnavailable, "ticket service unavailable")
		return
	}
	scope := TenantFromContext(r.Context())
	accID := scope.AccountID
	if accID == 0 && !scope.IsOwner {
		accID = c.AccountID
	}
	ticket, err := h.wsTickets.Issue(r.Context(), c.UserID, accID, c.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ticket issue error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticket": ticket, "expires_in": 30})
}

// EnsureBootstrapAdmin cria um admin padrão se não existir nenhum.
func EnsureBootstrapAdmin(ctx context.Context, pool *pgxpool.Pool, email, password string) error {
	if email == "" || password == "" {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return repo.NewAdminUserRepo(pool).EnsureBootstrap(ctx, email, hash)
}
