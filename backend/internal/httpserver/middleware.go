package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// tickerTouchTimeout é o timeout do UPDATE last_used_at em background. Curto
// porque um update bloqueado nunca pode atrapalhar a request original.
const tickerTouchTimeout = 5 * time.Second

// AuthMiddleware aceita autenticação por JWT (admin do painel) OU por API key
// (server-to-server, formato `trk_<base64>`).
//
// Quando uma API key é usada, sintetizamos Claims com role="member" + o
// account_id da key — o token age como um member da conta dona da key, sem
// poder gerenciar outras contas nem criar admins.
//
// API keys são opcionais: se apiKeys for nil, o middleware comporta-se como
// JWT-only (compat com instâncias que não rodaram a migration 0007).
func AuthMiddleware(j *auth.JWT, apiKeys *repo.APIKeyRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			tok := strings.TrimPrefix(h, "Bearer ")

			// Caminho 1: API key plaintext (`trk_*`). Olha primeiro porque a
			// validação é mais barata (1 query indexada por hash).
			if apiKeys != nil && strings.HasPrefix(tok, repo.PlaintextPrefix) {
				k, err := apiKeys.FindActiveByPlaintext(r.Context(), tok)
				if err != nil {
					http.Error(w, "invalid api key", http.StatusUnauthorized)
					return
				}
				// Atualiza last_used_at em background (best-effort).
				go func(id int64) {
					ctx, cancel := context.WithTimeout(context.Background(), tickerTouchTimeout)
					defer cancel()
					if err := apiKeys.TouchLastUsed(ctx, id); err != nil {
						slog.Warn("api_key touch last_used failed", "id", id, "err", err)
					}
				}(k.ID)

				claims := &auth.Claims{
					UserID:    0, // não há usuário associado
					Email:     "api-key:" + k.KeyPrefix,
					Role:      "member",
					AccountID: k.AccountID,
				}
				next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
				return
			}

			// Caminho 2: JWT clássico do painel.
			claims, err := j.Parse(tok)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
		})
	}
}

// JWTMiddleware é mantido por compat — equivalente a AuthMiddleware sem suporte
// a API key. Use AuthMiddleware nos novos pontos.
func JWTMiddleware(j *auth.JWT) func(http.Handler) http.Handler {
	return AuthMiddleware(j, nil)
}
