package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestOnixPay_ProductionPIXCreatedAndPaidWithEnvelope(t *testing.T) {
	createdWrapped := `[{
		"headers": {"user-agent": "axios/1.20.0"},
		"body": {
			"event": "TRANSACTION_CREATED",
			"token": "z147t7kh",
			"offerCode": null,
			"checkoutUrl": "",
			"orderId": "cmwtwwf126kqs01pkr8m51ev8",
			"orderUrl": "https://checkout.onixpayments.com.br/order/cmwtwwf126kqs01pkr8m51ev8",
			"client": {
				"id": "cmw6imqyr4nfk01ofwhpilmpy",
				"name": "nugtes",
				"email": "cliente@kronos.app",
				"phone": "(11) 99999-9999",
				"cpf": "111.444.777-35"
			},
			"transaction": {
				"id": "cmwtwwf216kqv01pkwnnj227b",
				"identifier": "eb_9020466897_8800425461_1795477423734",
				"status": "PENDING",
				"paymentMethod": "PIX",
				"originalAmount": 10.25,
				"amount": 10.25,
				"currency": "BRL"
			},
			"orderItems": [],
			"trackProps": {"isUpsell": false}
		}
	}]`

	paidWrapped := `[{
		"body": {
			"event": "TRANSACTION_PAID",
			"token": "z147t7kh",
			"orderId": "cmwtwwf126kqs01pkr8m51ev8",
			"orderUrl": "https://checkout.onixpayments.com.br/order/cmwtwwf126kqs01pkr8m51ev8",
			"client": {
				"name": "nugtes",
				"email": "cliente@kronos.app",
				"phone": "(11) 99999-9999"
			},
			"transaction": {
				"id": "cmwtwwf216kqv01pkwnnj227b",
				"identifier": "eb_9020466897_8800425461_1795477423734",
				"status": "COMPLETED",
				"paymentMethod": "PIX",
				"amount": 10.25,
				"commissionAmount": 9.5,
				"currency": "BRL"
			},
			"orderItems": [],
			"trackProps": {"isUpsell": false}
		}
	}]`

	createdItems, err := unwrapOnixPayPayloads([]byte(createdWrapped))
	if err != nil || len(createdItems) != 1 {
		t.Fatalf("expected 1 unwrapped created item, got %d err=%v", len(createdItems), err)
	}
	var pCreated onixPayPayload
	if err := json.Unmarshal(createdItems[0], &pCreated); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	evCreated, ok := resolveOnixPayEventName(pCreated.Event, pCreated.Transaction.Status)
	if !ok || evCreated != domain.EventAddPaymentInfo {
		t.Fatalf("expected AddPaymentInfo for TRANSACTION_CREATED, got %q ok=%v", evCreated, ok)
	}
	inCreated := buildOnixPayPurchaseInput(pCreated, evCreated, 7, "px-main", createdItems[0])
	if inCreated.OrderID != "cmwtwwf126kqs01pkr8m51ev8" || inCreated.Value != 10.25 || inCreated.Currency != "BRL" {
		t.Fatalf("unexpected created input: %+v", inCreated)
	}
	evIDCreated := webhookEventIDForEvent(inCreated.EventName, inCreated.OrderID, "", inCreated.ProductID)
	if evIDCreated != "addpaymentinfo:cmwtwwf126kqs01pkr8m51ev8" {
		t.Fatalf("expected distinct event_id for AddPaymentInfo, got %q", evIDCreated)
	}

	paidItems, err := unwrapOnixPayPayloads([]byte(paidWrapped))
	if err != nil || len(paidItems) != 1 {
		t.Fatalf("expected 1 unwrapped paid item, got %d err=%v", len(paidItems), err)
	}
	var pPaid onixPayPayload
	if err := json.Unmarshal(paidItems[0], &pPaid); err != nil {
		t.Fatalf("decode paid: %v", err)
	}
	evPaid, ok := resolveOnixPayEventName(pPaid.Event, pPaid.Transaction.Status)
	if !ok || evPaid != domain.EventPurchase {
		t.Fatalf("expected Purchase for TRANSACTION_PAID, got %q ok=%v", evPaid, ok)
	}
	inPaid := buildOnixPayPurchaseInput(pPaid, evPaid, 7, "px-main", paidItems[0])
	evIDPaid := webhookEventIDForEvent(inPaid.EventName, inPaid.OrderID, "", inPaid.ProductID)
	if evIDPaid != "cmwtwwf126kqs01pkr8m51ev8" {
		t.Fatalf("expected clean orderID for Purchase event_id, got %q", evIDPaid)
	}
}

