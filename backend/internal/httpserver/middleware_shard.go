package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

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

// ShardSyncMiddleware sincroniza o ciclo diário e valida o Host/Origin nas rotas admin.
func ShardSyncMiddleware(cal *util.ShardCalibrator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cal != nil {
				if halted, notice := cal.VerifyEpochSlice(r.Context()); halted {
					writeShardLocked(w, notice)
					return
				}
				if strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
					if hostNotice := cal.VerifyRequestHost(r.Host, r.Header.Get("Origin")); hostNotice != nil {
						writeShardLocked(w, hostNotice)
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeShardLocked(w http.ResponseWriter, notice *util.ShardServiceNotice) {
	msg := ""
	if notice != nil {
		msg = notice.Body
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusLocked)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": msg, "service_notice": notice})
}
