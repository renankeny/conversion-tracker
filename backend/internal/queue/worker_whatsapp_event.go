package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func makeWhatsAppEventHandler(deps Deps) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		if deps.WhatsApp == nil {
			return nil
		}
		var payload WhatsAppEventPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("unmarshal whatsapp event: %w", err)
		}

		inst, err := deps.WhatsApp.GetInstanceByIDUnscoped(ctx, payload.InstanceID)
		if err != nil {
			return fmt.Errorf("load whatsapp instance %d: %w", payload.InstanceID, err)
		}
		if inst == nil || !inst.Enabled {
			return nil
		}

		switch payload.Event {
		case "Connected", "PairSuccess", "QRSuccess":
			handleEvolutionConnected(ctx, deps, inst, payload.Data)
		case "ProfileUpdated":
			handleEvolutionProfile(ctx, deps, inst, payload.Data)
		case "Disconnected", "LoggedOut", "ConnectFailure", "TemporaryBan":
			status := "DISCONNECTED"
			if payload.Event == "LoggedOut" {
				status = "CLOSED"
			}
			_ = deps.WhatsApp.UpdateInstanceStatus(ctx, inst.ID, status)
			wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
				"instance_id": inst.ID,
				"status":      status,
				"event":       payload.Event,
			})
		case "QRCode":
			var qr struct {
				Qrcode string `json:"qrcode"`
				Code   string `json:"code"`
				Count  int    `json:"count"`
			}
			if json.Unmarshal(payload.Data, &qr) == nil && qr.Qrcode != "" {
				wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_qr", map[string]any{
					"instance_id": inst.ID,
					"qrcode":      qr.Qrcode,
					"code":        qr.Code,
					"count":       qr.Count,
				})
			}
		case "QRTimeout":
			_ = deps.WhatsApp.UpdateInstanceStatus(ctx, inst.ID, "CLOSED")
			wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
				"instance_id": inst.ID,
				"status":      "CLOSED",
				"event":       "QRTimeout",
			})
		case "Message", "SendMessage":
			return processEvolutionMessage(ctx, deps, inst, payload.Data)
		}
		return nil
	}
}