func TestOnixPay_DocSamplesNestedAndRootTrackProps(t *testing.T) {
	docCreated := `{
		"event": "TRANSACTION_CREATED",
		"token": "juj2ln0z7s",
		"offerCode": "ABCK181",
		"checkoutUrl": "https://checkout.teste.com/checkout/ckxyz123?session=01890a5d&offer=ABCK181&utm_source=facebook|userIdABC1234567|&utm_medium=cpc",
		"client": {
			"id": "pbhd1h9e1x0khpdk",
			"name": "John Doe",
			"email": "jondoe@gmail.com",
			"phone": "(11) 9 8888-7777",
			"address": {
				"country": "BR",
				"zipCode": "01304-000",
				"state": "SP",
				"city": "São Paulo"
			}
		},
		"transaction": {
			"id": "1pned18asp",
			"status": "COMPLETED",
			"paymentMethod": "CREDIT_CARD",
			"originalCurrency": "USD",
			"originalAmount": 20,
			"amount": 100,
			"currency": "BRL",
			"orderItems": [{
				"id": "wdod0abvdt",
				"price": 100,
				"product": {"id": "oigtz177jp", "name": "Curso de marketing", "externalId": "KSA912"}
			}],
			"trackProps": {
				"utm_source": "facebook|userIdABC1234567|",
				"utm_medium": "cpc",
				"utm_campaign": "lancamento",
				"fbc": "fb.1.1234567890.0987654321",
				"fbp": "fb.1.0987654321.1234567890",
				"ip": "179.241.195.127",
				"user_agent": "Mozilla/5.0 Mobile"
			}
		}
	}`

	var p onixPayPayload
	if err := json.Unmarshal([]byte(docCreated), &p); err != nil {
		t.Fatalf("decode docCreated: %v", err)
	}
	ev, ok := resolveOnixPayEventName(p.Event, p.Transaction.Status)
	if !ok || ev != domain.EventAddPaymentInfo {
		t.Fatalf("TRANSACTION_CREATED with COMPLETED card status must still map to AddPaymentInfo (avoiding duplicate Purchase), got %q", ev)
	}
	in := buildOnixPayPurchaseInput(p, ev, 1, "", []byte(docCreated))
	if in.ExternalID != "ABC1234567" {
		t.Fatalf("expected ExternalID ABC1234567, got %q", in.ExternalID)
	}
	if in.FirstName != "John" || in.LastName != "Doe" || in.OrderID != "1pned18asp" {
		t.Fatalf("unexpected buyer/order: %+v", in)
	}
	if in.Value != 100 || in.Currency != "BRL" || in.ProductID != "oigtz177jp" || in.ProductName != "Curso de marketing" {
		t.Fatalf("unexpected product/value: %+v", in)
	}
	if in.FBC != "fb.1.1234567890.0987654321" || in.FBP != "fb.1.0987654321.1234567890" || in.ClientIP != "179.241.195.127" {
		t.Fatalf("unexpected fbc/fbp/ip: %+v", in)
	}
	if in.Country != "BR" || in.State != "SP" || in.City != "São Paulo" || in.PostalCode != "01304-000" {
		t.Fatalf("unexpected address enrichment: %+v", in)
	}

	u := &domain.TrackedUser{}
	applyWebhookTrackingEnrichment(u, in)
	if u.FirstUTMSource != "facebook" || u.LastUTMCampaign != "lancamento" || u.ClientIP != "179.241.195.127" {
		t.Fatalf("unexpected user enrichment: %+v", u)
	}
}

func TestOnixPay_HTTPIgnoredEventAndSupportedPlatform(t *testing.T) {
	if !domain.IsSupportedCheckoutPlatform("onixpay") {
		t.Fatalf("expected onixpay to be registered in supportedCheckoutPlatforms")
	}
	if ev, ok := resolveOnixPayEventName("TRANSACTION_REFUNDED", "REFUNDED"); !ok || ev != domain.EventRefund {
		t.Fatalf("expected TRANSACTION_REFUNDED -> Refund, got %q (ok=%v)", ev, ok)
	}
	if ev, ok := resolveOnixPayEventName("TRANSACTION_CHARGEBACK", "CHARGEBACK"); !ok || ev != domain.EventChargeback {
		t.Fatalf("expected TRANSACTION_CHARGEBACK -> Chargeback, got %q (ok=%v)", ev, ok)
	}
	h := NewWebhooks(nil, nil, nil, nil, GatewaySecrets{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/onixpay?acc=1", bytes.NewBufferString(`{"event":"TRANSACTION_EXPIRED","transaction":{"status":"EXPIRED"}}`))
	rec := httptest.NewRecorder()
	h.OnixPay(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on ignored event, got %d", rec.Code)
	}
}
