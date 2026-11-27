package queue

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// extractWAHAEngineData extrai JIDs alternativos (@lid / telefone), PushName, CTWA e mídia
// do campo _data nas 4 engines da WAHA (GOWS, NOWEB, WEBJS e WPP).
func extractWAHAEngineData(rawData json.RawMessage, out *ParsedWhatsAppMessage) string {
	trimmed := bytes.TrimSpace(rawData)
	if len(trimmed) == 0 || string(trimmed) == "null" || trimmed[0] != '{' {
		return ""
	}
	var d struct {
		Info *struct {
			Chat           string `json:"Chat"`
			Sender         string `json:"Sender"`
			SenderAlt      string `json:"SenderAlt"`
			RecipientAlt   string `json:"RecipientAlt"`
			IsGroup        bool   `json:"IsGroup"`
			PushName       string `json:"PushName"`
			DeviceSentMeta *struct {
				DestinationJID string `json:"DestinationJID"`
			} `json:"DeviceSentMeta"`
		} `json:"Info"`
		GowsMessage map[string]json.RawMessage `json:"Message"`

		Key *struct {
			RemoteJID      string `json:"remoteJid"`
			RemoteJIDAlt   string `json:"remoteJidAlt"`
			Participant    string `json:"participant"`
			ParticipantAlt string `json:"participantAlt"`
		} `json:"key"`
		PushName     string                     `json:"pushName"`
		NowebMessage map[string]json.RawMessage `json:"message"`

		Type        string          `json:"type"`
		NotifyName  string          `json:"notifyName"`
		Caption     string          `json:"caption"`
		Mimetype    string          `json:"mimetype"`
		Filename    string          `json:"filename"`
		Size        json.RawMessage `json:"size"`
		Duration    json.RawMessage `json:"duration"`
		IsGroupMsg  bool            `json:"isGroupMsg"`
		CtwaContext *evoAdReferral  `json:"ctwaContext"`
		Referral    *evoAdReferral  `json:"referral"`
		SenderObj   *struct {
			PushName string `json:"pushname"`
			Name     string `json:"name"`
		} `json:"sender"`
		NestedData *struct {
			Type        string          `json:"type"`
			NotifyName  string          `json:"notifyName"`
			Caption     string          `json:"caption"`
			Mimetype    string          `json:"mimetype"`
			Filename    string          `json:"filename"`
			Size        json.RawMessage `json:"size"`
			Duration    json.RawMessage `json:"duration"`
			CtwaContext *evoAdReferral  `json:"ctwaContext"`
		} `json:"_data"`
	}
	if json.Unmarshal(trimmed, &d) != nil {
		return ""
	}
	if d.IsGroupMsg {
		out.IsGroup = true
	}

	if d.Info != nil {
		if d.Info.IsGroup {
			out.IsGroup = true
		}
		out.ChatJID = util.FirstNonEmpty(NormalizeWAHAJID(d.Info.Chat), out.ChatJID)
		out.SenderJID = util.FirstNonEmpty(NormalizeWAHAJID(d.Info.Sender), out.SenderJID)
		out.SenderAltJID = util.FirstNonEmpty(NormalizeWAHAJID(d.Info.SenderAlt), out.SenderAltJID)
		out.RecipientAltJID = util.FirstNonEmpty(NormalizeWAHAJID(d.Info.RecipientAlt), out.RecipientAltJID)
		if d.Info.DeviceSentMeta != nil {
			out.DeviceDestJID = util.FirstNonEmpty(NormalizeWAHAJID(d.Info.DeviceSentMeta.DestinationJID), out.DeviceDestJID)
		}
		if pn := strings.TrimSpace(d.Info.PushName); pn != "" && out.PushName == "" {
			out.PushName = pn
		}
	}
	if len(d.GowsMessage) > 0 {
		extractMessageContentAndMedia(unwrapMessageContainers(d.GowsMessage, out, 0), out)
	}

	if d.Key != nil {
		out.ChatJID = util.FirstNonEmpty(NormalizeWAHAJID(d.Key.RemoteJID), out.ChatJID)
		alt := NormalizeWAHAJID(util.FirstNonEmpty(d.Key.RemoteJIDAlt, d.Key.ParticipantAlt))
		if !out.IsFromMe {
			out.SenderAltJID = util.FirstNonEmpty(alt, out.SenderAltJID)
		} else {
			out.RecipientAltJID = util.FirstNonEmpty(alt, out.RecipientAltJID)
		}
	}
	if pn := strings.TrimSpace(d.PushName); pn != "" && out.PushName == "" {
		out.PushName = pn
	}
	if len(d.NowebMessage) > 0 {
		extractMessageContentAndMedia(unwrapMessageContainers(d.NowebMessage, out, 0), out)
	}

	var nestedNotify, nestedType, nestedCaption, nestedMime, nestedFilename string
	var nestedSize, nestedDuration json.RawMessage
	if d.NestedData != nil {
		nestedNotify = d.NestedData.NotifyName
		nestedType = d.NestedData.Type
		nestedCaption = d.NestedData.Caption
		nestedMime = d.NestedData.Mimetype
		nestedFilename = d.NestedData.Filename
		nestedSize = d.NestedData.Size
		nestedDuration = d.NestedData.Duration
		if d.NestedData.CtwaContext != nil {
			applyReferral(d.NestedData.CtwaContext, true, "", out)
		}
	}
	if d.CtwaContext != nil {
		applyReferral(d.CtwaContext, true, "", out)
	}
	if d.Referral != nil {
		applyReferral(d.Referral, true, "", out)
	}
	if out.PushName == "" {
		senderName := ""
		if d.SenderObj != nil {
			senderName = util.FirstNonEmpty(d.SenderObj.PushName, d.SenderObj.Name)
		}
		out.PushName = strings.TrimSpace(util.FirstNonEmpty(d.NotifyName, nestedNotify, senderName))
	}
	if out.Content == "" {
		out.Content = strings.TrimSpace(util.FirstNonEmpty(d.Caption, nestedCaption))
	}
	if out.MimeType == "" {
		out.MimeType = strings.TrimSpace(util.FirstNonEmpty(d.Mimetype, nestedMime))
	}
	if out.Filename == "" {
		out.Filename = strings.TrimSpace(util.FirstNonEmpty(d.Filename, nestedFilename))
	}
	if out.FileLength == 0 {
		out.FileLength = parseFlexibleInt64(d.Size)
		if out.FileLength == 0 {
			out.FileLength = parseFlexibleInt64(nestedSize)
		}
	}
	if out.DurationSec == 0 {
		out.DurationSec = int(parseFlexibleInt64(d.Duration))
		if out.DurationSec == 0 {
			out.DurationSec = int(parseFlexibleInt64(nestedDuration))
		}
	}
	return util.FirstNonEmpty(d.Type, nestedType)
}
