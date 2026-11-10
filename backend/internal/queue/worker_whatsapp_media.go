package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func maxAllowedMediaBytes(messageType string) int64 {
	switch messageType {
	case "audio":
		return 16 * 1024 * 1024
	case "image":
		return 10 * 1024 * 1024
	case "document", "video":
		return 15 * 1024 * 1024
	case "sticker":
		return 2 * 1024 * 1024
	default:
		return 10 * 1024 * 1024
	}
}

func makeWhatsAppMediaHandler(deps Deps) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		if deps.WhatsApp == nil {
			return nil
		}
		var payload WhatsAppMediaPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("unmarshal whatsapp media payload: %w", err)
		}
		if deps.Storage == nil || !deps.Storage.Enabled() {
			_ = deps.WhatsApp.UpdateMessageMediaStatus(ctx, payload.MessageDBID, domain.MediaStatusFailed)
			return nil
		}

		inst, err := deps.WhatsApp.GetInstance(ctx, payload.AccountID, payload.InstanceID)
		if err != nil || inst == nil {
			_ = deps.WhatsApp.UpdateMessageMediaStatus(ctx, payload.MessageDBID, domain.MediaStatusFailed)
			return nil
		}

		maxBytes := maxAllowedMediaBytes(payload.MessageType)
		cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)

		var data []byte
		var detectedMime string
		if payload.ExternalMediaURL != "" {
			data, detectedMime, err = cli.FetchExternalMedia(ctx, payload.ExternalMediaURL, maxBytes)
		} else if len(payload.RawMessageJSON) > 0 {
			data, detectedMime, err = cli.DownloadMedia(ctx, inst.InstanceToken, payload.RawMessageJSON, maxBytes)
		} else {
			_ = deps.WhatsApp.UpdateMessageMediaStatus(ctx, payload.MessageDBID, domain.MediaStatusFailed)
			return nil
		}

		if errors.Is(err, evolution.ErrMediaTooLarge) {
			_ = deps.WhatsApp.UpdateMessageMediaStatus(ctx, payload.MessageDBID, domain.MediaStatusSkippedSize)
			return nil
		}
		if err != nil {
			slog.Warn("download whatsapp media failed", "msg_db_id", payload.MessageDBID, "err", err)
			return err
		}

		mimeType := util.FirstNonEmpty(detectedMime, payload.MimeType, defaultMimeForType(payload.MessageType))
		ext := extensionForMime(mimeType, payload.MessageType)
		objKey := storage.BuildMediaObjectKey(payload.AccountID, payload.ChatID, payload.MessageDBID, ext)

		written, err := deps.Storage.PutMedia(ctx, objKey, bytes.NewReader(data), int64(len(data)), mimeType)
		if err != nil {
			slog.Error("minio upload whatsapp media failed", "key", objKey, "err", err)
			return err
		}

		if err := deps.WhatsApp.UpdateMessageMediaReady(ctx, payload.MessageDBID, objKey, mimeType, written); err != nil {
			return err
		}

		streamURL := ""
		if deps.PublicAPIURL != "" && deps.JWTSecret != "" {
			streamURL = storage.SignMediaURL(deps.PublicAPIURL, deps.JWTSecret, payload.AccountID, payload.MessageDBID, 2*time.Hour)
		}
		wshub.PublishCustom(ctx, deps.Redis, payload.AccountID, "whatsapp_media_ready", map[string]any{
			"chat_id":          payload.ChatID,
			"message_id":       payload.MessageDBID,
			"media_status":     domain.MediaStatusReady,
			"media_mime_type":  mimeType,
			"media_size_bytes": written,
			"media_stream_url": streamURL,
		})
		return nil
	}
}

func defaultMimeForType(msgType string) string {
	switch msgType {
	case "audio":
		return "audio/ogg"
	case "image":
		return "image/jpeg"
	case "video":
		return "video/mp4"
	case "sticker":
		return "image/webp"
	case "document":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func extensionForMime(mimeType, msgType string) string {
	m := strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.Contains(m, "ogg") || strings.Contains(m, "opus") || msgType == "audio":
		return ".ogg"
	case strings.Contains(m, "png"):
		return ".png"
	case strings.Contains(m, "webp"):
		return ".webp"
	case strings.Contains(m, "jpeg") || strings.Contains(m, "jpg") || msgType == "image":
		return ".jpg"
	case strings.Contains(m, "pdf"):
		return ".pdf"
	case strings.Contains(m, "mp4") || msgType == "video":
		return ".mp4"
	default:
		return ".bin"
	}
}

func evaluateAttendantTriggers(
	ctx context.Context,
	deps Deps,
	inst *domain.WhatsAppInstance,
	u *domain.TrackedUser,
	chatID int64,
	phone, waMessageID, content string,
) {
	if content == "" || deps.Enqueuer == nil {
		return
	}
	isPurchase := matchPhraseTrigger(inst.PurchaseTrigger, content)
	isLead := matchPhraseTrigger(inst.LeadTrigger, content)
	isCustom := inst.CustomEventName != "" && matchPhraseTrigger(inst.CustomEventTrigger, content)
	if !isPurchase && !isLead && !isCustom {
		return
	}

	pixel, err := deps.Pixels.FindByID(ctx, inst.PixelID)
	if err != nil || pixel == nil || !pixel.Enabled {
		return
	}

	var eventName string
	var valPtr *float64
	switch {
	case isPurchase:
		eventName = domain.EventPurchase
		if m := rxMoneyExplicit.FindAllString(content, -1); len(m) > 0 {
			if v := util.ParseMoneyAny(m[len(m)-1]); v > 0 {
				valPtr = &v
			}
		}
	case isLead:
		eventName = domain.EventLeadSubmitted
	case isCustom:
		eventName = inst.CustomEventName
	}

	eventID := waMessageID + ":" + eventName
	if exists, _ := deps.SentEvents.ExistsSent(ctx, eventID, pixel.Name); exists {
		return
	}

	chat, _ := deps.WhatsApp.GetChatByID(ctx, inst.AccountID, chatID)
	ctwaClid := u.CtwaClid
	pageID := ""
	if chat != nil {
		ctwaClid = util.FirstNonEmpty(chat.CtwaClid, ctwaClid)
		pageID = chat.PageID
	}

	actionSource := "website"
	msgChannel := ""
	if ctwaClid != "" {
		actionSource = "business_messaging"
		msgChannel = "whatsapp"
	}

	_ = deps.Enqueuer.EnqueueSendEvent(ctx, domain.SendEventInput{
		PixelName:        pixel.Name,
		EventName:        eventName,
		EventID:          eventID,
		EventTime:        time.Now().Unix(),
		ActionSource:     actionSource,
		MessagingChannel: msgChannel,
		CtwaClid:         ctwaClid,
		PageID:           pageID,
		ExternalID:       u.ExternalID,
		Phone:            util.FirstNonEmpty(phone, u.Phone),
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		FBP:              u.FBP,
		FBC:              u.FBC,
		GCLID:            u.GCLID,
		WBraid:           u.WBraid,
		GBraid:           u.GBraid,
		Value:            valPtr,
		Currency:         "BRL",
	})
}

func matchPhraseTrigger(trigger, text string) bool {
	trigger = strings.TrimSpace(trigger)
	if trigger == "" || text == "" {
		return false
	}
	if rx, err := regexp.Compile("(?i)" + trigger); err == nil {
		return rx.MatchString(text)
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(trigger))
}
