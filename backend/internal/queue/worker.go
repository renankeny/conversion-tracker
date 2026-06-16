package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/tiktok"
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
	TikTok   *tiktok.Client
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

// dispatchResult uniformiza resultados Meta e TikTok pra persistencia.
type dispatchResult struct {
	Channel     string // "meta" ou "tiktok"
	PixelID     string
	StatusCode  int
	Body        string
	RequestBody string
	Err         error
}

func fromMetaResults(rs []meta.SendResult) []dispatchResult {
	out := make([]dispatchResult, len(rs))
	for i, r := range rs {
		out[i] = dispatchResult{
			Channel:     "meta",
			PixelID:     r.PixelID,
			StatusCode:  r.StatusCode,
			Body:        r.Body,
			RequestBody: r.RequestBody,
			Err:         r.Err,
		}
	}
	return out
}

func fromTikTokResults(rs []tiktok.SendResult) []dispatchResult {
	out := make([]dispatchResult, len(rs))
	for i, r := range rs {
		out[i] = dispatchResult{
			Channel:     "tiktok",
			PixelID:     r.PixelID,
			StatusCode:  r.StatusCode,
			Body:        r.Body,
			RequestBody: r.RequestBody,
			Err:         r.Err,
		}
	}
	return out
}

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

		// Despacha em paralelo pra Meta e TikTok (quando configurados).
		// Wait pra ambos antes de processar resultados.
		var (
			metaResults   []meta.SendResult
			tiktokResults []tiktok.SendResult
			wg            sync.WaitGroup
		)
		if len(pixel.PixelIDs) > 0 && deps.Meta != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				metaResults = deps.Meta.Send(ctx, pixel, in)
			}()
		}
		if pixel.HasTikTok() && deps.TikTok != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tiktokResults = deps.TikTok.Send(ctx, pixel, in)
			}()
		}
		wg.Wait()

		results := append(
			fromMetaResults(metaResults),
			fromTikTokResults(tiktokResults)...,
		)

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
				slog.Error("send failed",
					"channel", r.Channel,
					"pixel_id", r.PixelID,
					"event", in.EventName,
					"event_id", in.EventID,
					"err", r.Err,
				)
			} else {
				delta.Sent = 1
				slog.Info("send ok",
					"channel", r.Channel,
					"pixel_id", r.PixelID,
					"event", in.EventName,
					"event_id", in.EventID,
					"status", r.StatusCode,
				)
			}

			// Counter agregado: mesmo event_name pra Meta e TikTok (é o mesmo
			// evento de negocio). O canal e diferenciado em sent_events via
			// coluna `channel`.
			if deps.Counters != nil {
				if cerr := deps.Counters.Increment(ctx, pixel.AccountID, pixel.Name, in.EventName, eventTime, delta); cerr != nil {
					slog.Warn("counter increment failed",
						"channel", r.Channel,
						"pixel", pixel.Name,
						"event", in.EventName,
						"err", cerr,
					)
				}
			}

			// Payload completo apenas para eventos importantes ou falhas.
			if storeFullPayload || r.Err != nil {
				persistResult(ctx, deps.Pool, pixel.AccountID, in, r)
			}
		}
		if hasErr {
			return fmt.Errorf("at least one channel/pixel failed")
		}
		return nil
	}
}

func persistResult(ctx context.Context, pool *pgxpool.Pool, accountID int64, in domain.SendEventInput, r dispatchResult) {
	if accountID == 0 {
		accountID = 1
	}
	status := "sent"
	errMsg := ""
	if r.Err != nil {
		status = "failed"
		errMsg = util.RedactSecrets(r.Err.Error())
	}
	payload := util.RedactSecrets(r.RequestBody)
	if payload == "" {
		payload = "null"
	}
	channel := r.Channel
	if channel == "" {
		channel = "meta"
	}
	_, _ = pool.Exec(ctx, `
		INSERT INTO sent_events
			(event_id, event_name, pixel_name, external_id, account_id,
			 channel, status, response_code, response_body, error, payload)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb)
	`,
		in.EventID, in.EventName, in.PixelName, in.ExternalID, accountID,
		channel, status, r.StatusCode, util.RedactSecrets(r.Body), errMsg, payload,
	)
}
