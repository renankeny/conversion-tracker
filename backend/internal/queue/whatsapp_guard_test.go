package queue

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestExtractProtocolCode_BlocksPrivateChatFalsePositives(t *testing.T) {
	blocked := []string{
		"Segue n.f referente ao serviço prestado",
		"Bom dia, referente ao aluguel de novembro",
		"Protocolo enviado para análise ontem",
		"Código de atendimento liberado na recepção",
		"Protocolo: 2026112500918273",
		"Protocolo de atendimento: 9988776655",
		"Referência: apartamento",
		"Protocolo: pagamento",
		"Ref: erente",
		"Protocolo: 550e8400-e29b-41d4-a716-446655440000",
	}
	for _, msg := range blocked {
		if got := ExtractProtocolCode(msg); got != "" {
			t.Fatalf("expected private message %q to NOT extract protocol, got %q", msg, got)
		}
	}

	valid := map[string]string{
		"Protocolo: A9F2K7M1Z0":                        "A9F2K7M1Z0",
		"Olá, vi no site!\nProtocolo: *ABC1234567*":    "ABC1234567",
		"*Protocolo:* X8B2M9Q4L1":                      "X8B2M9Q4L1",
		"Protocolo #Z1Y2X3W4V5.":                       "Z1Y2X3W4V5",
		"(Ref: K9L8M7N6P5)":                            "K9L8M7N6P5",
		"Olá!\nProtocolo: KRMRKC3NIR":                  "KRMRKC3NIR",
	}
	for msg, want := range valid {
		if got := ExtractProtocolCode(msg); got != want {
			t.Fatalf("expected %q from %q, got %q", want, msg, got)
		}
	}
}

func TestEvolutionGo_BlocksOrganicLinksReelsForwardedAndPlainWaMe(t *testing.T) {
	cases := []struct {
		name    string
		payload string
	}{
		{
			name: "shared Instagram Reel with sourceID but sourceType post",
			payload: `{
				"Info": {"ID": "PRIV_REEL", "Chat": "5511977776666@s.whatsapp.net", "Sender": "5511977776666@s.whatsapp.net"},
				"Message": {
					"extendedTextMessage": {
						"text": "Olha que vídeo engraçado https://www.instagram.com/reel/C9xyz/",
						"contextInfo": {
							"externalAdReply": {
								"title": "Instagram Reel",
								"sourceType": "post",
								"sourceID": "339182736451928374",
								"sourceURL": "https://www.instagram.com/reel/C9xyz/"
							}
						}
					}
				}
			}`,
		},
		{
			name: "rich link preview without sourceType or ctwaClid",
			payload: `{
				"Info": {"ID": "PRIV_YT", "Chat": "5511977776666@s.whatsapp.net", "Sender": "5511977776666@s.whatsapp.net"},
				"Message": {
					"extendedTextMessage": {
						"text": "Assiste isso https://youtu.be/abc",
						"contextInfo": {
							"externalAdReply": {
								"title": "YouTube",
								"sourceID": "123456789012345",
								"mediaURL": "https://youtu.be/abc"
							}
						}
					}
				}
			}`,
		},
		{
			name: "forwarded message containing old ad contextInfo",
			payload: `{
				"Info": {"ID": "PRIV_FWD", "Chat": "5511977776666@s.whatsapp.net", "Sender": "5511977776666@s.whatsapp.net"},
				"Message": {
					"extendedTextMessage": {
						"text": "Olha o anúncio do concorrente",
						"contextInfo": {
							"isForwarded": true,
							"forwardingScore": 2,
							"conversionSource": "FB_Ads",
							"externalAdReply": {
								"sourceType": "ad",
								"sourceID": "120259519567440386",
								"ctwaClid": "AfiOv5AdFoJPcE34i999"
							}
						}
					}
				}
			}`,
		},
		{
			name: "plain wa.me link click_to_chat_link",
			payload: `{
				"Info": {"ID": "PRIV_WAME", "Chat": "5511977776666@s.whatsapp.net", "Sender": "5511977776666@s.whatsapp.net"},
				"Message": {
					"extendedTextMessage": {
						"text": "Oi tudo bem? Peguei seu contato no cartão",
						"contextInfo": {
							"entryPointConversionSource": "click_to_chat_link",
							"entryPointConversionApp": "whatsapp"
						}
					}
				}
			}`,
		},
	}

	inst := &domain.WhatsAppInstance{ID: 1, AccountID: 1, PixelID: 1}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseEvolutionMessageEvent(json.RawMessage(tc.payload))
			if err != nil || parsed == nil {
				t.Fatalf("expected parsed message, err=%v", err)
			}
			if parsed.CtwaClid != "" || parsed.SourceID != "" || parsed.HasMsgAdReferral {
				t.Fatalf("expected ZERO ad markers on private message, got clid=%q sid=%q ref=%v", parsed.CtwaClid, parsed.SourceID, parsed.HasMsgAdReferral)
			}
			if u, orig := resolveOrCreatePortalLead(context.Background(), Deps{}, inst, parsed, ExtractProtocolCode(parsed.Content), "5511977776666", ""); u != nil || orig != "" {
				t.Fatalf("expected private message to NEVER open portal lead, got u=%v orig=%q", u, orig)
			}
		})
	}
}

