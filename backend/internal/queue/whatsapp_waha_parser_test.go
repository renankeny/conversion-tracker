package queue

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseWAHAMessageEventGOWSCTWAAndImage(t *testing.T) {
	raw := json.RawMessage(`{
		"event": "message.any",
		"session": "cli-1",
		"engine": "GOWS",
		"payload": {
			"id": "false_5511988887777@c.us_3EB0123456789",
			"timestamp": 1732700000,
			"from": "5511988887777@c.us",
			"to": "5511911112222@c.us",
			"fromMe": false,
			"body": "Quero saber o valor",
			"hasMedia": true,
			"media": {
				"url": "http://localhost:3000/api/files/cli-1/img1.jpg",
				"mimetype": "image/jpeg",
				"filename": "foto.jpg"
			},
			"_data": {
				"Info": {
					"Chat": "5511988887777@s.whatsapp.net",
					"Sender": "12345678901234@lid",
					"SenderAlt": "5511988887777@s.whatsapp.net",
					"PushName": "Maria Silva",
					"IsGroup": false
				},
				"Message": {
					"imageMessage": {
						"caption": "Quero saber o valor",
						"mimetype": "image/jpeg",
						"fileLength": 54321,
						"contextInfo": {
							"entryPointConversionSource": "ctwa_ad",
							"entryPointConversionApp": "instagram",
							"externalAdReply": {
								"sourceType": "ad",
								"sourceID": "120210998877665544",
								"ctwaClid": "AfZ1234567890abcdef"
							}
						}
					}
				}
			}
		}
	}`)

	parsed, err := ParseWAHAMessageEvent(raw)
	if err != nil || parsed == nil {
		t.Fatalf("expected parsed message, got parsed=%+v err=%v", parsed, err)
	}
	if parsed.MessageID != "3EB0123456789" {
		t.Fatalf("expected clean stanza id, got %q", parsed.MessageID)
	}
	if parsed.RawMessageID != "false_5511988887777@c.us_3EB0123456789" {
		t.Fatalf("expected raw id preserved, got %q", parsed.RawMessageID)
	}
	if parsed.PushName != "Maria Silva" {
		t.Fatalf("expected PushName Maria Silva, got %q", parsed.PushName)
	}
	if !parsed.HasMedia || parsed.MessageType != "image" || parsed.ExternalMediaURL != "http://localhost:3000/api/files/cli-1/img1.jpg" {
		t.Fatalf("unexpected media fields: has=%v type=%q url=%q", parsed.HasMedia, parsed.MessageType, parsed.ExternalMediaURL)
	}
	if parsed.CtwaClid != "AfZ1234567890abcdef" || parsed.SourceID != "120210998877665544" || parsed.SourceApp != "instagram" {
		t.Fatalf("unexpected CTWA fields: clid=%q sid=%q app=%q", parsed.CtwaClid, parsed.SourceID, parsed.SourceApp)
	}
	if parsed.SenderAltJID != "5511988887777@s.whatsapp.net" || parsed.SenderJID != "12345678901234@lid" {
		t.Fatalf("unexpected sender/lid fields: sender=%q alt=%q", parsed.SenderJID, parsed.SenderAltJID)
	}
}

func TestParseWAHAMessageEventNowebAndWebjsAndFilters(t *testing.T) {
	// 1. WEBJS audio PTT + source=api outgoing message (deve ser aceita!)
	webjsOut := json.RawMessage(`{
		"event": "message.any",
		"session": "cli-1",
		"engine": "WEBJS",
		"payload": {
			"id": "true_5511977776666@c.us_ABCD999",
			"timestamp": 1732700100,
			"from": "5511911112222@c.us",
			"to": "5511977776666@c.us",
			"fromMe": true,
			"source": "api",
			"body": "Pagamento confirmado!",
			"hasMedia": true,
			"media": {
				"url": "https://waha.example/api/files/cli-1/voice.ogg",
				"mimetype": "audio/ogg; codecs=opus"
			},
			"_data": {
				"type": "ptt",
				"duration": "12"
			}
		}
	}`)
	parsed, err := ParseWAHAMessageEvent(webjsOut)
	if err != nil || parsed == nil {
		t.Fatalf("expected outgoing api message accepted, got %+v err=%v", parsed, err)
	}
	if !parsed.IsFromMe || parsed.ChatJID != "5511977776666@s.whatsapp.net" || parsed.MessageType != "audio" || parsed.DurationSec != 12 {
		t.Fatalf("unexpected webjs parsed: %+v", parsed)
	}

	// 2. Grupo deve ser ignorado
	groupRaw := json.RawMessage(`{
		"event": "message.any",
		"payload": {
			"id": "false_120363025@g.us_XYZ",
			"from": "120363025@g.us",
			"fromMe": false,
			"body": "oi grupo"
		}
	}`)
	if !IsWAHAIgnorableMessage(groupRaw) {
		t.Fatal("expected group message to be ignored")
	}

	// 3. StripHeavyWAHAThumbnails remove JPEGThumbnail grande
	heavy := json.RawMessage(`{"event":"message.any","payload":{"_data":{"Message":{"imageMessage":{"JPEGThumbnail":"` + strings.Repeat("A", 400) + `","caption":"oi"}}}}}`)
	cleaned := string(StripHeavyWAHAThumbnails(heavy))
	if strings.Contains(cleaned, "JPEGThumbnail") {
		t.Fatalf("expected heavy JPEGThumbnail stripped, got %s", cleaned)
	}
}
