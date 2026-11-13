package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/ga4"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
	"github.com/renankeny/conversion-tracker/backend/internal/tiktok"
)

type Server struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

type Deps struct {
	Pool               *pgxpool.Pool
	Pixels             *repo.PixelRepo
	Users              *repo.UserRepo
	GoogleAdsAcc       *repo.GoogleAdsAccountRepo
	MetaAcc            *repo.MetaAccountRepo
	WhatsApp           *repo.WhatsAppRepo
	CRM                *repo.CRMRepo
	Storage            *storage.MediaStorage
	Enqueuer           *Enqueuer
	Meta               *meta.Client
	MetaOAuth          *metaoauth.Client
	TikTok             *tiktok.Client
	GoogleAds          *googleads.Client
	GA4                *ga4.Client
	Counters           *repo.EventCounterRepo
	SentEvents         *repo.SentEventRepo
	Redis              *redis.Client
	PublicAPIURL       string
	JWTSecret          string
	MediaRetentionDays int
}

func NewServer(opt asynq.RedisClientOpt, deps Deps, concurrency int) *Server {
	if concurrency <= 0 {
		concurrency = 10
	}
	srv := asynq.NewServer(
		opt,
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				QueueCritical:       6,
				QueueWhatsAppEvents: 4,
				QueueDefault:        2,
				QueueWhatsAppMedia:  1,
			},
		},
	)
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeSendEvent, makeSendEventHandler(deps))
	mux.HandleFunc(TypeProcessAggregatedPurchase, makeProcessAggregatedPurchaseHandler(deps))
	mux.HandleFunc(TypeWhatsAppEvent, makeWhatsAppEventHandler(deps))
	mux.HandleFunc(TypeWhatsAppMedia, makeWhatsAppMediaHandler(deps))
	return &Server{srv: srv, mux: mux}
}

func (s *Server) Run() error { return s.srv.Run(s.mux) }
func (s *Server) Stop()      { s.srv.Stop() }
func (s *Server) Shutdown()  { s.srv.Shutdown() }

type dispatchResult struct {
	Channel     string
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

func fromGoogleAdsResults(rs []googleads.SendResult) []dispatchResult {
	out := make([]dispatchResult, len(rs))
	for i, r := range rs {
		targetID := r.CustomerID
		if r.ActionID != "" {
			targetID = r.CustomerID + ":" + r.ActionID
		}
		out[i] = dispatchResult{
			Channel:     "google_ads",
			PixelID:     targetID,
			StatusCode:  r.StatusCode,
			Body:        r.Body,
			RequestBody: r.RequestBody,
			Err:         r.Err,
		}
	}
	return out
}

func fromGA4Result(r ga4.SendResult, measurementID string) []dispatchResult {
	if r.Err == ga4.ErrEventNotMapped {
		return nil
	}
	if r.Err != nil && r.StatusCode == 0 {
		return []dispatchResult{{Channel: "ga4", PixelID: measurementID, Err: r.Err}}
	}
	return []dispatchResult{{
		Channel:     "ga4",
		PixelID:     measurementID,
		StatusCode:  r.StatusCode,
		Body:        r.Body,
		RequestBody: r.RequestBody,
		Err:         r.Err,
	}}
}

func makeSendEventHandler(deps Deps) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var in domain.SendEventInput
		if err := json.Unmarshal(t.Payload(), &in); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		return dispatchSendEventInput(ctx, deps, in)
	}
}
