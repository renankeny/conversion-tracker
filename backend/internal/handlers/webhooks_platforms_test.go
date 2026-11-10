package handlers

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func TestFormBodyToJSONMonetizze(t *testing.T) {
	form := "chave_unica=sec123&venda[codigo]=41852883&venda[status]=Completa&venda[valor]=157.90&venda[utm_source]=ig%7CuserIdABCD123456%7C&comprador[nome]=Joao+Silva&comprador[email]=joao%40mail.com&comprador[celular]=11988887777&produto[codigo]=99&produto[nome]=Curso+VIP"
	req := httptest.NewRequest("POST", "/api/v1/webhooks/monetizze", bytes.NewBufferString(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var p monetizzePayload
	jsonBytes, err := decodeWebhookPayload(req, &p)
	if err != nil {
		t.Fatalf("decode form error: %v", err)
	}
	if p.ChaveUnica != "sec123" || anyString(p.Venda.Codigo) != "41852883" {
		t.Fatalf("unexpected venda/chave: %+v", p)
	}
	if got := util.ParseMoneyAny(p.Venda.Valor); got != 157.90 {
		t.Errorf("valor: got %v, want 157.90", got)
	}
	if uid := util.ExtractUserIDDeep(jsonBytes); uid != "ABCD123456" {
		t.Errorf("uid: got %q, want ABCD123456", uid)
	}
}

func TestPlatformPayloadsDecodeAndExtract(t *testing.T) {
	t.Run("eduzz_v2", func(t *testing.T) {
		body := `{"id":"ev1","event":"myeduzz.invoice_paid","data":{"id":"8899","status":"paid","price":{"currency":"BRL","value":197.00},"buyer":{"name":"Maria Souza","email":"m@s.com","cellphoneNullable":"+5511999998888"},"items":[{"productId":"P1","name":"Mentoria"}],"utmNullable":{"sourceNullable":"fb|userIdEDZ9988776|"}}}`
		req := httptest.NewRequest("POST", "/webhooks/eduzz", bytes.NewBufferString(body))
		var p eduzzPayload
		jb, err := decodeWebhookPayload(req, &p)
		if err != nil {
			t.Fatal(err)
		}
		if util.ParseMoneyAny(p.Data.Price.Value) != 197.0 || util.ExtractUserIDDeep(jb) != "EDZ9988776" {
			t.Fatalf("eduzz mismatch: %+v", p)
		}
	})

	t.Run("ticto_v2_phone_obj_and_cents", func(t *testing.T) {
		body := `{"token":"tk","status":"authorized","order":{"hash":"TOB1","paid_amount":19700},"customer":{"name":"Ana","email":"a@a.com","phone":{"ddi":"+55","ddd":"21","number":"987654321"}},"item":{"product_id":77,"product_name":"Ebook"},"url_params":{"query_params":{"sck":"|userIdTIC1122334|"}}}`
		req := httptest.NewRequest("POST", "/webhooks/ticto", bytes.NewBufferString(body))
		var p tictoPayload
		jb, err := decodeWebhookPayload(req, &p)
		if err != nil {
			t.Fatal(err)
		}
		if got := util.ParseMoneyCents(p.Order.PaidAmount); got != 197.0 {
			t.Errorf("paid_amount: got %v, want 197", got)
		}
		if ph := parseTictoPhone(p.Customer.Phone); ph != "5521987654321" {
			t.Errorf("phone: got %q, want 5521987654321", ph)
		}
		if uid := util.ExtractUserIDDeep(jb); uid != "TIC1122334" {
			t.Errorf("uid: got %q", uid)
		}
	})

	t.Run("perfectpay_and_guru_ddd", func(t *testing.T) {
		ppBody := `{"token":"t","code":"PP1","sale_amount":226.90,"sale_status_enum":2,"customer":{"full_name":"Carlos","email":"c@c.com","phone_area_code":"11","phone_number":"999991111"},"product":{"code":"PR1","name":"Curso"},"metadata":{"utm_source":"|userIdPP99887766|"}}`
		req := httptest.NewRequest("POST", "/webhooks/perfectpay", bytes.NewBufferString(ppBody))
		var pp perfectPayPayload
		jb, err := decodeWebhookPayload(req, &pp)
		if err != nil {
			t.Fatal(err)
		}
		if joinPhoneDDD("", pp.Customer.PhoneAreaCode, pp.Customer.PhoneNumber) != "11999991111" || util.ExtractUserIDDeep(jb) != "PP99887766" {
			t.Fatalf("perfectpay mismatch")
		}

		guruBody := `{"api_token":"gt","id":"tx1","status":"complete","contact":{"name":"Ric","email":"r@g.com","phone_local_code":"21","phone_number":"983491234"},"payment":{"status":"paid","value":197.00},"product":{"id":"p1","name":"Prod"},"source":{"checkout_source":"|userIdGUR1234567|"}}`
		gReq := httptest.NewRequest("POST", "/webhooks/guru", bytes.NewBufferString(guruBody))
		var gp guruPayload
		gjb, err := decodeWebhookPayload(gReq, &gp)
		if err != nil {
			t.Fatal(err)
		}
		if util.ParseMoneyAny(gp.Payment.Value) != 197.0 || util.ExtractUserIDDeep(gjb) != "GUR1234567" {
			t.Fatalf("guru mismatch")
		}
	})

	t.Run("lastlink_both_schemas", func(t *testing.T) {
		nested := `{"Event":"Purchase_Order_Confirmed","Data":{"Buyer":{"Name":"Leo","Email":"l@l.com","PhoneNumber":"+5511999990000"},"Purchase":{"PaymentId":"PAY1","Price":97.50},"Products":[{"Id":"PR9","Name":"Club"}],"Utm":{"UtmSource":"|userIdLLK1234567|"}}}`
		req1 := httptest.NewRequest("POST", "/webhooks/lastlink", bytes.NewBufferString(nested))
		var p1 lastlinkPayload
		jb1, err := decodeWebhookPayload(req1, &p1)
		if err != nil {
			t.Fatal(err)
		}
		if p1.Data.Purchase.PaymentID != "PAY1" || util.ParseMoneyAny(p1.Data.Purchase.Price) != 97.50 || util.ExtractUserIDDeep(jb1) != "LLK1234567" {
			t.Fatalf("lastlink nested mismatch: %+v", p1)
		}

		flat := `{"event":"Purchase_Order_Confirmed","status":"paid","order":{"id":"ORD2","amount":147.00},"buyer":{"name":"Bia","email":"b@b.com","phone":"+5521988887777"},"utm":{"source":"|userIdLLK7654321|"}}`
		req2 := httptest.NewRequest("POST", "/webhooks/lastlink", bytes.NewBufferString(flat))
		var p2 lastlinkPayload
		jb2, err := decodeWebhookPayload(req2, &p2)
		if err != nil {
			t.Fatal(err)
		}
		if p2.Order.ID != "ORD2" || util.ParseMoneyAny(p2.Order.Amount) != 147.00 || util.ExtractUserIDDeep(jb2) != "LLK7654321" {
			t.Fatalf("lastlink flat mismatch: %+v", p2)
		}
	})

	t.Run("hubla_pepper_kiwify_cents", func(t *testing.T) {
		if got := util.ParseMoneyCents("9700"); got != 97.0 {
			t.Errorf("string cents: got %v, want 97", got)
		}
		if got := util.ParseMoneyCents("97.50"); got != 97.50 {
			t.Errorf("decimal string in ParseMoneyCents: got %v, want 97.50", got)
		}
		hubla := `{"type":"invoice.status_updated","event":{"product":{"id":"H1","name":"Comunidade"},"invoice":{"id":"inv1","orderId":"ordH1","status":"paid","amountCents":9990,"currency":"BRL"},"payer":{"firstName":"John","lastName":"Doe","email":"j@d.com","phone":"+5511977776666"},"utm":{"utm_source":"|userIdHUB1234567|"}}}`
		req := httptest.NewRequest("POST", "/webhooks/hubla", bytes.NewBufferString(hubla))
		var hp hublaPayload
		jb, err := decodeWebhookPayload(req, &hp)
		if err != nil {
			t.Fatal(err)
		}
		if util.ParseMoneyCents(hp.Event.Invoice.AmountCents) != 99.90 || util.ExtractUserIDDeep(jb) != "HUB1234567" {
			t.Fatalf("hubla mismatch")
		}
	})

	t.Run("cartpanda_appmax_doppus_vega", func(t *testing.T) {
		cp := `{"event":"order.paid","order":{"id":555,"financial_status":"paid","total_price":"159.90","customer":{"first_name":"Joao","last_name":"Silva","email":"j@s.com","phone":"+5511999999999"},"conversion_summary":{"utm_source":"|userIdCPD1234567|"}}}`
		var cpP cartpandaPayload
		jb, err := decodeWebhookPayload(httptest.NewRequest("POST", "/", bytes.NewBufferString(cp)), &cpP)
		if err != nil || util.ParseMoneyAny(cpP.Order.TotalPrice) != 159.90 || util.ExtractUserIDDeep(jb) != "CPD1234567" {
			t.Fatalf("cartpanda err=%v", err)
		}

		vg := `{"checkout_id":"chk1","status":"approved","total_price":"236.00","customer":{"name":"Ana","email":"a@v.com","phone":"5511988887777"},"checkout":{"utm_source":"|userIdVEG1234567|"},"plans":[{"id":"pl1","name":"Plano","products":[{"id":"pr1","name":"Curso"}]}]}`
		var vgP vegaPayload
		vjb, err := decodeWebhookPayload(httptest.NewRequest("POST", "/", bytes.NewBufferString(vg)), &vgP)
		if err != nil || util.ParseMoneyAny(vgP.TotalPrice) != 236.00 || util.ExtractUserIDDeep(vjb) != "VEG1234567" {
			t.Fatalf("vega err=%v", err)
		}
	})
}

