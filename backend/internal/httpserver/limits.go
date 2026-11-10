package httpserver

import (
	"net/http"
)

// MaxBodyBytes limita o tamanho do body aceito pelas próximas leituras
// (uses http.MaxBytesReader). Aplique como middleware nas rotas públicas
// pra evitar request bombing.
//
// Tamanhos típicos:
//
//	/events/init   →   8 KB (body é minúsculo)
//	/events/collect →  64 KB (com PII e UTMs)
//	/webhooks/*    →   2 MB (alguns gateways mandam payload grande)
func MaxBodyBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}
