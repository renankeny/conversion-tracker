package domain

import "time"

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
