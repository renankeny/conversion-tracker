package tiktok

import (
	"net/http"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func TestBuildPayload_OfficialEventsAPIv13Compliance(t *testing.T) {
	val := 347.0
	in := domain.SendEventInput{
		PixelName:       "whatstracker",
		EventName:       domain.EventPurchase,
		EventID:         "EV_123",
		EventTime:       1762735569,
		SourceURL:       "https://exemplo.com/obrigado",
		Referrer:        "https://exemplo.com/checkout",
		ExternalID:      "3A80OIZZ4Z",
		Email:           "Willams@Exemplo.com ",
		Phone:           "(11) 99999-8888",
		FirstName:       "Willams",
		LastName:        "Ferreira",
		City:            "São Paulo",
		State:           "São Paulo",
		Country:         "Brasil",
		PostalCode:      "01310-100",
		TTClid:          "TTCLID_ABC",
		TTP:             "TTP_COOKIE_123",
		ClientIP:        "187.10.20.30",
		ClientUserAgent: "Mozilla/5.0",
		Value:           &val,
		Currency:        "BRL",
		OrderID:         "ORD-99",
		AggregatedItems: []domain.AggregatedItem{
			{OrderID: "ORD-99-A", ProductTitle: "Curso Principal", Value: 297.0},
			{OrderID: "ORD-99-B", ProductTitle: "Order Bump", Value: 50.0},
		},
	}

	evName := mapEventName(in.EventName)
	if evName != "Purchase" {
		t.Fatalf("expected mapped event 'Purchase', got %q", evName)
	}

	pld := buildPayload("PIXEL_TT_1", "TEST99", "", evName, in)
	dataArr := pld["data"].([]any)
	ev := dataArr[0].(map[string]any)
	user := ev["user"].(map[string]any)

	// 1. Campos geográficos em texto plano e nomes oficiais (country, locale)
	if user["city"] != "saopaulo" {
		t.Errorf("expected plain city 'saopaulo', got %v", user["city"])
	}
	if user["state"] != "sp" {
		t.Errorf("expected plain state 'sp', got %v", user["state"])
	}
	if user["country"] != "br" {
		t.Errorf("expected plain country 'br', got %v", user["country"])
	}
	if _, bad := user["country_code"]; bad {
		t.Errorf("unexpected legacy key country_code")
	}
	if user["locale"] != "pt-BR" {
		t.Errorf("expected locale 'pt-BR', got %v", user["locale"])
	}
	if _, bad := user["lang"]; bad {
		t.Errorf("unexpected legacy key lang")
	}

	// 2. external_id deve ser lowercased antes do SHA-256 para bater com ttq.identify
	wantExtHash := util.SHA256Hex("3a80oizz4z")
	if user["external_id"] != wantExtHash {
		t.Errorf("expected lowercased external_id hash %s, got %v", wantExtHash, user["external_id"])
	}

	// 3. Telefone E.164 com +55 antes do SHA-256
	wantPhoneHash := util.SHA256Hex("+5511999998888")
	if user["phone"] != wantPhoneHash {
		t.Errorf("expected E.164 phone hash %s, got %v", wantPhoneHash, user["phone"])
	}

	// 4. Properties: sem num_items inválido e com preços reais dos AggregatedItems
	props := ev["properties"].(map[string]any)
	if _, hasNumItems := props["num_items"]; hasNumItems {
		t.Errorf("expected num_items to be omitted from TikTok properties")
	}
	contents := props["contents"].([]map[string]any)
	if len(contents) != 2 || contents[0]["price"] != 297.0 || contents[1]["price"] != 50.0 {
		t.Errorf("expected individual aggregated prices [297, 50], got %v", contents)
	}
}

func TestMapEventNameAndLogicalErrors(t *testing.T) {
	if got := mapEventName(domain.EventLead); got != "Lead" {
		t.Errorf("EventLead -> %q, want 'Lead'", got)
	}
	if got := mapEventName(domain.EventAddShippingInfo); got != "AddShippingInfo" {
		t.Errorf("EventAddShippingInfo -> %q, want 'AddShippingInfo'", got)
	}
	if got := mapEventName(domain.EventPageView); got != "Pageview" {
		t.Errorf("EventPageView -> %q, want 'Pageview'", got)
	}

	st400, msg400 := extractTikTokError([]byte(`{"code":40002,"message":"Invalid parameter"}`))
	if st400 != http.StatusBadRequest || msg400 == "" {
		t.Errorf("expected 400 for code 40002, got %d (%s)", st400, msg400)
	}
	st401, _ := extractTikTokError([]byte(`{"code":40105,"message":"Access token invalid"}`))
	if st401 != http.StatusUnauthorized {
		t.Errorf("expected 401 for code 40105, got %d", st401)
	}
	st429, _ := extractTikTokError([]byte(`{"code":40016,"message":"Too many requests"}`))
	if st429 != http.StatusTooManyRequests {
		t.Errorf("expected 429 for code 40016, got %d", st429)
	}
}

