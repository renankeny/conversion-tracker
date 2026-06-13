package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

const (
	TypeSendEvent = "meta:send_event"
	QueueDefault  = "default"
)

type Enqueuer struct {
	client *asynq.Client
}

// NewEnqueuer aceita opções completas de Redis (auth/TLS/DB).
// Use queue.BuildRedisClientOpt(cfg) pra montar a partir do config.
func NewEnqueuer(opt asynq.RedisClientOpt) *Enqueuer {
	c := asynq.NewClient(opt)
	return &Enqueuer{client: c}
}

func (e *Enqueuer) Close() error { return e.client.Close() }

// EnqueueSendEvent agenda envio assíncrono pra Meta CAPI.
func (e *Enqueuer) EnqueueSendEvent(ctx context.Context, in domain.SendEventInput) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	task := asynq.NewTask(TypeSendEvent, payload)
	_, err = e.client.EnqueueContext(ctx, task,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(5),
	)
	return err
}
