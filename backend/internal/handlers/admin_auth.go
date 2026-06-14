package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

type AdminAuth struct {
	pool *pgxpool.Pool
	jwt  *auth.JWT
}

func NewAdminAuth(p *pgxpool.Pool, j *auth.JWT) *AdminAuth {
	return &AdminAuth{pool: p, jwt: j}
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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
		accountID *int64
	)
	err := h.pool.QueryRow(r.Context(),
		`SELECT id, password_hash, COALESCE(role, 'owner'), account_id
		   FROM admin_users WHERE email = $1`, req.Email,
	).Scan(&id, &hash, &role, &accountID)
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
	tok, err := h.jwt.Issue(id, req.Email, role, accID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tok,
		"user": map[string]any{
			"id":         id,
			"email":      req.Email,
			"role":       role,
			"account_id": accID,
		},
	})
}

func (h *AdminAuth) Me(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         c.UserID,
		"email":      c.Email,
		"role":       c.Role,
		"account_id": c.AccountID,
	})
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
		`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2)`, email, hash,
	)
	return err
}
