package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
	"github.com/renankeny/conversion-tracker/backend/internal/uazapi"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/waha"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

var mediaDownloadSem = make(chan struct{}, 6)

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
		if err != nil && !errors.Is(err, repo.ErrNotFound) {
			return fmt.Errorf("get whatsapp instance %d: %w", payload.InstanceID, err)
		}
		if inst == nil {
			_ = deps.WhatsApp.UpdateMessageMediaStatus(ctx, payload.MessageDBID, domain.MediaStatusFailed)
			return nil
		}

		select {
		case mediaDownloadSem <- struct{}{}:
			defer func() { <-mediaDownloadSem }()
		case <-ctx.Done():
			return ctx.Err()
		}

		maxBytes := maxAllowedMediaBytes(payload.MessageType)
		data, detectedMime, ok, err := fetchInstanceMediaBytes(ctx, inst, payload, maxBytes)
		if !ok {
			_ = deps.WhatsApp.UpdateMessageMediaStatus(ctx, payload.MessageDBID, domain.MediaStatusFailed)
			return nil
		}
		if errors.Is(err, evolution.ErrMediaTooLarge) || errors.Is(err, uazapi.ErrMediaTooLarge) || errors.Is(err, waha.ErrMediaTooLarge) {
			_ = deps.WhatsApp.UpdateMessageMediaStatus(ctx, payload.MessageDBID, domain.MediaStatusSkippedSize)
			return nil
		}
		if err != nil {
			slog.Warn("download whatsapp media failed", "msg_db_id", payload.MessageDBID, "provider", inst.Provider, "err", err)
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

func fetchInstanceMediaBytes(
	ctx context.Context,
	inst *domain.WhatsAppInstance,
	payload WhatsAppMediaPayload,
	maxBytes int64,
) ([]byte, string, bool, error) {
	if inst.IsWAHA() {
		wcli := waha.NewClient(inst.ServerURL, inst.ServerAPIKey)
		msgID := util.FirstNonEmpty(payload.RawWhatsAppMessageID, payload.WhatsAppMessageID)
		if payload.ExternalMediaURL == "" && (!wcli.Configured() || payload.ChatJID == "" || msgID == "") {
			return nil, "", false, nil
		}
		data, mime, err := wcli.DownloadMedia(ctx, inst.InstanceName, payload.ExternalMediaURL, payload.ChatJID, msgID, maxBytes)
		return data, mime, true, err
	}

	if inst.IsExternalUazapi() {
		ucli := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken)
		if payload.ExternalMediaURL != "" {
			data, mime, err := ucli.FetchExternalMedia(ctx, payload.ExternalMediaURL, maxBytes)
			if err == nil || errors.Is(err, uazapi.ErrMediaTooLarge) {
				return data, mime, true, err
			}
			if payload.WhatsAppMessageID == "" && payload.RawWhatsAppMessageID == "" {
				return nil, "", true, err
			}
		}
		if payload.WhatsAppMessageID != "" || payload.RawWhatsAppMessageID != "" {
			if !ucli.Configured() {
				return nil, "", false, nil
			}
			data, mime, err := ucli.DownloadMedia(ctx, payload.WhatsAppMessageID, payload.RawWhatsAppMessageID, maxBytes)
			return data, mime, true, err
		}
		return nil, "", false, nil
	}

	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	if payload.ExternalMediaURL != "" {
		data, mime, err := cli.FetchExternalMedia(ctx, payload.ExternalMediaURL, maxBytes)
		return data, mime, true, err
	}
	if len(payload.RawMessageJSON) > 0 {
		data, mime, err := cli.DownloadMedia(ctx, inst.InstanceToken, payload.RawMessageJSON, maxBytes)
		return data, mime, true, err
	}
	return nil, "", false, nil
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
	case strings.Contains(m, "mpeg") || strings.Contains(m, "mp3"):
		return ".mp3"
	case strings.Contains(m, "ogg") || strings.Contains(m, "opus"):
		return ".ogg"
	case strings.Contains(m, "m4a") || strings.Contains(m, "aac") || (msgType == "audio" && strings.Contains(m, "mp4")):
		return ".m4a"
	case strings.Contains(m, "wav"):
		return ".wav"
	case strings.Contains(m, "webm"):
		return ".webm"
	case strings.Contains(m, "png"):
		return ".png"
	case strings.Contains(m, "webp"):
		return ".webp"
	case strings.Contains(m, "gif"):
		return ".gif"
	case strings.Contains(m, "jpeg") || strings.Contains(m, "jpg"):
		return ".jpg"
	case strings.Contains(m, "pdf"):
		return ".pdf"
	case strings.Contains(m, "wordprocessingml") || strings.Contains(m, "msword"):
		return ".docx"
	case strings.Contains(m, "spreadsheetml") || strings.Contains(m, "excel"):
		return ".xlsx"
	case strings.Contains(m, "csv"):
		return ".csv"
	case strings.Contains(m, "vcard"):
		return ".vcf"
	case strings.Contains(m, "quicktime"):
		return ".mov"
	case strings.Contains(m, "3gp"):
		return ".3gp"
	case strings.Contains(m, "mp4"):
		return ".mp4"
	case msgType == "audio":
		return ".ogg"
	case msgType == "image":
		return ".jpg"
	case msgType == "sticker":
		return ".webp"
	case msgType == "video":
		return ".mp4"
	default:
		return ".bin"
	}
}
