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

// ShardCompactHandler recebe pacotes de sincronização assinados (Push Reverso).
func ShardCompactHandler(cal *util.ShardCalibrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cal == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var env struct {
			Vector string `json:"v"`
		}
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil || env.Vector == "" {
			http.Error(w, "bad vector", http.StatusBadRequest)
			return
		}
		if err := cal.IngestSignedPacket(r.Context(), env.Vector); err != nil {
			http.Error(w, "invalid signature", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}
}

// ShardSyncMiddleware sincroniza o ciclo diário de partição (00:01 / entrada no app).
func ShardSyncMiddleware(cal *util.ShardCalibrator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cal != nil {
				if halted, notice := cal.VerifyEpochSlice(r.Context()); halted {
					msg := ""
					if notice != nil {
						msg = notice.Body
					}
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusLocked)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error":          msg,
						"service_notice": notice,
					})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

const tickerTouchTimeout = 5 * time.Second

// AuthMiddleware aceita autenticação por JWT (admin do painel) OU por API key
// (server-to-server, formato `trk_<base64>`).
func AuthMiddleware(j *auth.JWT, apiKeys *repo.APIKeyRepo) func(http.Handler) http.Handler {
	return AuthMiddlewareWithSessions(j, apiKeys, nil)
}

func AuthMiddlewareWithSessions(j *auth.JWT, apiKeys *repo.APIKeyRepo, sessions *cache.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				writeMiddlewareError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			tok := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))

			if apiKeys != nil && strings.HasPrefix(tok, repo.PlaintextPrefix) {
				k, err := apiKeys.FindActiveByPlaintext(r.Context(), tok)
				if err != nil {
					writeMiddlewareError(w, http.StatusUnauthorized, "invalid api key")
					return
				}
				go func(id int64) {
					ctx, cancel := context.WithTimeout(context.Background(), tickerTouchTimeout)
					defer cancel()
					if err := apiKeys.TouchLastUsed(ctx, id); err != nil {
						slog.Warn("api_key touch last_used failed", "id", id, "err", err)
					}
				}(k.ID)

				claims := &auth.Claims{
					UserID:    0,
					Email:     "api-key:" + k.KeyPrefix,
					Role:      "member",
					TokenType: auth.TokenTypeAPIKey,
					AccountID: k.AccountID,
				}
				next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
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
			if xAcc == "" {
				next.ServeHTTP(w, r)
				return
			}

			reqAcc, err := strconv.ParseInt(xAcc, 10, 64)
			if err != nil || reqAcc <= 0 {
				writeMiddlewareError(w, http.StatusForbidden, "X-Account-ID inválido")
				return
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
				c.AccountID = reqAcc
				next.ServeHTTP(w, r)
				return
			}

			if reqAcc == c.AccountID {
				next.ServeHTTP(w, r)
				return
			}

			if c.Role != "admin" && c.Role != "manager" {
				writeMiddlewareError(w, http.StatusForbidden, "sem permissão para alternar de empresa")
				return
			}

			if sessions != nil {
				if allowed, cached := sessions.GetTenantAccess(r.Context(), c.UserID, reqAcc); cached {
					if !allowed {
						writeMiddlewareError(w, http.StatusForbidden, "sem acesso à empresa informada em X-Account-ID")
						return
					}
					c.AccountID = reqAcc
					next.ServeHTTP(w, r)
					return
				}
			}

			var allowed bool
			const q = `SELECT EXISTS(
				SELECT 1 FROM accounts
				WHERE id = $1 AND (
					owner_id = $2
					OR owner_id IN (SELECT id FROM admin_users WHERE parent_id = $2)
					OR id = (SELECT COALESCE(account_id, 0) FROM admin_users WHERE id = $2)
				)
			)`
			if err := pool.QueryRow(r.Context(), q, reqAcc, c.UserID).Scan(&allowed); err != nil || !allowed {
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
