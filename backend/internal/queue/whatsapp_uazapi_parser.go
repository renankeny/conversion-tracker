package queue

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type UazapiWebhookMeta struct {
	EventType     string
	Owner         string
	Token         string
	BaseURL       string
	InstanceName  string
	ConnStatus    string
	ProfileName   string
	ProfilePicURL string
}

// UnwrapUazapiEnvelope desencapsula payloads enviados diretamente pela API Premium / Uazapi
// ou repassados via n8n/proxies nos formatos [{"body": {...}}] ou {"body": {...}}.
func UnwrapUazapiEnvelope(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return raw
	}
	if trimmed[0] == '[' {
		var arr []json.RawMessage
		if json.Unmarshal(trimmed, &arr) == nil && len(arr) > 0 {
			return UnwrapUazapiEnvelope(arr[0])
		}
	}
	var wrapper struct {
		EventType string          `json:"EventType"`
		Body      json.RawMessage `json:"body"`
	}
	if json.Unmarshal(trimmed, &wrapper) == nil && wrapper.EventType == "" && len(wrapper.Body) > 0 {
		return UnwrapUazapiEnvelope(wrapper.Body)
	}
	return trimmed
}

func ParseUazapiWebhookMeta(raw json.RawMessage) UazapiWebhookMeta {
	raw = UnwrapUazapiEnvelope(raw)
	var env struct {
		EventType    string `json:"EventType"`
		Owner        string `json:"owner"`
		Token        string `json:"token"`
		BaseURL      string `json:"BaseUrl"`
		InstanceName string `json:"instanceName"`
		Type         string `json:"type"`
		Instance     struct {
			Name          string `json:"name"`
			Status        string `json:"status"`
			ProfileName   string `json:"profileName"`
			ProfilePicURL string `json:"profilePicUrl"`
		} `json:"instance"`
	}
	_ = json.Unmarshal(raw, &env)
	return UazapiWebhookMeta{
		EventType:     strings.TrimSpace(env.EventType),
		Owner:         strings.TrimSpace(env.Owner),
		Token:         strings.TrimSpace(env.Token),
		BaseURL:       strings.TrimRight(strings.TrimSpace(env.BaseURL), "/"),
		InstanceName:  strings.TrimSpace(env.InstanceName),
		ConnStatus:    strings.ToLower(strings.TrimSpace(util.FirstNonEmpty(env.Instance.Status, env.Type))),
		ProfileName:   strings.TrimSpace(util.FirstNonEmpty(env.Instance.ProfileName, env.Instance.Name, env.InstanceName)),
		ProfilePicURL: strings.TrimSpace(env.Instance.ProfilePicURL),
	}
}

func IsUazapiIgnorableMessage(raw json.RawMessage) bool {
	raw = UnwrapUazapiEnvelope(raw)
	var quick struct {
		EventType string `json:"EventType"`
		Chat      struct {
			WaChatID  string `json:"wa_chatid"`
			WaIsGroup bool   `json:"wa_isGroup"`
		} `json:"chat"`
		Message struct {
			ChatID       string `json:"chatid"`
			Sender       string `json:"sender"`
			IsGroup      bool   `json:"isGroup"`
			IsStatus     bool   `json:"isStatus"`
			WasSentByAPI bool   `json:"wasSentByApi"`
			SendFunction string `json:"sendFunction"`
			MessageType  string `json:"messageType"`
			Type         string `json:"type"`
		} `json:"message"`
	}
	if json.Unmarshal(raw, &quick) != nil {
		return false
	}
	if ev := strings.ToLower(strings.TrimSpace(quick.EventType)); ev != "" && ev != "messages" && ev != "message" {
		return true
	}
	if quick.Message.WasSentByAPI || strings.TrimSpace(quick.Message.SendFunction) != "" {
		return true
	}
	if quick.Message.IsGroup || quick.Chat.WaIsGroup || quick.Message.IsStatus {
		return true
	}
	if isIgnorableUazapiMessageType(quick.Message.MessageType, quick.Message.Type) {
		return true
	}
	for _, c := range []string{quick.Message.ChatID, quick.Chat.WaChatID, quick.Message.Sender} {
		if IsIgnorableSpecialJID(c) {
			return true
		}
	}
	return false
}

