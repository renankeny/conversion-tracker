package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
)

type AdminAuth struct {
	pool      *pgxpool.Pool
	jwt       *auth.JWT
	sessions  *cache.SessionStore
	wsTickets *cache.WSTicketStore
}

func NewAdminAuth(p *pgxpool.Pool, j *auth.JWT) *AdminAuth {
	return &AdminAuth{pool: p, jwt: j}
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

	var (
		id        int64
		hash      string
		role      string
		accountID   *int64
		parentID    *int64
		name        string
		phone       string
		prefsBytes  []byte
		permsBytes  []byte
	)
	err := h.pool.QueryRow(r.Context(),
		`SELECT id, password_hash, COALESCE(role, 'superadmin'), account_id,
		        parent_id, permissions, COALESCE(name, ''), COALESCE(phone, ''), preferences
		   FROM admin_users WHERE email = $1`, req.Email,
	).Scan(&id, &hash, &role, &accountID, &parentID, &permsBytes, &name, &phone, &prefsBytes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := auth.CheckPassword(hash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	var accID int64
	if accountID != nil {
		accID = *accountID
	}
	var perms map[string]any
	if len(permsBytes) > 0 {
		_ = json.Unmarshal(permsBytes, &perms)
	}
	if perms == nil {
		perms = make(map[string]any)
	}

	duration := time.Duration(h.jwt.ExpHours()) * time.Hour
	if req.RememberMe {
		duration = 30 * 24 * time.Hour
	}
	tok, err := h.jwt.IssueWithDuration(id, req.Email, role, accID, parentID, perms, duration)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue error")
		return
	}
	prefs := sanitizePrefsForClient(prefsBytes)

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tok,
		"user": map[string]any{
			"id":          id,
			"email":       req.Email,
			"name":        name,
			"phone":       phone,
			"role":        role,
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

	// Carrega name/phone do banco — esses campos não vão pro JWT pra evitar
	// invalidar tokens quando o usuário só atualiza o perfil.
	var name, phone string
	var prefsBytes []byte
	if c.UserID > 0 {
		_ = h.pool.QueryRow(r.Context(),
			`SELECT COALESCE(name, ''), COALESCE(phone, ''), preferences FROM admin_users WHERE id = $1`,
			c.UserID,
		).Scan(&name, &phone, &prefsBytes)
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
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO admin_users (email, password_hash, role, account_id) VALUES ($1, $2, 'superadmin', 1)`, email, hash,
	)
	return err
}
