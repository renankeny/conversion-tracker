package queue

import (
	"encoding/json"
	"testing"
)

func TestParseUazapiMessageEvent_AudioVoicePTT(t *testing.T) {
	raw := json.RawMessage(`{
		"EventType": "messages",
		"owner": "551832653991",
		"chat": {
			"phone": "5518997463184",
			"wa_chatid": "5518997463184@s.whatsapp.net",
			"wa_name": "Maria Silva"
		},
		"message": {
			"id": "551832653991:3EB0AUDIO001",
			"messageid": "3EB0AUDIO001",
			"chatid": "5518997463184@s.whatsapp.net",
			"fromMe": false,
			"isGroup": false,
			"wasSentByApi": false,
			"messageTimestamp": 1794348227000,
			"messageType": "AudioMessage",
			"mediaType": "ptt",
			"type": "media",
			"text": "",
			"content": {
				"URL": "https://mmg.whatsapp.net/v/t62.7117-24/123456.enc?ccb=11-4",
				"mimetype": "audio/ogg; codecs=opus",
				"fileLength": "48210",
				"seconds": 14,
				"PTT": true
			}
		}
	}`)

	parsed, err := ParseUazapiMessageEvent(raw)
	if err != nil || parsed == nil {
		t.Fatalf("expected audio message to be parsed, got err=%v", err)
	}
	if !parsed.HasMedia || parsed.MessageType != "audio" {
		t.Fatalf("expected HasMedia=true and MessageType=audio, got HasMedia=%v type=%q", parsed.HasMedia, parsed.MessageType)
	}
	if parsed.MessageID != "3EB0AUDIO001" || parsed.RawMessageID != "551832653991:3EB0AUDIO001" {
		t.Fatalf("unexpected IDs: msg=%q raw=%q", parsed.MessageID, parsed.RawMessageID)
	}
	if parsed.DurationSec != 14 || parsed.FileLength != 48210 {
		t.Fatalf("unexpected audio metadata: duration=%d len=%d", parsed.DurationSec, parsed.FileLength)
	}
	if parsed.ExternalMediaURL != "" {
		t.Fatalf("expected encrypted mmg.whatsapp.net URL to be excluded from ExternalMediaURL, got %q", parsed.ExternalMediaURL)
	}
}

func TestParseUazapiMessageEvent_PhotoVideoAndDocument(t *testing.T) {
	cases := []struct {
		name        string
		payload     string
		wantType    string
		wantCaption string
		wantFile    string
		wantExtURL  string
	}{
		{
			name: "captionless photo with public fileURL",
			payload: `{
				"EventType": "messages",
				"owner": "551832653991",
				"message": {
					"id": "551832653991:IMG01",
					"messageid": "IMG01",
					"chatid": "5518997463184@s.whatsapp.net",
					"messageType": "ImageMessage",
					"mediaType": "image",
					"type": "media",
					"fileURL": "https://rkagencia.uazapi.com/files/foto1.jpg",
					"content": { "mimetype": "image/jpeg", "fileLength": 204800 }
				}
			}`,
			wantType:   "image",
			wantExtURL: "https://rkagencia.uazapi.com/files/foto1.jpg",
		},
		{
			name: "video message with caption",
			payload: `{
				"EventType": "messages",
				"owner": "551832653991",
				"message": {
					"id": "551832653991:VID01",
					"messageid": "VID01",
					"chatid": "5518997463184@s.whatsapp.net",
					"messageType": "VideoMessage",
					"mediaType": "video",
					"type": "media",
					"text": "Olha como ficou a instalação",
					"content": {
						"URL": "https://mmg.whatsapp.net/o1/v/t62.7161-24/video.enc",
						"mimetype": "video/mp4",
						"seconds": 23,
						"caption": "Olha como ficou a instalação"
					}
				}
			}`,
			wantType:    "video",
			wantCaption: "Olha como ficou a instalação",
		},
		{
			name: "nested documentWithCaptionMessage PDF",
			payload: `{
				"EventType": "messages",
				"owner": "551832653991",
				"message": {
					"id": "551832653991:DOC01",
					"messageid": "DOC01",
					"chatid": "5518997463184@s.whatsapp.net",
					"messageType": "DocumentWithCaptionMessage",
					"mediaType": "document",
					"type": "media",
					"content": {
						"message": {
							"documentMessage": {
								"URL": "https://mmg.whatsapp.net/v/t62.7119-24/doc.enc",
								"mimetype": "application/pdf",
								"fileName": "comprovante_pix.pdf",
								"caption": "Segue o comprovante",
								"fileLength": 98304
							}
						}
					}
				}
			}`,
			wantType:    "document",
			wantCaption: "Segue o comprovante",
			wantFile:    "comprovante_pix.pdf",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseUazapiMessageEvent(json.RawMessage(tc.payload))
			if err != nil || parsed == nil {
				t.Fatalf("expected parsed media message, got err=%v", err)
			}
			if !parsed.HasMedia || parsed.MessageType != tc.wantType {
				t.Fatalf("want type=%s HasMedia=true, got type=%s HasMedia=%v", tc.wantType, parsed.MessageType, parsed.HasMedia)
			}
			if parsed.Content != tc.wantCaption {
				t.Fatalf("want caption=%q, got %q", tc.wantCaption, parsed.Content)
			}
			if parsed.Filename != tc.wantFile {
				t.Fatalf("want filename=%q, got %q", tc.wantFile, parsed.Filename)
			}
			if parsed.ExternalMediaURL != tc.wantExtURL {
				t.Fatalf("want extURL=%q, got %q", tc.wantExtURL, parsed.ExternalMediaURL)
			}
		})
	}
}

func TestExtensionForMime_SupportsMP3AndDocuments(t *testing.T) {
	if got := extensionForMime("audio/mpeg", "audio"); got != ".mp3" {
		t.Fatalf("expected .mp3 for audio/mpeg, got %s", got)
	}
	if got := extensionForMime("audio/ogg; codecs=opus", "audio"); got != ".ogg" {
		t.Fatalf("expected .ogg for audio/ogg, got %s", got)
	}
	if got := extensionForMime("application/pdf", "document"); got != ".pdf" {
		t.Fatalf("expected .pdf for application/pdf, got %s", got)
	}
	if got := extensionForMime("video/mp4", "video"); got != ".mp4" {
		t.Fatalf("expected .mp4 for video/mp4, got %s", got)
	}
}
