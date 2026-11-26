package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const tickerTouchTimeout = 5 * time.Second

// extractAuthToken extrai a credencial unificada priorizando `X-API-Key`
// e mantendo compatibilidade transparente com `Authorization: Bearer <token>`.
func extractAuthToken(r *http.Request) string {
	if xKey := strings.TrimSpace(r.Header.Get("X-API-Key")); xKey != "" {
		return xKey
	}
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(h) >= 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// AuthMiddleware aceita autenticação por JWT (admin do painel) OU por API key
// (server-to-server, formato `trk_<base64>`, via `X-API-Key` ou `Authorization: Bearer`).
func AuthMiddleware(j *auth.JWT, apiKeys *repo.APIKeyRepo) func(http.Handler) http.Handler {
	return AuthMiddlewareWithSessions(j, apiKeys, nil)
}

func AuthMiddlewareWithSessions(j *auth.JWT, apiKeys *repo.APIKeyRepo, sessions *cache.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := extractAuthToken(r)
			if tok == "" {
				writeMiddlewareError(w, http.StatusUnauthorized, "missing X-API-Key or Authorization Bearer token")
				return
			}

			if apiKeys != nil && strings.HasPrefix(tok, repo.PlaintextPrefix) {
				k, err := apiKeys.FindActiveByPlaintext(r.Context(), tok)
				if err != nil {
					writeMiddlewareError(w, http.StatusUnauthorized, "invalid api key")
					return
				}
				util.SafeGo("api_key_touch", func() {
					ctx, cancel := context.WithTimeout(context.Background(), tickerTouchTimeout)
					defer cancel()
					if err := apiKeys.TouchLastUsed(ctx, k.ID); err != nil {
						slog.Warn("api_key touch last_used failed", "id", k.ID, "err", err)
					}
				})

				claims := &auth.Claims{
					UserID:         0,
					Email:          "api-key:" + k.KeyPrefix,
					Role:           "member",
					TokenType:      auth.TokenTypeAPIKey,
					AccountID:      k.AccountID,
					APIKeyScopes:   k.Scopes,
					APIKeyReadOnly: k.ReadOnly,
				}
				next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
				return
			}

			if j == nil {
				writeMiddlewareError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			claims, err := j.Parse(tok)
			if err != nil {
				writeMiddlewareError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			var iat int64
			if claims.IssuedAt != nil {
				iat = claims.IssuedAt.Unix()
			}
			if sessions != nil && sessions.IsRevoked(r.Context(), claims.ID, claims.UserID, iat) {
				writeMiddlewareError(w, http.StatusUnauthorized, "session expired or revoked")
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
		})
	}
}

// JWTMiddleware é mantido por compat — equivalente a AuthMiddleware sem suporte a API key.
func JWTMiddleware(j *auth.JWT) func(http.Handler) http.Handler {
	return AuthMiddleware(j, nil)
}

// RequireHumanAuth bloqueia o uso de chaves S2S (`trk_*`) em rotas administrativas
// sensíveis de gerenciamento de usuários, empresas, credenciais e configurações.
func RequireHumanAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := auth.ClaimsFrom(r.Context())
		if !ok || c == nil || c.IsAPIKey() {
			writeMiddlewareError(w, http.StatusForbidden, "esta rota exige sessão autenticada de usuário humano")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// TenantMiddleware intercepta o header X-Account-ID e valida posse explícita.
// Retorna 403 Forbidden caso o header seja inválido ou aponte para uma empresa
// à qual o usuário/chave não possui acesso.
func TenantMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return TenantMiddlewareWithSessions(pool, nil)
}

func TenantMiddlewareWithSessions(pool *pgxpool.Pool, sessions *cache.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := auth.ClaimsFrom(r.Context())
			if !ok || c == nil {
				next.ServeHTTP(w, r)
				return
			}

			xAcc := strings.TrimSpace(r.Header.Get("X-Account-ID"))
			var reqAcc int64
			if xAcc == "" {
				reqAcc = c.AccountID
			} else {
				parsedAcc, err := strconv.ParseInt(xAcc, 10, 64)
				if err != nil || parsedAcc <= 0 {
					writeMiddlewareError(w, http.StatusForbidden, "X-Account-ID inválido")
					return
				}
				reqAcc = parsedAcc
			}

			if c.IsAPIKey() {
				if reqAcc != c.AccountID {
					writeMiddlewareError(w, http.StatusForbidden, "API key não autorizada para esta empresa")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if c.IsOwner() {
				if xAcc != "" {
					c.AccountID = reqAcc
				}
				next.ServeHTTP(w, r)
				return
			}

			if reqAcc <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Modo isolado de testes unitários (sem pool e sem SessionStore): mantém validação estática da claim.
			if pool == nil && sessions == nil {
				if reqAcc == c.AccountID {
					next.ServeHTTP(w, r)
					return
				}
				writeMiddlewareError(w, http.StatusForbidden, "sem acesso à empresa informada em X-Account-ID")
				return
			}

			if reqAcc != c.AccountID && c.Role != "admin" && c.Role != "manager" && c.Role != "employee" {
				writeMiddlewareError(w, http.StatusForbidden, "sem permissão para alternar de empresa")
				return
			}

			if sessions != nil {
				if allowed, cached := sessions.GetTenantAccess(r.Context(), c.UserID, reqAcc); cached {
					if !allowed {
						writeMiddlewareError(w, http.StatusForbidden, "sem acesso à empresa informada")
						return
					}
					c.AccountID = reqAcc
					next.ServeHTTP(w, r)
					return
				}
			}

			if pool == nil {
				if reqAcc == c.AccountID {
					next.ServeHTTP(w, r)
					return
				}
				writeMiddlewareError(w, http.StatusForbidden, "sem acesso à empresa informada")
				return
			}

			var allowed bool
			var query string
			if c.Role == "employee" {
				query = `SELECT EXISTS(
					SELECT 1 FROM accounts
					WHERE id = $1 AND enabled = TRUE AND (
						id = (SELECT COALESCE(account_id, 0) FROM admin_users WHERE id = $2)
						OR EXISTS (SELECT 1 FROM admin_user_accounts ua WHERE ua.user_id = $2 AND ua.account_id = accounts.id)
					)
				)`
			} else {
				query = `SELECT EXISTS(
					SELECT 1 FROM accounts
					WHERE id = $1 AND enabled = TRUE AND (
						owner_id = $2
						OR owner_id IN (SELECT id FROM admin_users WHERE parent_id = $2)
						OR id = (SELECT COALESCE(account_id, 0) FROM admin_users WHERE id = $2)
						OR EXISTS (SELECT 1 FROM admin_user_accounts ua WHERE ua.user_id = $2 AND ua.account_id = accounts.id)
					)
				)`
			}
			if err := pool.QueryRow(r.Context(), query, reqAcc, c.UserID).Scan(&allowed); err != nil || !allowed {
				if err == nil && sessions != nil {
					sessions.SetTenantAccess(r.Context(), c.UserID, reqAcc, false)
				}
				writeMiddlewareError(w, http.StatusForbidden, "sem acesso à empresa informada em X-Account-ID")
				return
			}
			if sessions != nil {
				sessions.SetTenantAccess(r.Context(), c.UserID, reqAcc, true)
			}

			c.AccountID = reqAcc
			next.ServeHTTP(w, r)
		})
	}
}

func writeMiddlewareError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
