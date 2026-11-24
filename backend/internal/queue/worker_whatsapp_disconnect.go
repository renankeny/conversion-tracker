package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

const (
	whatsappDisconnectGracePeriod = 35 * time.Second
	whatsappPendingDiscKeyPrefix  = "wa:disc_pending:"
	whatsappPendingDiscTTL        = 2 * time.Minute
)

var (
	pendingDisconnectMem     sync.Map
	evolutionVerifyRetryWait = 3 * time.Second
)

func whatsAppPendingDisconnectKey(instanceID int64) string {
	return fmt.Sprintf("%s%d", whatsappPendingDiscKeyPrefix, instanceID)
}

func markPendingWhatsAppDisconnect(ctx context.Context, rdb *redis.Client, instanceID, triggeredAtUnix int64) bool {
	key := whatsAppPendingDisconnectKey(instanceID)
	if rdb != nil {
		ok, err := rdb.SetNX(ctx, key, strconv.FormatInt(triggeredAtUnix, 10), whatsappPendingDiscTTL).Result()
		if err == nil {
			return ok
		}
	}
	_, loaded := pendingDisconnectMem.LoadOrStore(key, triggeredAtUnix)
	return !loaded
}

func hasPendingWhatsAppDisconnect(ctx context.Context, rdb *redis.Client, instanceID int64) bool {
	key := whatsAppPendingDisconnectKey(instanceID)
	if rdb != nil {
		n, err := rdb.Exists(ctx, key).Result()
		if err == nil {
			return n > 0
		}
	}
	_, ok := pendingDisconnectMem.Load(key)
	return ok
}

func clearPendingWhatsAppDisconnect(ctx context.Context, rdb *redis.Client, instanceID int64) {
	key := whatsAppPendingDisconnectKey(instanceID)
	pendingDisconnectMem.Delete(key)
	if rdb != nil {
		_ = rdb.Del(ctx, key).Err()
	}
}

// isPermanentConnectFailure distingue logout/banimento definitivo de falhas transitórias de socket.
func isPermanentConnectFailure(raw json.RawMessage) (permanent bool, loggedOut bool) {
	if len(raw) == 0 {
		return false, false
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) != nil {
		return false, false
	}
	var rawReason string
	for _, k := range []string{"reason", "Reason", "code", "status"} {
		if v, ok := payload[k]; ok && v != nil {
			rawReason = strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v)))
			if rawReason != "" {
				break
			}
		}
	}
	switch {
	case rawReason == "401" || strings.Contains(rawReason, "loggedout") || strings.Contains(rawReason, "logged_out") || strings.Contains(rawReason, "maindevicegone"):
		return true, true
	case rawReason == "403" || rawReason == "406" || strings.Contains(rawReason, "banned") || strings.Contains(rawReason, "forbidden") || strings.Contains(rawReason, "outdated"):
		return true, false
	default:
		return false, false
	}
}

func applyConfirmedWhatsAppDisconnect(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, event string, resetSession bool) {
	clearPendingWhatsAppDisconnect(ctx, deps.Redis, inst.ID)
	wasOpen := strings.EqualFold(inst.ConnectionStatus, "OPEN")
	status := "DISCONNECTED"
	if resetSession {
		status = "CLOSED"
	}
	if deps.WhatsApp != nil {
		if resetSession {
			_ = deps.WhatsApp.ResetInstanceSession(ctx, inst.ID, status)
		} else {
			_ = deps.WhatsApp.UpdateInstanceStatus(ctx, inst.ID, status)
		}
	}
	inst.ConnectionStatus = status
	wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
		"instance_id": inst.ID,
		"status":      status,
		"event":       event,
	})
	if wasOpen {
		notifyWhatsAppStateChange(deps, inst, false)
	}
}