func ParseUazapiMessageEvent(rawData json.RawMessage) (*ParsedWhatsAppMessage, error) {
	rawData = UnwrapUazapiEnvelope(rawData)
	if IsUazapiIgnorableMessage(rawData) {
		return nil, nil
	}

	var env struct {
		Owner string `json:"owner"`
		Chat  struct {
			Phone         string `json:"phone"`
			WaChatID      string `json:"wa_chatid"`
			WaChatLID     string `json:"wa_chatlid"`
			WaContactName string `json:"wa_contactName"`
			WaName        string `json:"wa_name"`
			Name          string `json:"name"`
			LeadFullName  string `json:"lead_fullName"`
			LeadName      string `json:"lead_name"`
			WaIsGroup     bool   `json:"wa_isGroup"`
		} `json:"chat"`
		Message struct {
			ID               string          `json:"id"`
			MessageID        string          `json:"messageid"`
			ChatID           string          `json:"chatid"`
			ChatLID          string          `json:"chatlid"`
			Sender           string          `json:"sender"`
			SenderLID        string          `json:"sender_lid"`
			SenderPN         string          `json:"sender_pn"`
			SenderName       string          `json:"senderName"`
			FromMe           bool            `json:"fromMe"`
			IsGroup          bool            `json:"isGroup"`
			WasSentByAPI     bool            `json:"wasSentByApi"`
			MessageTimestamp int64           `json:"messageTimestamp"`
			MessageType      string          `json:"messageType"`
			Type             string          `json:"type"`
			MediaType        string          `json:"mediaType"`
			Text             string          `json:"text"`
			Vote             string          `json:"vote"`
			ButtonOrListID   string          `json:"buttonOrListid"`
			FileURL          string          `json:"fileURL"`
			Content          json.RawMessage `json:"content"`
			Referral         *evoAdReferral  `json:"referral"`
		} `json:"message"`
	}
	if err := json.Unmarshal(rawData, &env); err != nil {
		return nil, err
	}
	msgID := strings.TrimSpace(env.Message.MessageID)
	if msgID == "" {
		msgID = stripOwnerPrefix(env.Message.ID)
	}
	if msgID == "" {
		return nil, nil
	}

	ownerBare := normalizeUazapiPhoneJID(env.Owner)
	if ownerBare != "" && (normalizeUazapiPhoneJID(env.Message.ChatID) == ownerBare ||
		normalizeUazapiPhoneJID(env.Chat.WaChatID) == ownerBare ||
		normalizeUazapiPhoneJID(env.Chat.Phone) == ownerBare) {
		return nil, nil
	}
	phoneJID := resolveUazapiPhoneJID(env.Message.FromMe, ownerBare, env.Message.SenderPN, env.Message.ChatID, env.Chat.WaChatID, env.Chat.Phone)
	lidJID := resolveUazapiLID(env.Message.FromMe, env.Message.ChatLID, env.Chat.WaChatLID, env.Message.SenderLID, env.Message.Sender)

	ts := time.Now()
	if env.Message.MessageTimestamp > 1e11 {
		ts = time.UnixMilli(env.Message.MessageTimestamp)
	} else if env.Message.MessageTimestamp > 0 {
		ts = time.Unix(env.Message.MessageTimestamp, 0)
	}

	pushName := cleanUazapiName(util.FirstNonEmpty(env.Chat.WaContactName, env.Chat.WaName, env.Chat.Name, env.Chat.LeadFullName, env.Chat.LeadName))
	if !env.Message.FromMe && cleanUazapiName(env.Message.SenderName) != "" {
		pushName = cleanUazapiName(env.Message.SenderName)
	}

	out := &ParsedWhatsAppMessage{
		MessageID:    msgID,
		RawMessageID: strings.TrimSpace(env.Message.ID),
		ChatJID:      util.FirstNonEmpty(phoneJID, lidJID, NormalizeBareJID(env.Message.ChatID), NormalizeBareJID(env.Chat.WaChatID)),
		IsFromMe:     env.Message.FromMe,
		IsGroup:      false,
		PushName:     pushName,
		Timestamp:    ts,
		MessageType:  "text",
		Content:      strings.TrimSpace(util.FirstNonEmpty(env.Message.Text, env.Message.Vote, env.Message.ButtonOrListID)),
	}
	if !env.Message.FromMe {
		out.SenderJID = util.FirstNonEmpty(phoneJID, NormalizeBareJID(env.Message.Sender))
		out.SenderAltJID = lidJID
	} else {
		out.RecipientAltJID = phoneJID
		out.DeviceDestJID = util.FirstNonEmpty(lidJID, phoneJID)
	}

	extractUazapiContentAndMedia(env.Message.MessageType, env.Message.MediaType, env.Message.Type, env.Message.FileURL, env.Message.Content, out)
	if env.Message.Referral != nil {
		applyReferral(env.Message.Referral, false, "", out)
	}
	sanitizeParsedMessageDirection(out)
	if out.Content == "" && !out.HasMedia && out.CtwaClid == "" && out.SourceID == "" {
		return nil, nil
	}
	return out, nil
}