func TestUazapiAndApiPremium_BlocksLeadFieldsSelfChatStatusAndSystemEvents(t *testing.T) {
	// 1. Campos persistentes de CRM externo (n8n / Typebot / Kommo / OpenAI thread_*) NUNCA podem virar CTWA.
	crmPersisted := json.RawMessage(`{
		"EventType": "messages",
		"owner": "551832653991",
		"chat": {
			"phone": "5518991112233",
			"wa_chatid": "5518991112233@s.whatsapp.net",
			"lead_field10": "thread_abc123def456ghi789xyz",
			"lead_field11": "210702790877292",
			"lead_field12": "instagram"
		},
		"message": {
			"messageid": "PRIV_CRM_1",
			"chatid": "5518991112233@s.whatsapp.net",
			"fromMe": false,
			"text": "Mãe, já chego em casa!"
		}
	}`)
	parsed, err := ParseUazapiMessageEvent(crmPersisted)
	if err != nil || parsed == nil {
		t.Fatalf("expected parsed message, err=%v", err)
	}
	if parsed.CtwaClid != "" || parsed.SourceID != "" || parsed.HasMsgAdReferral {
		t.Fatalf("expected chat.lead_field10/11 to be completely ignored, got clid=%q sid=%q", parsed.CtwaClid, parsed.SourceID)
	}

	// 2. Self-chat ("Mensagem para você mesmo") com @lid na UAZAPI / API Premium e na Evolution Go.
	uazapiSelf := json.RawMessage(`{
		"EventType": "messages",
		"owner": "551832653991",
		"chat": { "phone": "551832653991", "wa_chatid": "551832653991@s.whatsapp.net", "wa_chatlid": "888777666@lid" },
		"message": { "messageid": "SELF_UAZ", "chatid": "551832653991@s.whatsapp.net", "chatlid": "888777666@lid", "sender": "888777666@lid", "fromMe": false, "text": "Lembrar de pagar boleto" }
	}`)
	if pSelf, _ := ParseUazapiMessageEvent(uazapiSelf); pSelf != nil {
		t.Fatalf("expected UAZAPI self-chat to be ignored, got %+v", pSelf)
	}
	evoSelf := &ParsedWhatsAppMessage{
		MessageID: "SELF_EVO", ChatJID: "551832653991@s.whatsapp.net",
		SenderJID: "551832653991@s.whatsapp.net", SenderAltJID: "888777666@lid", IsFromMe: false, Content: "Anotação pessoal",
	}
	if _, _, _, skip := ResolveContactJID(evoSelf, "551832653991@s.whatsapp.net", nil, nil); !skip {
		t.Fatal("expected Evolution Go self-chat with @lid to be skipped")
	}

	// 3. Status, Reactions e Meta AI bot devem ser ignorados imediatamente.
	for _, raw := range []string{
		`{"EventType":"messages","owner":"551832653991","message":{"messageid":"ST1","chatid":"5518991112233@s.whatsapp.net","isStatus":true,"text":"status"}}`,
		`{"EventType":"messages","owner":"551832653991","message":{"messageid":"RX1","chatid":"5518991112233@s.whatsapp.net","messageType":"ReactionMessage","text":"❤️"}}`,
		`{"EventType":"history","owner":"551832653991","message":{"messageid":"HS1","chatid":"5518991112233@s.whatsapp.net","text":"antiga"}}`,
		`{"EventType":"messages","owner":"551832653991","message":{"messageid":"AI1","chatid":"13135550002@s.whatsapp.net","text":"Meta AI"}}`,
	} {
		if p, _ := ParseUazapiMessageEvent(json.RawMessage(raw)); p != nil {
			t.Fatalf("expected ignorable UAZAPI event %s to return nil, got %+v", raw, p)
		}
	}
}

func TestIOSStrippedCtwaClid_StillAcceptsVerifiedMetaAdSourceType(t *testing.T) {
	iosAd := json.RawMessage(`{
		"Info": {"ID": "IOS_AD_1", "Chat": "5511966665555@s.whatsapp.net", "Sender": "5511966665555@s.whatsapp.net"},
		"Message": {
			"extendedTextMessage": {
				"text": "Olá! Quero orçamento",
				"contextInfo": {
					"conversionSource": "FB_Ads",
					"entryPointConversionSource": "ctwa_ad",
					"entryPointConversionApp": "facebook",
					"externalAdReply": {
						"sourceType": "ad",
						"sourceID": "120259519567440386"
					}
				}
			}
		}
	}`)
	parsed, err := ParseEvolutionMessageEvent(iosAd)
	if err != nil || parsed == nil {
		t.Fatalf("expected parsed iOS CTWA message, err=%v", err)
	}
	if parsed.SourceID != "120259519567440386" || parsed.SourceApp != "facebook" || !parsed.HasMsgAdReferral {
		t.Fatalf("expected verified iOS Meta Ad sourceID to be accepted, got %+v", parsed)
	}
}
