package domain

import (
	"encoding/json"
	"testing"
)

func TestDashboardConfig_PortalDefaultsAndPermissions(t *testing.T) {
	var zero DashboardConfig
	if !zero.IsPublicCRMEnabled() {
		t.Fatal("expected CRM to be enabled by default on existing pixels")
	}
	if zero.IsPublicCRMEditable() {
		t.Fatal("expected CRM edit mode to be disabled by default")
	}
	if !zero.IsPublicConversationsEnabled() {
		t.Fatal("expected Conversations to be enabled by default on existing pixels")
	}
	if !zero.IsPublicSalesGoalEnabled() {
		t.Fatal("expected SalesGoal visibility to be enabled by default")
	}
	if NormalizeSalesGoalType("count") != "count" || NormalizeSalesGoalType("other") != "revenue" {
		t.Fatal("unexpected NormalizeSalesGoalType behavior")
	}

	var parsed DashboardConfig
	if err := json.Unmarshal([]byte(`{"enabled_cards":["Purchase"],"public_allow_crm_edit":true}`), &parsed); err != nil {
		t.Fatal(err)
	}
	if !parsed.IsPublicCRMEnabled() || !parsed.IsPublicCRMEditable() {
		t.Fatal("expected CRM editable when CRM is shown by default and allow_edit=true")
	}

	f := false
	parsed.PublicShowCRM = &f
	if parsed.IsPublicCRMEnabled() {
		t.Fatal("expected CRM disabled when PublicShowCRM=false")
	}
	if parsed.IsPublicCRMEditable() {
		t.Fatal("expected CRM edit blocked when CRM tab itself is hidden")
	}

	parsed.PublicShowConversations = &f
	if parsed.IsPublicConversationsEnabled() {
		t.Fatal("expected Conversations disabled when PublicShowConversations=false")
	}
}
