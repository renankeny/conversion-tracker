package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Health agrupa o handler do /health. Faz check sincrono no Postgres
// e Redis e devolve 503 se algum falhar — formato compatível com
// liveness/readiness probes (Kubernetes, Docker healthcheck, etc.).
type Health struct {
	pool      *pgxpool.Pool
	redisOpt  asynq.RedisClientOpt
	startedAt time.Time
}

func NewHealth(pool *pgxpool.Pool, redisOpt asynq.RedisClientOpt) *Health {
	return &Health{pool: pool, redisOpt: redisOpt, startedAt: time.Now()}
}

func (h *Health) Handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{}
	overall := "ok"

	// Postgres ping
	if err := h.pool.Ping(ctx); err != nil {
		checks["postgres"] = "fail: " + err.Error()
		overall = "degraded"
	} else {
		checks["postgres"] = "ok"
	}

	// Redis ping (via asynq inspector)
	insp := asynq.NewInspector(h.redisOpt)
	defer insp.Close()
	if _, err := insp.Queues(); err != nil {
		checks["redis"] = "fail: " + err.Error()
		overall = "degraded"
	} else {
		checks["redis"] = "ok"
	}

	body := map[string]any{
		"status":     overall,
		"checks":     checks,
		"uptime_sec": int(time.Since(h.startedAt).Seconds()),
		"now":        time.Now().Format(time.RFC3339),
	}
	status := http.StatusOK
	if overall != "ok" {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, body)
}
