package notify

import (
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestFormatMoneyAndFriendlyOrigin(t *testing.T) {
	if got := FormatMoney(1497.5, "BRL"); got != "R$ 1.497,50" {
		t.Fatalf("unexpected BRL format: %s", got)
	}
	if got := FormatMoney(99, "USD"); got != "US$ 99,00" {
		t.Fatalf("unexpected USD format: %s", got)
	}
	if got := FriendlyGatewayOrigin("hotmart"); got != "Hotmart" {
		t.Fatalf("unexpected gateway origin: %s", got)
	}
	if got := FriendlyGatewayOrigin("whatsapp_link"); got != "Link Rastreável WhatsApp" {
		t.Fatalf("unexpected whatsapp_link origin: %s", got)
	}
}

func TestNotificationPreferencesScopeAndMandatoryCategories(t *testing.T) {
	prefs := domain.DefaultNotificationPreferences(7)
	if !prefs.AllowsCategory(domain.NotifSaleApproved) {
		t.Fatal("expected sale_approved enabled by default")
	}
	if prefs.AllowsCategory(domain.NotifEventDispatchError) {
		t.Fatal("expected event_dispatch_error disabled by default")
	}

	prefs.AccountScopeMode = domain.AccountScopeNone
	if prefs.AllowsAccount(10, domain.NotifSaleApproved) {
		t.Fatal("expected regular category blocked when account_scope_mode=none")
	}
	if !prefs.AllowsAccount(10, domain.NotifCompanyLifecycle) || !prefs.AllowsCategory(domain.NotifCompanyLifecycle) {
		t.Fatal("expected company_lifecycle mandatory regardless of scope=none")
	}
	if !prefs.AllowsAccount(10, domain.NotifAPIKeyLifecycle) || !prefs.AllowsCategory(domain.NotifAPIKeyLifecycle) {
		t.Fatal("expected api_key_lifecycle mandatory regardless of scope=none")
	}

	prefs.AccountScopeMode = domain.AccountScopeCustom
	prefs.EnabledAccountIDs = []int64{3, 8}
	if !prefs.AllowsAccount(8, domain.NotifLeadCreated) {
		t.Fatal("expected custom account 8 allowed")
	}
	if prefs.AllowsAccount(5, domain.NotifLeadCreated) {
		t.Fatal("expected custom account 5 blocked")
	}
}
