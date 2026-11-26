package handlers

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestCaktoUnmarshalSingle verifica que data como objeto único (formato v1)
// é decodificado para um slice com 1 item.
func TestCaktoUnmarshalSingle(t *testing.T) {
	body := `{
		"event": "purchase_approved",
		"data": {
			"id": "abc-123",
			"status": "paid",
			"baseAmount": 347,
			"amount": 421.28,
			"utm_source": "|userIdAS2WMBB6AM|",
			"customer": {
				"name": "Felipe Devatz",
				"email": "f@dev.com",
				"phone": "5517991381882"
			},
			"offer": { "currency": "BRL" }
		}
	}`

	var p caktoPayload
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(p.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(p.Data))
	}
	got := p.Data[0]
	if got.ID != "abc-123" {
		t.Errorf("ID: got %q, want abc-123", got.ID)
	}
	if got.UTMSource != "|userIdAS2WMBB6AM|" {
		t.Errorf("UTMSource: got %q", got.UTMSource)
	}
	if got.BaseAmount != 347 {
		t.Errorf("BaseAmount: got %v, want 347", got.BaseAmount)
	}
	if got.Customer.Email != "f@dev.com" {
		t.Errorf("Email: got %q", got.Customer.Email)
	}
}

// TestCaktoUnmarshalArray verifica que data como array (formato v2 batch)
// é decodificado item a item.
func TestCaktoUnmarshalArray(t *testing.T) {
	body := `{
		"event": "purchase_approved",
		"data": [
			{
				"id": "748dcf12",
				"status": "paid",
				"baseAmount": 5,
				"utm_source": "|userIdAS2WMBB6AM|",
				"customer": { "name": "Cliente Teste", "email": "cliente@exemplo.com", "phone": "5511999998888" },
				"offer": { "currency": "BRL", "price": 5 }
			},
			{
				"id": "another",
				"status": "paid",
				"baseAmount": 10,
				"customer": { "name": "outro", "email": "o@o.com", "phone": "5511" },
				"offer": { "currency": "BRL" }
			}
		]
	}`

	var p caktoPayload
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(p.Data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(p.Data))
	}
	if p.Data[0].ID != "748dcf12" {
		t.Errorf("first.ID: got %q", p.Data[0].ID)
	}
	if p.Data[1].ID != "another" {
		t.Errorf("second.ID: got %q", p.Data[1].ID)
	}
}

// TestCaktoUnmarshalEmpty: webhook sem data devolve slice vazia, não nil-deref.
func TestCaktoUnmarshalEmpty(t *testing.T) {
	cases := map[string]string{
		"null data":  `{"event":"purchase_approved","data":null}`,
		"empty data": `{"event":"purchase_approved","data":[]}`,
		"no data":    `{"event":"purchase_approved"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			var p caktoPayload
			if err := json.Unmarshal([]byte(body), &p); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if len(p.Data) != 0 {
				t.Errorf("expected 0 items, got %d", len(p.Data))
			}
		})
	}
}

// TestCaktoUnmarshalInvalid: JSON malformado deve retornar erro.
func TestCaktoUnmarshalInvalid(t *testing.T) {
	body := `{"event":"x","data":invalid}`
	var p caktoPayload
	if err := json.Unmarshal([]byte(body), &p); err == nil {
		t.Errorf("expected error for invalid JSON, got nil")
	} else if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "character") {
		// Aceita variações de mensagem entre versões do encoding/json.
		// Só queremos confirmar que algum erro foi propagado.
	}
}

// TestCaktoRefundRealPayload valida o formato real de webhook de reembolso da Cakto.
func TestCaktoRefundRealPayload(t *testing.T) {
	body := `{
		"event": "refund",
		"secret": "test-secret",
		"data": {
			"id": "8f10d341-b2d0-428f-8284-a3b0db62ee18",
			"status": "refunded",
			"baseAmount": 429,
			"amount": 429.99,
			"currency": "BRL",
			"offer_type": "main",
			"refundedAt": "2026-11-25T23:16:19.399782-03:00",
			"refund_reason": "A qualidade do produto não atingiu as minhas expectativas",
			"utm_source": "direto|userIdOYXRMZSPEW|",
			"customer": { "name": "Cliente Teste", "email": "teste@exemplo.com", "phone": "5561999990000" },
			"offer": { "id": "35q3hkt", "name": "Produto Teste", "price": 429, "currency": "BRL" }
		}
	}`
	var p caktoPayload
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("decode refund error: %v", err)
	}
	if len(p.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(p.Data))
	}
	it := p.Data[0]
	if detectRefundOrChargeback(p.Event, it.Status) != "Refund" {
		t.Fatalf("expected Refund detection")
	}
	if it.OfferType != "main" || it.RefundedAt == "" || it.RefundReason == "" || it.BaseAmount != 429 {
		t.Fatalf("unexpected parsed cakto refund fields: %+v", it)
	}
}
