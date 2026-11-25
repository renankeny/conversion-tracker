package googleads

import (
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestBuildUserData_NoPostalCodeWithFullNameOmitsIncompleteAddress(t *testing.T) {
	in := domain.SendEventInput{
		Email:     "teste@exemplo.com",
		Phone:     "5511999998888",
		FirstName: "Willams",
		LastName:  "Ferreira",
		Country:   "BR",
	}

	identifiers := buildUserData(in)

	for _, id := range identifiers {
		if _, hasAddr := id["address"]; hasAddr {
			t.Fatalf("expected address to be omitted when postal_code is empty (Google Data Manager requires postal_code), got %v", id)
		}
	}
	if len(identifiers) != 2 {
		t.Fatalf("expected 2 identifiers (email, phone), got %d: %v", len(identifiers), identifiers)
	}
}

func TestBuildUserData_NoLastNameWithPostalCodeOmitsIncompleteAddress(t *testing.T) {
	in := domain.SendEventInput{
		Email:      "teste@exemplo.com",
		Phone:      "5511999998888",
		FirstName:  "Nome",
		PostalCode: "01310-100",
	}

	identifiers := buildUserData(in)

	for _, id := range identifiers {
		if _, hasAddr := id["address"]; hasAddr {
			t.Fatalf("expected address to be omitted when family_name is empty, got %v", id)
		}
	}
	if len(identifiers) != 2 {
		t.Fatalf("expected 2 identifiers (email, phone), got %d: %v", len(identifiers), identifiers)
	}
}

func TestBuildUserData_CompleteAddressIncludesAllFourFields(t *testing.T) {
	in := domain.SendEventInput{
		Email:      "teste@exemplo.com",
		Phone:      "5511999998888",
		FirstName:  "Renan",
		LastName:   "Keny",
		PostalCode: "01310-100",
		Country:    "br",
	}

	identifiers := buildUserData(in)

	var foundAddr map[string]any
	for _, id := range identifiers {
		if addr, ok := id["address"].(map[string]any); ok {
			foundAddr = addr
			break
		}
	}

	if foundAddr == nil {
		t.Fatalf("expected address to be present when all 4 fields are provided")
	}
	if foundAddr["postalCode"] != "01310100" {
		t.Errorf("expected normalized postalCode '01310100', got %v", foundAddr["postalCode"])
	}
	if foundAddr["regionCode"] != "BR" {
		t.Errorf("expected regionCode 'BR', got %v", foundAddr["regionCode"])
	}
	if foundAddr["givenName"] == "" || foundAddr["familyName"] == "" {
		t.Errorf("expected hashed givenName and familyName, got %v", foundAddr)
	}
}

func TestDetectPartialFailure(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "clean empty json",
			body:    `{}`,
			wantErr: "",
		},
		{
			name:    "clean upload result",
			body:    `{"results":[{"conversionAction":"customers/123/conversionActions/456"}]}`,
			wantErr: "",
		},
		{
			name:    "ok status code zero",
			body:    `{"responses":[{"status":{"code":0}}]}`,
			wantErr: "",
		},
		{
			name:    "partialFailureError",
			body:    `{"results":[{}],"partialFailureError":{"code":3,"message":"The click associated with the given GCLID is too old."}}`,
			wantErr: "The click associated with the given GCLID is too old.",
		},
		{
			name:    "partialFailureStatus",
			body:    `{"partialFailureStatus":{"code":3,"message":"Invalid conversion action id"}}`,
			wantErr: "Invalid conversion action id",
		},
		{
			name:    "top-level error object",
			body:    `{"error":{"code":400,"message":"Request contains an invalid argument."}}`,
			wantErr: "Request contains an invalid argument.",
		},
		{
			name:    "responses status message",
			body:    `{"responses":[{"status":{"code":3,"message":"EXPIRED_GCLID"}}]}`,
			wantErr: "EXPIRED_GCLID",
		},
		{
			name:    "results nested error",
			body:    `{"results":[{"error":{"message":"Conversion action disabled"}}]}`,
			wantErr: "Conversion action disabled",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectPartialFailure(tc.body)
			if got != tc.wantErr {
				t.Fatalf("detectPartialFailure() = %q, want %q", got, tc.wantErr)
			}
		})
	}
}

func TestResolveTargetOperatingAndDataManagerPayload(t *testing.T) {
	acc := &domain.GoogleAdsAccount{
		CustomerID:      "111-222-3333",
		LoginCustomerID: "999-888-7777",
	}
	act := &domain.GoogleAdsConversionAction{
		EventType:          "Purchase",
		ConversionActionID: "customers/9998887777/conversionActions/654321",
	}
	opID, actID := resolveTargetOperatingAndAction(acc, act)
	if opID != "9998887777" || actID != "654321" {
		t.Fatalf("expected MCC owner 9998887777 and action 654321, got op=%q act=%q", opID, actID)
	}

	pld := buildDataManagerPayload("1112223333", "9998887777", "654321", domain.SendEventInput{GCLID: "test_gclid"}, nil, true)
	dests := pld["destinations"].([]map[string]any)
	if dests[0]["reference"] != "dest_0" || dests[0]["productDestinationId"] != "654321" {
		t.Fatalf("unexpected destination payload: %+v", dests[0])
	}
	if _, hasLogin := dests[0]["loginAccount"]; !hasLogin {
		t.Fatalf("expected loginAccount when MCC differs from operatingAccount")
	}

	standalone := buildDataManagerPayload("1112223333", "1112223333", "654321", domain.SendEventInput{GCLID: "test_gclid"}, nil, true)
	destSolo := standalone["destinations"].([]map[string]any)[0]
	if _, hasLogin := destSolo["loginAccount"]; hasLogin {
		t.Fatalf("expected loginAccount to be omitted when equal to operatingAccount")
	}
}

func TestFormatGoogleDataManagerError_IncompatibleType(t *testing.T) {
	raw := `{"error":{"code":400,"details":[{"fieldViolations":[{"field":"events.events[0].destination_references","description":"Resource not found.","reason":"NOT_FOUND"}]}]}}`
	msg := formatGoogleDataManagerError(400, raw, &domain.GoogleAdsConversionAction{
		ConversionActionID: "777",
		ActionType:         "WEBPAGE",
	}, "1112223333")
	if msg == "" || msg[:2] != "A " {
		t.Fatalf("expected Portuguese explanation for WEBPAGE type, got %q", msg)
	}
}