// handleEvolutionDisconnectSignal aplica janela de confirmação em quedas transitórias da Evolution Go.
func handleEvolutionDisconnectSignal(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, event string, raw json.RawMessage) {
	switch event {
	case "LoggedOut":
		applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, true)
		return
	case "TemporaryBan":
		applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, false)
		return
	case "ConnectFailure":
		if perm, loggedOut := isPermanentConnectFailure(raw); perm {
			applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, loggedOut)
			return
		}
	}

	if strings.EqualFold(inst.ConnectionStatus, "CLOSED") || strings.EqualFold(inst.ConnectionStatus, "DISCONNECTED") {
		return
	}

	nowUnix := time.Now().Unix()
	if deps.Enqueuer != nil {
		if !markPendingWhatsAppDisconnect(ctx, deps.Redis, inst.ID, nowUnix) {
			return
		}
		err := deps.Enqueuer.EnqueueWhatsAppDisconnectCheck(ctx, WhatsAppVerifyDisconnectPayload{
			AccountID:       inst.AccountID,
			InstanceID:      inst.ID,
			Event:           event,
			TriggeredAtUnix: nowUnix,
		}, whatsappDisconnectGracePeriod)
		if err == nil {
			slog.Debug("whatsapp transient disconnect queued for confirmation", "instance_id", inst.ID, "event", event)
			return
		}
		clearPendingWhatsAppDisconnect(ctx, deps.Redis, inst.ID)
		slog.Warn("whatsapp disconnect check enqueue failed, probing inline", "instance_id", inst.ID, "err", err)
	}

	verifyAndConfirmEvolutionDisconnect(ctx, deps, inst, event, true)
}

func makeWhatsAppVerifyDisconnectHandler(deps Deps) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var payload WhatsAppVerifyDisconnectPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("unmarshal whatsapp verify disconnect: %w", err)
		}
		if !hasPendingWhatsAppDisconnect(ctx, deps.Redis, payload.InstanceID) {
			return nil
		}
		if deps.WhatsApp == nil {
			clearPendingWhatsAppDisconnect(ctx, deps.Redis, payload.InstanceID)
			return nil
		}
		inst, err := deps.WhatsApp.GetInstanceByIDUnscoped(ctx, payload.InstanceID)
		if err != nil {
			return fmt.Errorf("load whatsapp instance %d: %w", payload.InstanceID, err)
		}
		if inst == nil || !inst.Enabled || strings.EqualFold(inst.ConnectionStatus, "CLOSED") || strings.EqualFold(inst.ConnectionStatus, "DISCONNECTED") {
			clearPendingWhatsAppDisconnect(ctx, deps.Redis, payload.InstanceID)
			return nil
		}
		retryCount, _ := asynq.GetRetryCount(ctx)
		maxRetry, hasMax := asynq.GetMaxRetry(ctx)
		allowRetryOnNetErr := !hasMax || retryCount < maxRetry
		return verifyAndConfirmEvolutionDisconnect(ctx, deps, inst, payload.Event, !allowRetryOnNetErr)
	}
}

func verifyAndConfirmEvolutionDisconnect(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, event string, confirmOnNetErr bool) error {
	if inst.ServerURL == "" || inst.InstanceToken == "" {
		applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, false)
		return nil
	}
	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	st, err := cli.GetStatus(ctx, inst.InstanceToken)
	if err != nil {
		if !confirmOnNetErr {
			return fmt.Errorf("probe evolution status %d: %w", inst.ID, err)
		}
		applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, false)
		return nil
	}
	if !st.Connected || !st.LoggedIn {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(evolutionVerifyRetryWait):
		}
		if st2, err2 := cli.GetStatus(ctx, inst.InstanceToken); err2 == nil && st2 != nil {
			st = st2
		}
	}
	target := util.FirstNonEmpty(inst.EvolutionUUID, inst.InstanceName)
	info, _ := cli.GetInfo(ctx, target)
	if st.Connected && (st.LoggedIn || (info != nil && info.Connected)) {
		clearPendingWhatsAppDisconnect(ctx, deps.Redis, inst.ID)
		ownerBare := util.FirstNonEmpty(NormalizeBareJID(st.MyJID), inst.OwnerJID)
		profName := util.FirstNonEmpty(strings.TrimSpace(st.Name), inst.ProfileName)
		if deps.WhatsApp != nil {
			_ = deps.WhatsApp.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, inst.ProfilePicURL)
		}
		return nil
	}
	resetSession := info != nil && strings.TrimSpace(info.JID) == "" && !st.LoggedIn && strings.TrimSpace(info.Qrcode) == ""
	applyConfirmedWhatsAppDisconnect(ctx, deps, inst, event, resetSession)
	return nil
}
