package queue

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type ParsedWhatsAppMessage struct {
	MessageID        string
	ChatJID          string
	SenderJID        string
	SenderAltJID     string
	RecipientAltJID  string
	DeviceDestJID    string
	IsFromMe         bool
	IsGroup          bool
	PushName         string
	Timestamp        time.Time
	MessageType      string
	Content          string
	HasMedia         bool
	MimeType         string
	FileLength       int64
	DurationSec      int
	Filename         string
	ExternalMediaURL string
	MediaProtoJSON   json.RawMessage
	CtwaClid         string
	SourceID         string
	SourceURL        string
	SourceApp        string
	AdThumbnailURL   string
}

// StripHeavyBase64FromEventData remove data.Message.base64 antes de gravar no Redis (256MB cap).
func StripHeavyBase64FromEventData(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || !strings.Contains(string(raw), `"base64"`) {
		return raw
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return raw
	}
	msgBytes, ok := root["Message"]
	if !ok {
		return raw
	}
	var msgMap map[string]json.RawMessage
	if err := json.Unmarshal(msgBytes, &msgMap); err != nil {
		return raw
	}
	if _, hasB64 := msgMap["base64"]; !hasB64 {
		return raw
	}
	delete(msgMap, "base64")
	cleanedMsg, err := json.Marshal(msgMap)
	if err != nil {
		return raw
	}
	root["Message"] = cleanedMsg
	cleanedRoot, err := json.Marshal(root)
	if err != nil {
		return raw
	}
	return cleanedRoot
}

type evoAdReferral struct {
	CtwaClid     string `json:"ctwaClid"`
	SourceID     string `json:"sourceID"`
	SourceIdAlt  string `json:"sourceId"`
	SourceURL    string `json:"sourceURL"`
	SourceApp    string `json:"sourceApp"`
	ThumbnailURL string `json:"thumbnailURL"`
	MediaURL     string `json:"mediaURL"`
}

type evoContextInfo struct {
	ConversionApp   string        `json:"entryPointConversionApp"`
	ExternalAdReply evoAdReferral `json:"externalAdReply"`
}

func ParseEvolutionMessageEvent(rawData json.RawMessage) (*ParsedWhatsAppMessage, error) {
	var env struct {
		Info struct {
			Chat           string `json:"Chat"`
			Sender         string `json:"Sender"`
			SenderAlt      string `json:"SenderAlt"`
			RecipientAlt   string `json:"RecipientAlt"`
			IsFromMe       bool   `json:"IsFromMe"`
			IsGroup        bool   `json:"IsGroup"`
			ID             string `json:"ID"`
			PushName       string `json:"PushName"`
			Timestamp      string `json:"Timestamp"`
			DeviceSentMeta *struct {
				DestinationJID string `json:"DestinationJID"`
			} `json:"DeviceSentMeta"`
		} `json:"Info"`
		Message  map[string]json.RawMessage `json:"Message"`
		Referral *evoAdReferral             `json:"referral"`
	}
	if err := json.Unmarshal(rawData, &env); err != nil {
		return nil, err
	}
	if env.Info.ID == "" {
		return nil, nil
	}

	ts := time.Now()
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if env.Info.Timestamp != "" {
			if parsed, err := time.Parse(layout, env.Info.Timestamp); err == nil {
				ts = parsed
				break
			}
		}
	}

	out := &ParsedWhatsAppMessage{
		MessageID:       env.Info.ID,
		ChatJID:         env.Info.Chat,
		SenderJID:       env.Info.Sender,
		SenderAltJID:    env.Info.SenderAlt,
		RecipientAltJID: env.Info.RecipientAlt,
		IsFromMe:        env.Info.IsFromMe,
		IsGroup:         env.Info.IsGroup,
		PushName:        strings.TrimSpace(env.Info.PushName),
		Timestamp:       ts,
		MessageType:     "text",
	}
	if env.Info.DeviceSentMeta != nil {
		out.DeviceDestJID = env.Info.DeviceSentMeta.DestinationJID
	}

	extractMessageContentAndMedia(env.Message, out)
	if env.Referral != nil {
		applyReferral(env.Referral, "", out)
	}
	if out.Content == "" && !out.HasMedia && out.CtwaClid == "" && out.SourceID == "" {
		return nil, nil
	}
	return out, nil
}

func extractMessageContentAndMedia(msgMap map[string]json.RawMessage, out *ParsedWhatsAppMessage) {
	if len(msgMap) == 0 {
		return
	}
	if rawURL, ok := msgMap["mediaUrl"]; ok {
		_ = json.Unmarshal(rawURL, &out.ExternalMediaURL)
	}
	if rawConv, ok := msgMap["conversation"]; ok {
		var s string
		if json.Unmarshal(rawConv, &s) == nil && s != "" {
			out.Content = s
		}
	}
	if rawExt, ok := msgMap["extendedTextMessage"]; ok {
		var ext struct {
			Text        string          `json:"text"`
			ContextInfo *evoContextInfo `json:"contextInfo"`
		}
		if json.Unmarshal(rawExt, &ext) == nil {
			if ext.Text != "" {
				out.Content = ext.Text
			}
			if ext.ContextInfo != nil {
				applyReferral(&ext.ContextInfo.ExternalAdReply, ext.ContextInfo.ConversionApp, out)
			}
		}
	}
	extractMediaSubMessage(msgMap, out)
}

func extractMediaSubMessage(msgMap map[string]json.RawMessage, out *ParsedWhatsAppMessage) {
	mediaKeys := [][2]string{
		{"audioMessage", "audio"}, {"imageMessage", "image"},
		{"documentMessage", "document"}, {"videoMessage", "video"}, {"stickerMessage", "sticker"},
	}
	for _, mk := range mediaKeys {
		rawSub, ok := msgMap[mk[0]]
		if !ok {
			continue
		}
		var sub struct {
			URL         string          `json:"URL"`
			UrlLower    string          `json:"url"`
			Mimetype    string          `json:"mimetype"`
			Caption     string          `json:"caption"`
			FileLength  uint64          `json:"fileLength"`
			Seconds     uint32          `json:"seconds"`
			FileName    string          `json:"fileName"`
			Title       string          `json:"title"`
			ContextInfo *evoContextInfo `json:"contextInfo"`
		}
		if json.Unmarshal(rawSub, &sub) == nil && (sub.URL != "" || sub.UrlLower != "" || out.ExternalMediaURL != "") {
			out.MessageType = mk[1]
			out.HasMedia = true
			out.MimeType = sub.Mimetype
			out.FileLength = int64(sub.FileLength)
			out.DurationSec = int(sub.Seconds)
			out.Filename = util.FirstNonEmpty(sub.FileName, sub.Title)
			if sub.Caption != "" && out.Content == "" {
				out.Content = sub.Caption
			}
			if sub.ContextInfo != nil {
				applyReferral(&sub.ContextInfo.ExternalAdReply, sub.ContextInfo.ConversionApp, out)
			}
			protoEnvelope, _ := json.Marshal(map[string]json.RawMessage{mk[0]: rawSub})
			out.MediaProtoJSON = protoEnvelope
			return
		}
	}
}
