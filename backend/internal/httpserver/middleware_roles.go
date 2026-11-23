package httpserver

import (
	"net/http"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

// RequireManager restringe operações destrutivas/administrativas da conta
// (prune, retenção, reset, exclusão em lote) a owner/admin/manager humanos.
func RequireManager(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.CanManageUsers(r.Context()) {
			writeMiddlewareError(w, http.StatusForbidden, "sem permissão para manutenção de dados")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireManagerOrAPIKey permite operações operacionais não-destrutivas (reenvio de eventos falhos,
// enriquecimento de dados de contato) para gestores humanos OU chaves S2S autorizadas.
func RequireManagerOrAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := auth.ClaimsFrom(r.Context())
		if ok && c != nil && c.IsAPIKey() && c.AccountID > 0 {
			next.ServeHTTP(w, r)
			return
		}
		if !auth.CanManageUsers(r.Context()) {
			writeMiddlewareError(w, http.StatusForbidden, "sem permissão para esta operação")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAPIScope valida que uma API Key (`trk_*`) possui permissão para o módulo da rota
// e respeita o modo somente leitura (`read_only`) em métodos mutáveis. Sessões humanas passam direto.
func RequireAPIScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := auth.ClaimsFrom(r.Context())
			if ok && c != nil && c.IsAPIKey() {
				isWrite := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
				if !c.HasAPIScope(scope, isWrite) {
					if isWrite && c.APIKeyReadOnly {
						writeMiddlewareError(w, http.StatusForbidden, "esta API key está em modo somente leitura")
						return
					}
					writeMiddlewareError(w, http.StatusForbidden, "esta API key não possui permissão para o módulo '"+scope+"'")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
