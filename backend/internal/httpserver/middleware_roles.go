package httpserver

import (
	"net/http"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

// RequireManager restringe operações destrutivas/administrativas da conta
// (prune, retenção, reset) a owner/admin/manager. Employees e chaves S2S recebem 403.
func RequireManager(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.CanManageUsers(r.Context()) {
			writeMiddlewareError(w, http.StatusForbidden, "sem permissão para manutenção de dados")
			return
		}
		next.ServeHTTP(w, r)
	})
}
