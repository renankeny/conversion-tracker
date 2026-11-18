package domain

import (
	"strings"
	"time"
)

const CRMTriggerNone = "none"

type CRMStage struct {
	ID           int64     `json:"id"`
	AccountID    int64     `json:"account_id"`
	Name         string    `json:"name"`
	Color        string    `json:"color"`
	OrderIndex              int       `json:"order_index"`
	TriggerEvent            string    `json:"trigger_event"`
	TriggerEventCTWA        string    `json:"trigger_event_ctwa"`
	GoogleAdsAccountID      *int64    `json:"google_ads_account_id,omitempty"`
	GoogleAdsActionID       string    `json:"google_ads_action_id,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type CRMLead struct {
	ID            int64     `json:"id"`
	AccountID     int64     `json:"account_id"`
	TrackedUserID int64     `json:"tracked_user_id"`
	StageID       *int64    `json:"stage_id"`
	Value         float64   `json:"value"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	TrackedUser *TrackedUser `json:"tracked_user,omitempty"`
	Tags        []CRMTag     `json:"tags,omitempty"`
	Notes       []CRMNote    `json:"notes,omitempty"`
	HasNotes    bool         `json:"has_notes"`
}

type CRMTag struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

type CRMNote struct {
	ID        int64     `json:"id"`
	CRMLeadID int64     `json:"crm_lead_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// IsWhatsAppOrCTWAUser indica se o lead entrou por WhatsApp/CTWA para seleção da coluna de gatilho no CRM.
func IsWhatsAppOrCTWAUser(tu *TrackedUser) bool {
	if tu == nil {
		return false
	}
	return tu.CtwaClid != "" || tu.WaAdID != "" || strings.HasPrefix(tu.ExternalID, "wpp:")
}

// EffectiveStageTrigger resolve o evento efetivo da etapa do CRM para leads Web ou WhatsApp/CTWA,
// respeitando o valor sentinela "none" (desativado explicitamente).
func EffectiveStageTrigger(stage *CRMStage, isCTWA bool) string {
	if stage == nil {
		return ""
	}
	webEv := strings.TrimSpace(stage.TriggerEvent)
	if strings.EqualFold(webEv, CRMTriggerNone) {
		webEv = ""
	}
	if !isCTWA {
		return webEv
	}
	ctwaRaw := strings.TrimSpace(stage.TriggerEventCTWA)
	if strings.EqualFold(ctwaRaw, CRMTriggerNone) {
		return ""
	}
	ev := ctwaRaw
	if ev == "" {
		ev = webEv
	}
	if ev == EventLead {
		return EventLeadSubmitted
	}
	return ev
}
