package queue

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
	"github.com/renankeny/conversion-tracker/backend/internal/uazapi"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func handleEvolutionConnected(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, raw json.RawMessage) {
	var d struct {
		JID      string `json:"jid"`
		PushName string `json:"pushName"`
	}
	_ = json.Unmarshal(raw, &d)
	ownerBare := NormalizeBareJID(d.JID)
	profName := strings.TrimSpace(d.PushName)
	profPic := ""

	if inst.ServerURL != "" && inst.InstanceToken != "" {
		cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
		if st, err := cli.GetStatus(ctx, inst.InstanceToken); err == nil && st != nil {
			if ownerBare == "" && st.MyJID != "" {
				ownerBare = NormalizeBareJID(st.MyJID)
			}
			if profName == "" && st.Name != "" {
				profName = strings.TrimSpace(st.Name)
			}
		}
		if ownerBare != "" {
			profPic = cli.GetUserAvatar(ctx, inst.InstanceToken, ownerBare)
		}
	}

	_ = deps.WhatsApp.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, profPic)
	wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
		"instance_id":     inst.ID,
		"status":          "OPEN",
		"owner_jid":       ownerBare,
		"profile_name":    profName,
		"profile_pic_url": profPic,
	})
}

func handleEvolutionProfile(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, raw json.RawMessage) {
	var d struct {
		JID           string `json:"jid"`
		ProfileName   string `json:"profileName"`
		ProfilePicURL string `json:"profilePicUrl"`
	}
	if json.Unmarshal(raw, &d) == nil {
		ownerBare := NormalizeBareJID(d.JID)
		_ = deps.WhatsApp.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, d.ProfileName, d.ProfilePicURL)
		wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
			"instance_id":     inst.ID,
			"status":          "OPEN",
			"owner_jid":       ownerBare,
			"profile_name":    strings.TrimSpace(d.ProfileName),
			"profile_pic_url": strings.TrimSpace(d.ProfilePicURL),
		})
	}
}

func syncUazapiMetaFromWebhook(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, raw json.RawMessage) {
	meta := ParseUazapiWebhookMeta(raw)
	if (meta.BaseURL != "" && inst.ExternalAPIURL == "") || (meta.Token != "" && inst.InstanceToken == "") {
		_ = deps.WhatsApp.UpdateInstanceExternalCredentials(ctx, inst.ID, meta.BaseURL, meta.Token)
		if inst.ExternalAPIURL == "" {
			inst.ExternalAPIURL = meta.BaseURL
		}
		if inst.InstanceToken == "" {
			inst.InstanceToken = meta.Token
		}
	}

	ownerBare := normalizeUazapiPhoneJID(meta.Owner)
	if ownerBare == "" {
		ownerBare = inst.OwnerJID
	}
	profName := meta.ProfileName
	profPic := meta.ProfilePicURL

	if meta.ConnStatus == "disconnected" || meta.ConnStatus == "hibernated" {
		_ = deps.WhatsApp.UpdateInstanceStatus(ctx, inst.ID, "DISCONNECTED")
		wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
			"instance_id": inst.ID,
			"status":      "DISCONNECTED",
		})
		return
	}

	if (profPic == "" || profName == "" || ownerBare == "") && inst.ExternalAPIURL != "" && inst.InstanceToken != "" {
		cli := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken)
		if st, err := cli.GetStatus(ctx); err == nil && st != nil {
			if ownerBare == "" && st.OwnerJID != "" {
				ownerBare = NormalizeBareJID(st.OwnerJID)
			}
			if profName == "" && st.ProfileName != "" {
				profName = st.ProfileName
			}
			if profPic == "" && st.ProfilePicURL != "" {
				profPic = st.ProfilePicURL
			}
		}
	}

	needUpdate := inst.ConnectionStatus != "OPEN" ||
		(ownerBare != "" && ownerBare != inst.OwnerJID) ||
		(profName != "" && profName != inst.ProfileName) ||
		(profPic != "" && profPic != inst.ProfilePicURL)

	if ownerBare != "" {
		inst.OwnerJID = ownerBare
	}
	if needUpdate {
		_ = deps.WhatsApp.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, profPic)
		wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
			"instance_id":     inst.ID,
			"status":          "OPEN",
			"owner_jid":       ownerBare,
			"profile_name":    profName,
			"profile_pic_url": profPic,
		})
	}
}

func classifyInitialMediaStatus(deps Deps, m *ParsedWhatsAppMessage) (string, *time.Time) {
	if !m.HasMedia {
		return domain.MediaStatusNone, nil
	}
	maxBytes := maxAllowedMediaBytes(m.MessageType)
	if maxBytes > 0 && m.FileLength > maxBytes {
		return domain.MediaStatusSkippedSize, nil
	}
	if deps.Storage == nil || !deps.Storage.Enabled() {
		return domain.MediaStatusFailed, nil
	}
	retDays := deps.MediaRetentionDays
	if retDays <= 0 {
		retDays = 14
	}
	exp := m.Timestamp.AddDate(0, 0, retDays)
	return domain.MediaStatusPending, &exp
}

func SignMessageStreamIfReady(m *domain.WhatsAppMessage, publicAPIURL, jwtSecret string, accountID int64) {
	if m != nil && m.MediaStatus == domain.MediaStatusReady && m.MediaObjectKey != "" && publicAPIURL != "" && jwtSecret != "" {
		m.MediaStreamURL = storage.SignMediaURL(publicAPIURL, jwtSecret, accountID, m.ID, 2*time.Hour)
	}
}

// HandleWhatsAppLifecycleEvent processa eventos de conexão/desconexão e QR code.
// Retorna true se o evento foi tratado aqui.
func HandleWhatsAppLifecycleEvent(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, event string, data json.RawMessage) bool {
	switch event {
	case "Connected", "PairSuccess", "QRSuccess":
		handleEvolutionConnected(ctx, deps, inst, data)
		return true
	case "ProfileUpdated":
		handleEvolutionProfile(ctx, deps, inst, data)
		return true
	case "Disconnected", "LoggedOut", "ConnectFailure", "TemporaryBan":
		status := "DISCONNECTED"
		if event == "LoggedOut" {
			status = "CLOSED"
			_ = deps.WhatsApp.ResetInstanceSession(ctx, inst.ID, status)
		} else {
			_ = deps.WhatsApp.UpdateInstanceStatus(ctx, inst.ID, status)
		}
		wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
			"instance_id": inst.ID,
			"status":      status,
			"event":       event,
		})
		return true
	case "QRCode":
		var qr struct {
			Qrcode string `json:"qrcode"`
			Code   string `json:"code"`
			Count  int    `json:"count"`
		}
		if json.Unmarshal(data, &qr) == nil && qr.Qrcode != "" {
			wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_qr", map[string]any{
				"instance_id": inst.ID,
				"qrcode":      qr.Qrcode,
				"code":        qr.Code,
				"count":       qr.Count,
			})
		}
		return true
	case "QRTimeout":
		_ = deps.WhatsApp.UpdateInstanceStatus(ctx, inst.ID, "CLOSED")
		wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
			"instance_id": inst.ID,
			"status":      "CLOSED",
			"event":       "QRTimeout",
		})
		return true
	}
	return false
}

