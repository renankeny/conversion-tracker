package wshub

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// PublishToUser envia um evento WebSocket direcionado especificamente a um user_id,
// independente de qual empresa esteja selecionada no painel no momento.
func PublishToUser(ctx context.Context, rc *redis.Client, userID int64, eventType string, data any) {
	if rc == nil || eventType == "" || userID <= 0 {
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	env, err := json.Marshal(customRedisEvent{
		TargetUserID: userID,
		Type:         eventType,
		Data:         raw,
	})
	if err != nil {
		return
	}
	if err := rc.Publish(ctx, CustomChannel, env).Err(); err != nil {
		slog.Warn("wshub: user publish failed", "type", eventType, "err", err)
	}
}

func (h *Hub) broadcast(e LiveEvent) {
	if e.AccountID <= 0 {
		return
	}
	out, err := json.Marshal(wsEnvelope{Type: "new_event", Data: e})
	if err != nil {
		return
	}
	var slow []*Client
	h.mu.RLock()
	for c := range h.clients {
		if c.AccountID > 0 {
			if c.AccountID != e.AccountID {
				continue
			}
		} else if !c.IsOwner {
			continue
		}
		if c.PixelFilter != "" && c.PixelFilter != e.PixelName {
			continue
		}
		select {
		case c.Send <- out:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range slow {
		h.Unregister(c)
	}
}

func (h *Hub) broadcastCustom(ce customRedisEvent) {
	if ce.TargetUserID <= 0 && ce.AccountID <= 0 {
		return
	}
	out, err := json.Marshal(wsEnvelope{Type: ce.Type, Data: ce.Data})
	if err != nil {
		return
	}
	var slow []*Client
	h.mu.RLock()
	for c := range h.clients {
		if ce.TargetUserID > 0 {
			if c.UserID != ce.TargetUserID {
				continue
			}
		} else if c.AccountID > 0 {
			if c.AccountID != ce.AccountID {
				continue
			}
		} else if !c.IsOwner {
			continue
		}
		select {
		case c.Send <- out:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range slow {
		h.Unregister(c)
	}
}
