package queue

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

type wahaWebhookEnvelope struct {
	Event   string          `json:"event"`
	Session string          `json:"session"`
	Engine  string          `json:"engine"`
	Me      *wahaMeInfo     `json:"me"`
	Payload json.RawMessage `json:"payload"`
}

type wahaMeInfo struct {
	ID       string `json:"id"`
	LID      string `json:"lid"`
	JID      string `json:"jid"`
	PushName string `json:"pushName"`
}

type wahaMediaPayload struct {
	URL      string `json:"url"`
	Mimetype string `json:"mimetype"`
	Filename string `json:"filename"`
}

type wahaMessagePayload struct {
	ID          string            `json:"id"`
	Timestamp   int64             `json:"timestamp"`
	From        string            `json:"from"`
	To          string            `json:"to"`
	FromMe      bool              `json:"fromMe"`
	Source      string            `json:"source"`
	Participant string            `json:"participant"`
	Body        string            `json:"body"`
	HasMedia    bool              `json:"hasMedia"`
	Media       *wahaMediaPayload `json:"media"`
	MediaURL    string            `json:"mediaUrl"`
	Data        json.RawMessage   `json:"_data"`
}

// NormalizeWAHAJID converte @c.us para @s.whatsapp.net mantendo @lid e removendo sufixos de device (:14).
func NormalizeWAHAJID(raw string) string {
	bare := NormalizeBareJID(raw)
	if bare == "" {
		return ""
	}
	if strings.HasSuffix(bare, "@c.us") {
		return strings.TrimSuffix(bare, "@c.us") + "@s.whatsapp.net"
	}
	return bare
}

// extractWAHAMessageID extrai o stanza ID limpo de IDs serializados WAHA (ex: false_5511...@c.us_ABC123).
func extractWAHAMessageID(rawID string) string {
	s := strings.TrimSpace(rawID)
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "_")
	if len(parts) >= 3 && (parts[0] == "true" || parts[0] == "false") && strings.Contains(parts[1], "@") {
		if id := strings.TrimSpace(parts[2]); id != "" {
			return id
		}
	}
	return s
}

func IsWAHAIgnorableMessage(raw json.RawMessage) bool {
	var env wahaWebhookEnvelope
	if json.Unmarshal(bytes.TrimSpace(raw), &env) != nil {
		return false
	}
	ev := strings.ToLower(strings.TrimSpace(env.Event))
	if ev != "" && ev != "message.any" && ev != "message" {
		return true
	}
	var msg wahaMessagePayload
	if json.Unmarshal(env.Payload, &msg) != nil {
		return false
	}
	peer := msg.From
	if msg.FromMe && strings.TrimSpace(msg.To) != "" {
		peer = msg.To
	}
	if IsIgnorableSpecialJID(peer) || IsIgnorableSpecialJID(msg.From) || IsIgnorableSpecialJID(msg.To) {
		return true
	}
	return isWAHAIgnorableRawData(msg.Data)
}

func ParseWAHAMessageEvent(rawData json.RawMessage) (*ParsedWhatsAppMessage, error) {
	if IsWAHAIgnorableMessage(rawData) {
		return nil, nil
	}
	var env wahaWebhookEnvelope
	if err := json.Unmarshal(bytes.TrimSpace(rawData), &env); err != nil {
		return nil, err
	}
	var msg wahaMessagePayload
	if err := json.Unmarshal(env.Payload, &msg); err != nil {
		return nil, err
	}
	msgID := extractWAHAMessageID(msg.ID)
	if msgID == "" {
		return nil, nil
	}

	ts := time.Now()
	if msg.Timestamp > 1e11 {
		ts = time.UnixMilli(msg.Timestamp)
	} else if msg.Timestamp > 0 {
		ts = time.Unix(msg.Timestamp, 0)
	}

	rawChatID := strings.TrimSpace(msg.From)
	if msg.FromMe && strings.TrimSpace(msg.To) != "" {
		rawChatID = strings.TrimSpace(msg.To)
	}
	peerJID := NormalizeWAHAJID(rawChatID)
	if IsIgnorableSpecialJID(peerJID) {
		return nil, nil
	}

	out := &ParsedWhatsAppMessage{
		MessageID:    msgID,
		RawMessageID: strings.TrimSpace(msg.ID),
		RawChatID:    rawChatID,
		ChatJID:      peerJID,
		IsFromMe:     msg.FromMe,
		Timestamp:    ts,
		MessageType:  "text",
		Content:      strings.TrimSpace(msg.Body),
	}
	if !msg.FromMe {
		out.SenderJID = peerJID
	} else {
		out.DeviceDestJID = peerJID
	}

	// Preenche ExternalMediaURL e HasMedia antes do parser _data para que extractMediaSubMessage
	// sempre processe sub-mensagens de mídia (e seus contextInfo/externalAdReply de CTWA).
	preloadWAHAMediaFields(&msg, out)
	rawDataType := extractWAHAEngineData(msg.Data, out)
	applyWAHAMediaMetadata(&msg, rawDataType, out)
	sanitizeParsedMessageDirection(out)

	if out.IsGroup || IsIgnorableSpecialJID(out.ChatJID) || IsIgnorableSpecialJID(out.SenderJID) {
		return nil, nil
	}
	if out.Content == "" && !out.HasMedia && out.CtwaClid == "" && out.SourceID == "" {
		return nil, nil
	}
	return out, nil
}
