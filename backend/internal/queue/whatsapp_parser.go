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
	HasMsgAdReferral bool
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
	CtwaClid      string `json:"ctwaClid"`
	CtwaClidSnake string `json:"ctwa_clid"`
	SourceID      string `json:"sourceID"`
	SourceIdAlt   string `json:"sourceId"`
	SourceIDSnake string `json:"source_id"`
	SourceURL     string `json:"sourceURL"`
	SourceUrlAlt  string `json:"sourceUrl"`
	SourceApp     string `json:"sourceApp"`
	SourceAppAlt  string `json:"source_app"`
	ThumbnailURL     string `json:"thumbnailURL"`
	OriginalImageURL string `json:"originalImageURL"`
	MediaURL         string `json:"mediaURL"`
}

type evoContextInfo struct {
	ConversionApp   string        `json:"entryPointConversionApp"`
	ConversionSrc   string        `json:"entryPointConversionSource"`
	ConversionFB    string        `json:"conversionSource"`
	CtwaClidDirect  string        `json:"ctwaClid"`
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

	extractMessageContentAndMedia(unwrapMessageContainers(env.Message, out, 0), out)
	if env.Referral != nil {
		applyReferral(env.Referral, "", out)
	}
	if out.Content == "" && !out.HasMedia && out.CtwaClid == "" && out.SourceID == "" {
		return nil, nil
	}
	return out, nil
}

// unwrapMessageContainers desencapsula envelopes aninhados do WhatsApp (mensagens temporárias
// ephemeralMessage, viewOnceMessage, deviceSentMessage, documentWithCaptionMessage, editedMessage).
func unwrapMessageContainers(msgMap map[string]json.RawMessage, out *ParsedWhatsAppMessage, depth int) map[string]json.RawMessage {
	if len(msgMap) == 0 || depth > 4 {
		return msgMap
	}
	if rawURL, ok := msgMap["mediaUrl"]; ok && out.ExternalMediaURL == "" {
		_ = json.Unmarshal(rawURL, &out.ExternalMediaURL)
	}
	if rawCtx, ok := msgMap["contextInfo"]; ok {
		var ctxInfo evoContextInfo
		if json.Unmarshal(rawCtx, &ctxInfo) == nil {
			applyContextInfo(&ctxInfo, out)
		}
	}
	containerKeys := []string{
		"ephemeralMessage", "viewOnceMessage", "viewOnceMessageV2",
		"viewOnceMessageV2Extension", "deviceSentMessage",
		"documentWithCaptionMessage", "editedMessage",
	}
	for _, key := range containerKeys {
		rawWrap, ok := msgMap[key]
		if !ok {
			continue
		}
		var wrap struct {
			DestinationJID  string                     `json:"destinationJID"`
			Message         map[string]json.RawMessage `json:"message"`
			ProtocolMessage *struct {
				EditedMessage map[string]json.RawMessage `json:"editedMessage"`
			} `json:"protocolMessage"`
		}
		if json.Unmarshal(rawWrap, &wrap) == nil {
			if wrap.DestinationJID != "" && out.DeviceDestJID == "" {
				out.DeviceDestJID = wrap.DestinationJID
			}
			if len(wrap.Message) > 0 {
				return unwrapMessageContainers(wrap.Message, out, depth+1)
			}
			if wrap.ProtocolMessage != nil && len(wrap.ProtocolMessage.EditedMessage) > 0 {
				return unwrapMessageContainers(wrap.ProtocolMessage.EditedMessage, out, depth+1)
			}
		}
	}
	return msgMap
}

func extractMessageContentAndMedia(msgMap map[string]json.RawMessage, out *ParsedWhatsAppMessage) {
	if len(msgMap) == 0 {
		return
	}
	if rawURL, ok := msgMap["mediaUrl"]; ok && out.ExternalMediaURL == "" {
		_ = json.Unmarshal(rawURL, &out.ExternalMediaURL)
	}
	if rawConv, ok := msgMap["conversation"]; ok {
		var s string
		if json.Unmarshal(rawConv, &s) == nil && s != "" {
			out.Content = s
		}
	}
	for _, key := range []string{"extendedTextMessage", "buttonsResponseMessage", "listResponseMessage", "templateButtonReplyMessage"} {
		if rawExt, ok := msgMap[key]; ok {
			var ext struct {
				Text                string          `json:"text"`
				SelectedDisplayText string          `json:"selectedDisplayText"`
				Title               string          `json:"title"`
				Description         string          `json:"description"`
				ContextInfo         *evoContextInfo `json:"contextInfo"`
			}
			if json.Unmarshal(rawExt, &ext) == nil {
				txt := util.FirstNonEmpty(ext.Text, ext.SelectedDisplayText, ext.Title, ext.Description)
				if txt != "" && out.Content == "" {
					out.Content = txt
				}
				applyContextInfo(ext.ContextInfo, out)
			}
		}
	}
	extractMediaSubMessage(msgMap, out)
}
