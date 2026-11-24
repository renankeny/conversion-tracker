package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

const (
	TypeSendEvent                = "meta:send_event"
	TypeWhatsAppEvent            = "whatsapp:process_event"
	TypeWhatsAppMedia            = "whatsapp:process_media"
	TypeWhatsAppVerifyDisconnect = "whatsapp:verify_disconnect"

	QueueCritical       = "critical"
	QueueWhatsAppEvents = "whatsapp_events"
	QueueDefault        = "default"
	QueueWhatsAppMedia  = "whatsapp_media"
)

type WhatsAppEventPayload struct {
	AccountID  int64           `json:"account_id"`
	InstanceID int64           `json:"instance_id"`
	Event      string          `json:"event"`
	Data       json.RawMessage `json:"data"`
}

type WhatsAppVerifyDisconnectPayload struct {
	AccountID       int64  `json:"account_id"`
	InstanceID      int64  `json:"instance_id"`
	Event           string `json:"event"`
	TriggeredAtUnix int64  `json:"triggered_at_unix"`
}

type WhatsAppMediaPayload struct {
	AccountID        int64           `json:"account_id"`
	InstanceID       int64           `json:"instance_id"`
	ChatID           int64           `json:"chat_id"`
	MessageDBID      int64           `json:"message_db_id"`
	MessageType      string          `json:"message_type"`
	MimeType         string          `json:"mime_type"`
	ExternalMediaURL string          `json:"external_media_url,omitempty"`
	ExpectedBytes    int64           `json:"expected_bytes,omitempty"`
	RawMessageJSON   json.RawMessage `json:"raw_message_json"`
}

type Enqueuer struct {
	client        *asynq.Client
	sendHook      func(context.Context, domain.SendEventInput) error
	discCheckHook func(context.Context, WhatsAppVerifyDisconnectPayload, time.Duration) error
}

func NewEnqueuer(opt asynq.RedisClientOpt) *Enqueuer {
	c := asynq.NewClient(opt)
	return &Enqueuer{client: c}
}

func (e *Enqueuer) Close() error { return e.client.Close() }

// EnqueueSendEvent agenda envio assíncrono de conversões com prioridade alta.
func (e *Enqueuer) EnqueueSendEvent(ctx context.Context, in domain.SendEventInput) error {
	if e.sendHook != nil {
		return e.sendHook(ctx, in)
	}
	payload, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	task := asynq.NewTask(TypeSendEvent, payload)
	_, err = e.client.EnqueueContext(ctx, task,
		asynq.Queue(QueueCritical),
		asynq.MaxRetry(5),
	)
	return err
}

func (e *Enqueuer) EnqueueWhatsAppEvent(ctx context.Context, in WhatsAppEventPayload) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal whatsapp event: %w", err)
	}
	task := asynq.NewTask(TypeWhatsAppEvent, payload)
	_, err = e.client.EnqueueContext(ctx, task,
		asynq.Queue(QueueWhatsAppEvents),
		asynq.MaxRetry(4),
		asynq.Timeout(30*time.Second),
	)
	return err
}

func (e *Enqueuer) EnqueueWhatsAppMedia(ctx context.Context, in WhatsAppMediaPayload) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal whatsapp media: %w", err)
	}
	task := asynq.NewTask(TypeWhatsAppMedia, payload)
	_, err = e.client.EnqueueContext(ctx, task,
		asynq.Queue(QueueWhatsAppMedia),
		asynq.MaxRetry(3),
		asynq.Timeout(60*time.Second),
	)
	return err
}

// EnqueueWhatsAppDisconnectCheck agenda confirmação atrasada para absorver reconexões transitórias de WebSocket.
func (e *Enqueuer) EnqueueWhatsAppDisconnectCheck(ctx context.Context, in WhatsAppVerifyDisconnectPayload, delay time.Duration) error {
	if e.discCheckHook != nil {
		return e.discCheckHook(ctx, in, delay)
	}
	if e.client == nil {
		return fmt.Errorf("asynq client not initialized")
	}
	payload, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal whatsapp disconnect check: %w", err)
	}
	task := asynq.NewTask(TypeWhatsAppVerifyDisconnect, payload)
	_, err = e.client.EnqueueContext(ctx, task,
		asynq.Queue(QueueWhatsAppEvents),
		asynq.ProcessIn(delay),
		asynq.MaxRetry(2),
		asynq.Timeout(30*time.Second),
	)
	return err
}
