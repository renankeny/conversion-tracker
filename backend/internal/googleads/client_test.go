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