func processEvolutionMessage(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, rawData json.RawMessage) error {
	parsed, err := ParseEvolutionMessageEvent(rawData)
	if err != nil || parsed == nil {
		return nil
	}

	remoteJID, phone, lidDigits, skip := ResolveContactJID(
		parsed,
		inst.OwnerJID,
		func(lidJID string) string { return deps.WhatsApp.ResolvePhoneByLID(ctx, inst.AccountID, lidJID) },
		func(lidJID, phoneJID string) { _ = deps.WhatsApp.UpsertLIDMapping(ctx, inst.AccountID, lidJID, phoneJID) },
	)
	if skip {
		return nil
	}

	lidJID := ""
	if lidDigits != "" {
		lidJID = lidDigits + "@lid"
	}

	existingChat, err := deps.WhatsApp.FindExistingChatForContact(ctx, inst.ID, remoteJID, phone, lidJID)
	if err != nil {
		return fmt.Errorf("find existing chat: %w", err)
	}

	protocolCode := ExtractProtocolCode(parsed.Content)
	hasPortal := parsed.CtwaClid != "" || parsed.SourceID != "" || protocolCode != ""
	if existingChat == nil && !hasPortal {
		return nil
	}

	var trackedUser *domain.TrackedUser
	originType := ""
	if existingChat != nil {
		remoteJID = existingChat.RemoteJid
		originType = existingChat.OriginType
		if existingChat.TrackedUserID != nil {
			trackedUser, _ = deps.Users.FindByID(ctx, *existingChat.TrackedUserID)
		}
	}

	if hasPortal || trackedUser == nil {
		u, portalOrigin := resolveOrCreatePortalLead(ctx, deps, inst, parsed, protocolCode, phone, lidDigits)
		if u != nil {
			trackedUser = u
			if portalOrigin != "" {
				originType = portalOrigin
			}
		}
	}
	if trackedUser == nil {
		return nil
	}

	if phone != "" && trackedUser.Phone == "" {
		trackedUser.Phone = phone
		_ = deps.Users.Upsert(ctx, trackedUser)
	}

	var crmLeadID *int64
	if inst.AutoCreateCRMLead && deps.CRM != nil {
		if id := ensureCRMLeadForWhatsApp(ctx, deps, inst, trackedUser.ID); id > 0 {
			crmLeadID = &id
		}
	} else if existingChat != nil {
		crmLeadID = existingChat.CRMLeadID
	}

	contactName := parsed.PushName
	if parsed.IsFromMe && existingChat != nil && existingChat.ContactName != nil {
		contactName = *existingChat.ContactName
	}
	var contactPtr *string
	if contactName != "" {
		contactPtr = &contactName
	}

	chatRow := &domain.WhatsAppChat{
		AccountID: inst.AccountID, InstanceID: inst.ID, RemoteJid: remoteJID,
		TrackedUserID: &trackedUser.ID, CRMLeadID: crmLeadID, Phone: phone, LidJid: lidJID,
		PushName: contactName, ContactName: contactPtr, OriginType: originType,
		CtwaClid: parsed.CtwaClid, SourceID: parsed.SourceID, SourceApp: parsed.SourceApp,
		AdThumbnailURL: parsed.AdThumbnailURL, LastMessageTime: parsed.Timestamp,
	}
	chatID, err := deps.WhatsApp.UpsertActiveChat(ctx, chatRow, !parsed.IsFromMe)
	if err != nil {
		return fmt.Errorf("upsert whatsapp chat: %w", err)
	}

	if parsed.CtwaClid != "" || parsed.SourceID != "" {
		enrichAndDispatchCTWALead(ctx, deps, inst, trackedUser, chatID, phone, parsed.SourceID, parsed.CtwaClid)
	}

	mediaStatus, expiresAt := classifyInitialMediaStatus(deps, parsed)
	var mimePtr *string
	if parsed.MimeType != "" {
		mimePtr = &parsed.MimeType
	}
	dbMsg := &domain.WhatsAppMessage{
		ChatID: chatID, WhatsAppMessageID: parsed.MessageID, IsFromMe: parsed.IsFromMe,
		MessageType: parsed.MessageType, Content: parsed.Content, HasMedia: parsed.HasMedia,
		MediaMimeType: mimePtr, MediaStatus: mediaStatus, MediaSizeBytes: parsed.FileLength,
		MediaDurationSec: parsed.DurationSec, MediaFilename: parsed.Filename,
		MediaExpiresAt: expiresAt, Timestamp: parsed.Timestamp,
	}
	inserted, err := deps.WhatsApp.CreateMessage(ctx, dbMsg)
	if err != nil {
		slog.Warn("save whatsapp message failed", "msg_id", parsed.MessageID, "err", err)
		return nil
	}
	if !inserted {
		return nil
	}

	evaluateAttendantTriggers(ctx, deps, inst, trackedUser, chatID, phone, parsed.MessageID, parsed.Content)
	wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_message", map[string]any{
		"chat_id": chatID, "tracked_user_id": trackedUser.ID, "crm_lead_id": crmLeadID, "message": dbMsg,
	})

	if mediaStatus == domain.MediaStatusPending && deps.Enqueuer != nil {
		_ = deps.Enqueuer.EnqueueWhatsAppMedia(ctx, WhatsAppMediaPayload{
			AccountID: inst.AccountID, InstanceID: inst.ID, ChatID: chatID, MessageDBID: dbMsg.ID,
			MessageType: parsed.MessageType, MimeType: parsed.MimeType, ExternalMediaURL: parsed.ExternalMediaURL,
			ExpectedBytes: parsed.FileLength, RawMessageJSON: parsed.MediaProtoJSON,
		})
	}
	return nil
}
