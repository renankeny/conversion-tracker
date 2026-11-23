package domain

import "time"

// ConversationAuditItem representa uma linha da tabela executiva de Conversas
// (cruzando atribuição de mídia paga/orgânica rastreada, funil CRM e SLA de atendimento).
type ConversationAuditItem struct {
	ChatID            int64      `json:"chat_id"`
	InstanceID        int64      `json:"instance_id"`
	InstanceName      string     `json:"instance_name"`
	PixelID           int64      `json:"pixel_id"`
	PixelName         string     `json:"pixel_name"`
	Currency          string     `json:"currency"`
	TrackedUserID     *int64     `json:"tracked_user_id,omitempty"`
	CRMLeadID         *int64     `json:"crm_lead_id,omitempty"`
	LeadName          string     `json:"lead_name"`
	Phone             string     `json:"phone"`
	Email             string     `json:"email,omitempty"`
	ProfilePic        *string    `json:"profile_pic,omitempty"`
	OriginCategory    string     `json:"origin_category"`
	OriginType        string     `json:"origin_type"`
	CampaignName      string     `json:"campaign_name,omitempty"`
	AdName            string     `json:"ad_name,omitempty"`
	UTMSource         string     `json:"utm_source,omitempty"`
	TrackableLinkName string     `json:"trackable_link_name,omitempty"`
	StageID           *int64     `json:"stage_id,omitempty"`
	StageName         string     `json:"stage_name,omitempty"`
	StageColor        string     `json:"stage_color,omitempty"`
	LeadValue         float64    `json:"lead_value"`
	IsWon             bool       `json:"is_won"`
	Tags              []CRMTag   `json:"tags"`
	AIQualified       bool       `json:"ai_qualified"`
	SLAStatus         string     `json:"sla_status"`
	TotalMessages     int        `json:"total_messages"`
	InboundMessages   int        `json:"inbound_messages"`
	OutboundMessages  int        `json:"outbound_messages"`
	FirstResponseSec  *int       `json:"first_response_sec"`
	AvgResponseSec    *int       `json:"avg_response_sec"`
	FirstInboundAt    *time.Time `json:"first_inbound_at,omitempty"`
	LastInboundAt     *time.Time `json:"last_inbound_at,omitempty"`
	LastOutboundAt    *time.Time `json:"last_outbound_at,omitempty"`
	LastMessageTime   time.Time  `json:"last_message_time"`
	CreatedAt         time.Time  `json:"created_at"`
}

type ConversationAuditKPIs struct {
	TotalConversations  int     `json:"total_conversations"`
	WaitingReplyCount   int     `json:"waiting_reply_count"`
	UnansweredCount     int     `json:"unanswered_count"`
	AvgFirstResponseSec *int    `json:"avg_first_response_sec"`
	WonCount            int     `json:"won_count"`
	WonRevenue          float64 `json:"won_revenue"`
}

type ConversationOriginCounts struct {
	All       int `json:"all"`
	MetaAds   int `json:"meta_ads"`
	GoogleAds int `json:"google_ads"`
	TikTokAds int `json:"tiktok_ads"`
	BioLink   int `json:"bio_link"`
	Site      int `json:"site"`
}

type ConversationAuditFilter struct {
	AccountID      int64
	PixelName      string
	PixelID        int64
	InstanceID     int64
	From           time.Time
	To             time.Time
	Search         string
	SLAStatus      string
	StageID        int64
	TagID          int64
	OriginCategory string
	Campaign       string
	UTM            string
	SortBy         string
	SortDir        string
	Limit          int
	Offset         int
}

type ConversationAuditResponse struct {
	Items        []ConversationAuditItem  `json:"items"`
	Total        int                      `json:"total"`
	KPIs         ConversationAuditKPIs    `json:"kpis"`
	OriginCounts ConversationOriginCounts `json:"origin_counts"`
	Campaigns    []string                 `json:"campaigns"`
	UTMs         []string                 `json:"utms"`
}
