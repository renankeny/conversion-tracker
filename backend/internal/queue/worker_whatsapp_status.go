package queue

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func handleEvolutionConnected(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, raw json.RawMessage) {
	var d struct {
		JID      string `json:"jid"`
		PushName string `json:"pushName"`
	}
	_ = json.Unmarshal(raw, &d)
	ownerBare := NormalizeBareJID(d.JID)
	_ = deps.WhatsApp.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, strings.TrimSpace(d.PushName), "")
	wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "whatsapp_status", map[string]any{
		"instance_id":  inst.ID,
		"status":       "OPEN",
		"owner_jid":    ownerBare,
		"profile_name": strings.TrimSpace(d.PushName),
	})
}

func handleEvolutionProfile(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, raw json.RawMessage) {
	var d struct {
		ProfileName   string `json:"profileName"`
		ProfilePicURL string `json:"profilePicUrl"`
	}
	if json.Unmarshal(raw, &d) == nil {
		_ = deps.WhatsApp.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "", "", d.ProfileName, d.ProfilePicURL)
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

