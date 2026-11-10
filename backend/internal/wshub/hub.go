package wshub

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	Channel       = "ws:events"
	CustomChannel = "ws:custom"
)

type LiveEvent struct {
	AccountID  int64   `json:"account_id"`
	EventID    string  `json:"event_id"`
	EventName  string  `json:"event_name"`
	PixelName  string  `json:"pixel_name"`
	ExternalID string  `json:"external_id"`
	Channels   string  `json:"channels"`
	Status     string  `json:"status"`
	Ts         string  `json:"ts"`
	Value      float64 `json:"value,omitempty"`
	City       string  `json:"city,omitempty"`
	State      string  `json:"state,omitempty"`
	Country    string  `json:"country,omitempty"`
}

type customRedisEvent struct {
	AccountID int64           `json:"account_id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
}

type wsEnvelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Client struct {
	AccountID   int64
	PixelFilter string
	Send        chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	done       chan struct{}
	redis      *redis.Client
}

func New(rc *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		done:       make(chan struct{}),
		redis:      rc,
	}
}

func (h *Hub) Register(c *Client) {
	select {
	case h.register <- c:
	case <-h.done:
	}
}

func (h *Hub) Unregister(c *Client) {
	select {
	case h.unregister <- c:
	case <-h.done:
	}
}

func Publish(ctx context.Context, rc *redis.Client, e LiveEvent) {
	if rc == nil {
		return
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	if err := rc.Publish(ctx, Channel, b).Err(); err != nil {
		slog.Warn("wshub: redis publish failed", "err", err)
	}
}

// PublishCustom publica notificações tipadas em tempo real por conta (ex: whatsapp_message,
// whatsapp_media_ready, whatsapp_qr, whatsapp_status).
func PublishCustom(ctx context.Context, rc *redis.Client, accountID int64, eventType string, data any) {
	if rc == nil || eventType == "" {
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	env, err := json.Marshal(customRedisEvent{
		AccountID: accountID,
		Type:      eventType,
		Data:      raw,
	})
	if err != nil {
		return
	}
	if err := rc.Publish(ctx, CustomChannel, env).Err(); err != nil {
		slog.Warn("wshub: custom publish failed", "type", eventType, "err", err)
	}
}

func (h *Hub) Run(ctx context.Context) {
	defer close(h.done)
	if h.redis == nil {
		return
	}

	for {
		if ctx.Err() != nil {
			return
		}
		if !h.runSession(ctx) {
			return
		}
		slog.Warn("wshub: redis pubsub disconnected, reconnecting in 2s")
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (h *Hub) runSession(ctx context.Context) (reconnect bool) {
	sub := h.redis.Subscribe(ctx, Channel, CustomChannel)
	defer sub.Close()
	ch := sub.Channel()

	for {
		select {
		case <-ctx.Done():
			return false

		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.Send)
			}
			h.mu.Unlock()

		case msg, ok := <-ch:
			if !ok {
				return ctx.Err() == nil
			}
			if msg.Channel == CustomChannel {
				var ce customRedisEvent
				if err := json.Unmarshal([]byte(msg.Payload), &ce); err != nil {
					slog.Warn("wshub: invalid custom payload", "err", err)
					continue
				}
				h.broadcastCustom(ce)
				continue
			}
			var e LiveEvent
			if err := json.Unmarshal([]byte(msg.Payload), &e); err != nil {
				slog.Warn("wshub: invalid payload", "err", err)
				continue
			}
			h.broadcast(e)
		}
	}
}

func (h *Hub) broadcast(e LiveEvent) {
	out, err := json.Marshal(wsEnvelope{Type: "new_event", Data: e})
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c.AccountID != 0 && c.AccountID != e.AccountID {
			continue
		}
		if c.PixelFilter != "" && c.PixelFilter != e.PixelName {
			continue
		}
		select {
		case c.Send <- out:
		default:
			slog.Warn("wshub: slow client, dropping message", "account_id", e.AccountID, "event", e.EventName)
		}
	}
}

func (h *Hub) broadcastCustom(ce customRedisEvent) {
	out, err := json.Marshal(wsEnvelope{Type: ce.Type, Data: ce.Data})
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c.AccountID != 0 && ce.AccountID != 0 && c.AccountID != ce.AccountID {
			continue
		}
		select {
		case c.Send <- out:
		default:
		}
	}
}
