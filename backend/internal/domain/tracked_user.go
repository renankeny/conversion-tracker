package domain

import "time"

// TrackedUser representa um visitante/lead identificado pelo external_id
// (gerado no client com 10 caracteres [A-Z0-9], igual ao sistema legado).
//
// JSON tags em snake_case pra consistência com o resto da API e com
// as convenções esperadas pelo frontend.
type TrackedUser struct {
	ID         int64  `json:"id"`
	ExternalID string `json:"external_id"`
	IDWpp      string `json:"id_wpp"`
	PixelName  string `json:"pixel_name"`

	FBP             string `json:"fbp"`
	FBC             string `json:"fbc"`
	GCLID           string `json:"gclid"`
	ClientIP        string `json:"client_ip"`
	ClientUserAgent string `json:"client_user_agent"`

	Country    string `json:"country"`
	State      string `json:"state"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`

	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`

	URL string `json:"url"`

	FirstUTMSource   string `json:"first_utm_source"`
	FirstUTMMedium   string `json:"first_utm_medium"`
	FirstUTMCampaign string `json:"first_utm_campaign"`
	FirstUTMContent  string `json:"first_utm_content"`
	FirstUTMTerm     string `json:"first_utm_term"`

	LastUTMSource   string `json:"last_utm_source"`
	LastUTMMedium   string `json:"last_utm_medium"`
	LastUTMCampaign string `json:"last_utm_campaign"`
	LastUTMContent  string `json:"last_utm_content"`
	LastUTMTerm     string `json:"last_utm_term"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
