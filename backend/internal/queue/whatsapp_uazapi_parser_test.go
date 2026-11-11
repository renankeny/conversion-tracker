package queue

import (
	"encoding/json"
	"testing"
)

func TestParseUazapiMessageEvent_RealCTWAPayload(t *testing.T) {
	raw := json.RawMessage(`[
	  {
	    "headers": {
	      "user-agent": "uazapiGO-Webhook/1.0"
	    },
	    "body": {
	      "BaseUrl": "https://rkagencia.uazapi.com",
	      "EventType": "messages",
	      "chat": {
	        "id": "ra76dc5d86ed789",
	        "owner": "551832653991",
	        "phone": "5518997463184",
	        "wa_chatid": "5518997463184@s.whatsapp.net",
	        "wa_chatlid": "210702790877292@lid",
	        "wa_isGroup": false,
	        "wa_name": "João Pedro Carneiro \""
	      },
	      "instanceName": "id2-WSOM RANCHARIA",
	      "message": {
	        "chatid": "5518997463184@s.whatsapp.net",
	        "chatlid": "210702790877292@lid",
	        "content": {
	          "text": "Olá gostaria de saber como funciona o Wsom TV",
	          "contextInfo": {
	            "conversionSource": "FB_Ads",
	            "entryPointConversionSource": "ctwa_ad",
	            "entryPointConversionApp": "instagram",
	            "externalAdReply": {
	              "title": "Converse conosco",
	              "thumbnailURL": "https://scontent-gru1-2.cdninstagram.com/thumb.jpg",
	              "sourceType": "ad",
	              "sourceID": "120259519567440386",
	              "sourceURL": "https://www.instagram.com/p/Dd9NlTVMrFh/",
	              "ctwaClid": "AfiOv5AdFoJPcE34i-akqQjN1ZFswJmqhCcGrWkkQf7LMRfOzXwNkHyEIm9IttKud9mfNjJ0wNeOiyp9ELk4qZoIW5DzefG0o5Brs5vo7bUR1pmEltQqSL6o56hXTsl5hUyY6CTwQQc",
	              "sourceApp": "instagram"
	            }
	          }
	        },
	        "fromMe": false,
	        "id": "551832653991:AC5E9B7AC6D373A842728D3E4E182F06",
	        "isGroup": false,
	        "messageTimestamp": 1794348227000,
	        "messageType": "ExtendedTextMessage",
	        "messageid": "AC5E9B7AC6D373A842728D3E4E182F06",
	        "owner": "551832653991",
	        "sender": "210702790877292@lid",
	        "senderName": "João Pedro Carneiro \"",
	        "sender_lid": "210702790877292@lid",
	        "sender_pn": "5518997463184@s.whatsapp.net",
	        "text": "Olá gostaria de saber como funciona o Wsom TV",
	        "type": "text",
	        "wasSentByApi": false
	      },
	      "owner": "551832653991",
	      "token": "642e3745-716c-4751-9224-6d48d5ee58a5"
	    }
	  }
	]`)

	meta := ParseUazapiWebhookMeta(raw)
	if meta.EventType != "messages" || meta.Owner != "551832653991" || meta.BaseURL != "https://rkagencia.uazapi.com" {
		t.Fatalf("unexpected meta: %+v", meta)
	}

	parsed, err := ParseUazapiMessageEvent(raw)
	if err != nil || parsed == nil {
		t.Fatalf("expected parsed message, got err=%v", err)
	}
	if parsed.MessageID != "AC5E9B7AC6D373A842728D3E4E182F06" {
		t.Fatalf("unexpected MessageID: %s", parsed.MessageID)
	}
	if parsed.PushName != "João Pedro Carneiro" {
		t.Fatalf("unexpected PushName: %q", parsed.PushName)
	}
	if parsed.SourceID != "120259519567440386" || parsed.SourceApp != "instagram" {
		t.Fatalf("unexpected CTWA source: id=%s app=%s", parsed.SourceID, parsed.SourceApp)
	}
	if parsed.CtwaClid == "" {
		t.Fatal("expected non-empty CtwaClid")
	}

	var savedLID, savedPhone string
	remoteJID, phone, lid, skip := ResolveContactJID(
		parsed,
		"551832653991@s.whatsapp.net",
		nil,
		func(l, p string) { savedLID = l; savedPhone = p },
	)
	if skip {
		t.Fatal("expected skip=false")
	}
	if remoteJID != "5518997463184@s.whatsapp.net" || phone != "5518997463184" || lid != "210702790877292" {
		t.Fatalf("unexpected contact resolution: remote=%s phone=%s lid=%s", remoteJID, phone, lid)
	}
	if savedLID != "210702790877292@lid" || savedPhone != "5518997463184@s.whatsapp.net" {
		t.Fatalf("expected LID mapping saved, got lid=%s phone=%s", savedLID, savedPhone)
	}
}

func TestParseUazapiMessageEvent_IgnoresAPISentAndGroups(t *testing.T) {
	apiSent := json.RawMessage(`{
		"EventType": "messages",
		"owner": "551832653991",
		"message": {
			"messageid": "MSG1",
			"chatid": "5518997463184@s.whatsapp.net",
			"text": "mensagem automatica do bot",
			"fromMe": true,
			"wasSentByApi": true
		}
	}`)
	if parsed, _ := ParseUazapiMessageEvent(apiSent); parsed != nil {
		t.Fatal("expected message with wasSentByApi=true to be ignored")
	}

	groupMsg := json.RawMessage(`{
		"EventType": "messages",
		"owner": "551832653991",
		"chat": { "wa_isGroup": true, "wa_chatid": "1203630123456@g.us" },
		"message": {
			"messageid": "MSG2",
			"chatid": "1203630123456@g.us",
			"isGroup": true,
			"text": "mensagem em grupo"
		}
	}`)
	if parsed, _ := ParseUazapiMessageEvent(groupMsg); parsed != nil {
		t.Fatal("expected group message to be ignored")
	}
}

