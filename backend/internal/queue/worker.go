package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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
	Pool   *pgxpool.Pool
	Pixels *repo.PixelRepo
	Meta   *meta.Client
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

func (s *Server) Run() error  { return s.srv.Run(s.mux) }
func (s *Server) Stop()        { s.srv.Stop() }
func (s *Server) Shutdown()    { s.srv.Shutdown() }

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

		results := deps.Meta.Send(ctx, pixel, in)
		hasErr := false
		for _, r := range results {
			persistResult(ctx, deps.Pool, in, r)
			if r.Err != nil {
				hasErr = true
				slog.Error("meta send failed",
					"pixel_id", r.PixelID,
					"event", in.EventName,
					"event_id", in.EventID,
					"err", r.Err,
				)
			} else {
				slog.Info("meta send ok",
					"pixel_id", r.PixelID,
					"event", in.EventName,
					"event_id", in.EventID,
					"status", r.StatusCode,
				)
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
