package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
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

		if HandleWhatsAppLifecycleEvent(ctx, deps, inst, payload.Event, payload.Data) {
			return nil
		}

		switch payload.Event {
		case "Message", "SendMessage":
			parsed, err := ParseEvolutionMessageEvent(payload.Data)
			if err != nil || parsed == nil {
				return nil
			}
			return processParsedWhatsAppMessage(ctx, deps, inst, parsed)
		case "UazapiConnection":
			syncUazapiMetaFromWebhook(ctx, deps, inst, payload.Data)
		case "UazapiMessage":
			syncUazapiMetaFromWebhook(ctx, deps, inst, payload.Data)
			parsed, err := ParseUazapiMessageEvent(payload.Data)
			if err != nil || parsed == nil {
				return nil
			}
			return processParsedWhatsAppMessage(ctx, deps, inst, parsed)
		}
		return nil
	}
}

func processParsedWhatsAppMessage(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, parsed *ParsedWhatsAppMessage) error {
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
	msgCtwaClid, msgSourceID, msgProto := resolveMessagePortalFields(parsed, existingChat, protocolCode)
	hasPortal := msgCtwaClid != "" || msgSourceID != "" || msgProto != ""

	var trackedUser *domain.TrackedUser
	originType := ""
	if existingChat != nil {
		remoteJID = existingChat.RemoteJid
		originType = existingChat.OriginType
		if existingChat.TrackedUserID != nil {
			trackedUser, _ = deps.Users.FindByID(ctx, *existingChat.TrackedUserID)
		}
	}

	assistedUser, assistedOrigin, matchedClick, matchedLinkID := resolveAssistedTrackableLinkLead(
		ctx, deps, inst, existingChat, parsed, protocolCode, phone, lidDigits,
	)
	if assistedUser != nil {
		trackedUser = assistedUser
		if assistedOrigin != "" {
			originType = assistedOrigin
		}
		hasPortal = true
	} else if hasPortal || trackedUser == nil {
		u, portalOrigin := resolveOrCreatePortalLead(ctx, deps, inst, parsed, protocolCode, phone, lidDigits)
		if u != nil {
			trackedUser = u
			if portalOrigin != "" && (originType == "" || originType == "unknown" || hasPortal) {
				originType = portalOrigin
			}
		}
	} else if originType == domain.WhatsAppOriginWebsiteOrganic || originType == domain.WhatsAppOriginBioLinkOrganic {
		if reclassified := ClassifyPortalOrigin("", "", trackedUser); reclassified != "" && reclassified != domain.WhatsAppOriginWebsiteOrganic {
			originType = reclassified
		}
	}
	if trackedUser == nil {
		return nil
	}
	effLinkID := resolveEffectiveTrackableLinkID(matchedClick, matchedLinkID, existingChat, trackedUser)

	contactName, contactPtr := enrichInboundWhatsAppContact(ctx, deps, existingChat, parsed, trackedUser, phone)

	var crmLeadID *int64
	if inst.AutoCreateCRMLead && deps.CRM != nil {
		if id := ensureCRMLeadForWhatsApp(ctx, deps, inst, trackedUser.ID); id > 0 {
			crmLeadID = &id
		}
	} else if existingChat != nil {
		crmLeadID = existingChat.CRMLeadID
	}

	effCtwaClid := parsed.CtwaClid
	effSourceID := parsed.SourceID
	if existingChat != nil {
		effCtwaClid = util.FirstNonEmpty(effCtwaClid, existingChat.CtwaClid)
		effSourceID = util.FirstNonEmpty(effSourceID, existingChat.SourceID)
	}
	effCtwaClid = util.FirstNonEmpty(effCtwaClid, trackedUser.CtwaClid)
	effSourceID = util.FirstNonEmpty(effSourceID, trackedUser.WaAdID)

	chatRow := &domain.WhatsAppChat{
		AccountID: inst.AccountID, InstanceID: inst.ID, RemoteJid: remoteJID,
		TrackedUserID: &trackedUser.ID, CRMLeadID: crmLeadID, TrackableLinkID: effLinkID, Phone: phone, LidJid: lidJID,
		PushName: contactName, ContactName: contactPtr, OriginType: originType,
		CtwaClid: effCtwaClid, SourceID: effSourceID, SourceApp: parsed.SourceApp,
		AdThumbnailURL: parsed.AdThumbnailURL, LastMessageTime: parsed.Timestamp,
	}
	chatID, err := deps.WhatsApp.UpsertActiveChat(ctx, chatRow, !parsed.IsFromMe)
	if err != nil {
		return fmt.Errorf("upsert whatsapp chat: %w", err)
	}

	var ctwaPixel *domain.Pixel
	var ctwaPageID string
	if effCtwaClid != "" || effSourceID != "" {
		ctwaPixel, ctwaPageID = enrichCTWAAdMetadata(ctx, deps, inst, trackedUser, chatID, effSourceID)
		if ctwaPageID == "" && existingChat != nil {
			ctwaPageID = existingChat.PageID
		}
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
		MediaExpiresAt: expiresAt, CtwaClid: msgCtwaClid, SourceID: msgSourceID,
		ProtocolCode: msgProto, Timestamp: parsed.Timestamp,
	}
	inserted, err := deps.WhatsApp.CreateMessage(ctx, dbMsg)
	if err != nil {
		slog.Warn("save whatsapp message failed", "msg_id", parsed.MessageID, "err", err)
		return nil
	}
	if inserted {
		_ = deps.WhatsApp.RefreshChatSLAMetrics(ctx, chatID)
	}
	retryCount, _ := asynq.GetRetryCount(ctx)
	if !inserted && retryCount == 0 {
		return nil
	}

	if effCtwaClid != "" || effSourceID != "" {
		if err := maybeDispatchCTWALeadSubmitted(ctx, deps, inst, ctwaPixel, trackedUser, chatID, phone, effSourceID, effCtwaClid, ctwaPageID, !parsed.IsFromMe); err != nil {
			return err
		}
	} else if effLinkID != nil {
		finalizeTrackableLinkFirstMessage(ctx, deps, inst, matchedClick, effLinkID, trackedUser, chatID, phone, parsed.IsFromMe)
	}

	if err := evaluateAttendantTriggers(ctx, deps, inst, trackedUser, chatID, phone, parsed.MessageID, parsed.Content, parsed.IsFromMe); err != nil {
		return fmt.Errorf("evaluate attendant triggers: %w", err)
	}
	if !inserted {
		return nil
	}
	if existingChat == nil || hasPortal {
		cache.BumpDashboardVersion(ctx, deps.Redis, inst.AccountID)
	}
	maybeNotifyNewWhatsAppLead(deps, inst, existingChat, parsed.IsFromMe, trackedUser, contactName, phone, originType)
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
