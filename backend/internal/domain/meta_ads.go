package domain

import "time"

type MetaAdAccount struct {
	ID          int64     `json:"id"`
	AccountID   int64     `json:"account_id"`
	Label       string    `json:"label"`
	AdAccountID string    `json:"ad_account_id"`
	AdsToken    string    `json:"ads_token,omitempty"` // Omitted in some responses for security
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
