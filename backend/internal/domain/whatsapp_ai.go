package domain

import "time"

const (
	AIEventQualified = "qualified"
	AIEventPurchase  = "purchase"

	AIEventStatusActive    = "active"
	AIEventStatusContested = "contested"
)

type AILearnedRule struct {
	ID          string    `json:"id"`
	EventType   string    `json:"event_type"`
	Rule        string    `json:"rule"`
	AuthorEmail string    `json:"author_email,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type AccountAISettings struct {
	AccountID        int64           `json:"account_id"`
	Enabled          bool            `json:"enabled"`
	HasOpenAIKey     bool            `json:"has_openai_key"`
	Model            string          `json:"model"`
	QualifiedStageID *int64          `json:"qualified_stage_id"`
	SaleStageID      *int64          `json:"sale_stage_id"`
	BasePrompt       string          `json:"base_prompt"`
	DefaultPrompt    string          `json:"default_prompt,omitempty"`
	LearnedRules     []AILearnedRule `json:"learned_rules"`
	LastRunAt        *time.Time      `json:"last_run_at,omitempty"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type WhatsAppAIEvent struct {
	ID                int64      `json:"id"`
	AccountID         int64      `json:"account_id"`
	ChatID            int64      `json:"chat_id"`
	CRMLeadID         *int64     `json:"crm_lead_id,omitempty"`
	TrackedUserID     *int64     `json:"tracked_user_id,omitempty"`
	EventType         string     `json:"event_type"`
	EvidenceMessageID *int64     `json:"evidence_message_id,omitempty"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	AIReason          string     `json:"ai_reason"`
	CRMPrevStageID    *int64     `json:"crm_prev_stage_id,omitempty"`
	CRMNewStageID     *int64     `json:"crm_new_stage_id,omitempty"`
	CRMPrevValue      float64    `json:"crm_prev_value"`
	DispatchedEventID string     `json:"dispatched_event_id,omitempty"`
	Deduplicated      bool       `json:"deduplicated"`
	Status            string     `json:"status"`
	ContestReason     string     `json:"contest_reason,omitempty"`
	ContestedByUserID *int64     `json:"contested_by_user_id,omitempty"`
	ContestedAt       *time.Time `json:"contested_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}
