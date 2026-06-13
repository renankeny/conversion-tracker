package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type Server struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

type Deps struct {
	Pool     *pgxpool.Pool
	Pixels   *repo.PixelRepo
	Meta     *meta.Client
	Counters *repo.EventCounterRepo
}

// NewServer aceita opções completas de Redis (auth/TLS/DB).
// Use queue.BuildRedisClientOpt(cfg) pra montar a partir do config.
func NewServer(opt asynq.RedisClientOpt, deps Deps) *Server {
	srv := asynq.NewServer(
		opt,
		asynq.Config{
			Concurrency: 10,
			Queues:      map[string]int{QueueDefault: 10},
		},
	)
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeSendEvent, makeSendEventHandler(deps))
	return &Server{srv: srv, mux: mux}
}

func (s *Server) Run() error { return s.srv.Run(s.mux) }
func (s *Server) Stop()      { s.srv.Stop() }
func (s *Server) Shutdown()  { s.srv.Shutdown() }

func makeSendEventHandler(deps Deps) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var in domain.SendEventInput
		if err := json.Unmarshal(t.Payload(), &in); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}

		pixel, err := deps.Pixels.FindByName(ctx, in.PixelName)
		if err != nil {
			return fmt.Errorf("pixel %q: %w", in.PixelName, err)
		}
		if !pixel.Enabled {
			slog.Warn("pixel disabled, skipping", "pixel", pixel.Name)
			return nil
		}

		// Estratégia dual:
		//   - Sempre incrementa o counter agregado (1 row/hora por par pixel+evento)
		//   - Se for evento "importante" (Lead, InitiateCheckout, Purchase, ...),
		//     ADICIONALMENTE persiste o resultado completo em sent_events.
		results := deps.Meta.Send(ctx, pixel, in)
		hasErr := false

		eventTime := time.Unix(in.EventTime, 0)
		if in.EventTime == 0 {
			eventTime = time.Now()
		}
		storeFullPayload := domain.ShouldStoreFullPayload(in.EventName)

		for _, r := range results {
			delta := domain.CounterDelta{Total: 1}
			if r.Err != nil {
				delta.Failed = 1
				hasErr = true
				slog.Error("meta send failed",
					"pixel_id", r.PixelID,
					"event", in.EventName,
					"event_id", in.EventID,
					"err", r.Err,
				)
			} else {
				delta.Sent = 1
				slog.Info("meta send ok",
					"pixel_id", r.PixelID,
					"event", in.EventName,
					"event_id", in.EventID,
					"status", r.StatusCode,
				)
			}

			// Counter agregado (sempre).
			if deps.Counters != nil {
				if cerr := deps.Counters.Increment(ctx, pixel.Name, in.EventName, eventTime, delta); cerr != nil {
					slog.Warn("counter increment failed",
						"pixel", pixel.Name,
						"event", in.EventName,
						"err", cerr,
					)
				}
			}

			// Payload completo apenas para eventos importantes ou falhas.
			// Falhas SEMPRE são persistidas (independente do tipo) para auditoria.
			if storeFullPayload || r.Err != nil {
				persistResult(ctx, deps.Pool, in, r)
			}
		}
		if hasErr {
			return fmt.Errorf("at least one pixel failed")
		}
		return nil
	}
}

func persistResult(ctx context.Context, pool *pgxpool.Pool, in domain.SendEventInput, r meta.SendResult) {
	status := "sent"
	errMsg := ""
	if r.Err != nil {
		status = "failed"
		errMsg = util.RedactSecrets(r.Err.Error())
	}
	_, _ = pool.Exec(ctx, `
		INSERT INTO sent_events (event_id, event_name, pixel_name, external_id, status, response_code, response_body, error)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, in.EventID, in.EventName, in.PixelName, in.ExternalID, status, r.StatusCode, util.RedactSecrets(r.Body), errMsg)
}
