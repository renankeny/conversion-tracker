package domain

import "time"

const (
	WhatsAppOriginMetaCTWA       = "meta_ads_ctwa"
	WhatsAppOriginMetaLP         = "meta_ads_lp"
	WhatsAppOriginGoogleAds      = "google_ads"
	WhatsAppOriginTikTokAds      = "tiktok_ads"
	WhatsAppOriginBioLinkOrganic = "bio_link_organic"
	WhatsAppOriginWebsiteOrganic = "website_organic"

	MediaStatusNone        = "none"
	MediaStatusPending     = "pending"
	MediaStatusReady       = "ready"
	MediaStatusExpired     = "expired"
	MediaStatusFailed      = "failed"
	MediaStatusSkippedSize = "skipped_size"
)

// WhatsAppServer representa o servidor Evolution Go configurado por uma conta/cliente.
type WhatsAppServer struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	APIKey    string    `json:"-"`
	HasAPIKey bool      `json:"has_api_key"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WhatsAppInstance representa uma instância/número conectado na Evolution Go.
type WhatsAppInstance struct {
	ID                 int64     `json:"id"`
	AccountID          int64     `json:"account_id"`
	PixelID            int64     `json:"pixel_id"`
	PixelName          string    `json:"pixel_name,omitempty"`
	ServerID           int64     `json:"server_id"`
	Name               string    `json:"name"`
	InstanceName       string    `json:"instance_name"`
	EvolutionUUID      string    `json:"evolution_uuid"`
	InstanceToken      string    `json:"-"`
	WebhookSecret      string    `json:"webhook_secret,omitempty"`
	OwnerJID           string    `json:"owner_jid"`
	ProfileName        string    `json:"profile_name"`
	ProfilePicURL      string    `json:"profile_pic_url"`
	ShareToken         *string   `json:"share_token,omitempty"`
	ConnectionStatus   string    `json:"connection_status"`
	PurchaseTrigger    string    `json:"purchase_trigger"`
	LeadTrigger        string    `json:"lead_trigger"`
	CustomEventName    string    `json:"custom_event_name"`
	CustomEventTrigger string    `json:"custom_event_trigger"`
	AutoCreateCRMLead  bool      `json:"auto_create_crm_lead"`
	DefaultCRMStageID  *int64    `json:"default_crm_stage_id"`
	Enabled            bool      `json:"enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	ServerURL    string `json:"server_url,omitempty"`
	ServerAPIKey string `json:"-"`
}

// WhatsAppChat representa uma conversa elegível (CTWA ou Protocolo Web) vinculada a um lead.
type WhatsAppChat struct {
	ID              int64     `json:"id"`
	AccountID       int64     `json:"account_id"`
	InstanceID      int64     `json:"instance_id"`
	InstanceName    string    `json:"instance_name,omitempty"`
	RemoteJid       string    `json:"remote_jid"`
	TrackedUserID   *int64    `json:"tracked_user_id,omitempty"`
	CRMLeadID       *int64    `json:"crm_lead_id,omitempty"`
	Phone           string    `json:"phone"`
	LidJid          string    `json:"lid_jid"`
	PushName        string    `json:"push_name"`
	ContactName     *string   `json:"contact_name"`
	ProfilePic      *string   `json:"profile_pic"`
	OriginType      string    `json:"origin_type"`
	CtwaClid        string    `json:"ctwa_clid"`
	SourceID        string    `json:"source_id"`
	PageID          string    `json:"page_id"`
	SourceApp       string    `json:"source_app"`
	AdName          string    `json:"ad_name"`
	AdsetID         string    `json:"adset_id"`
	AdsetName       string    `json:"adset_name"`
	CampaignID      string    `json:"campaign_id"`
	CampaignName    string    `json:"campaign_name"`
	AdThumbnailURL  string    `json:"ad_thumbnail_url"`
	UnreadCount     int       `json:"unread_count"`
	LastMessageTime time.Time `json:"last_message_time"`
	Metadata        []byte    `json:"metadata,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Source string `json:"source,omitempty"`
}

// WhatsAppMessage representa uma única mensagem (texto permanente + mídia temporária 14d).
type WhatsAppMessage struct {
	ID                int64      `json:"id"`
	ChatID            int64      `json:"chat_id"`
	WhatsAppMessageID string     `json:"whatsapp_message_id"`
	IsFromMe          bool       `json:"is_from_me"`
	MessageType       string     `json:"message_type"`
	Content           string     `json:"content"`
	HasMedia          bool       `json:"has_media"`
	MediaUrl          *string    `json:"media_url,omitempty"`
	MediaMimeType     *string    `json:"media_mime_type,omitempty"`
	MediaObjectKey    string     `json:"-"`
	MediaStatus       string     `json:"media_status"`
	MediaSizeBytes    int64      `json:"media_size_bytes"`
	MediaDurationSec  int        `json:"media_duration_sec"`
	MediaFilename     string     `json:"media_filename"`
	MediaExpiresAt    *time.Time `json:"media_expires_at,omitempty"`
	MediaStreamURL    string     `json:"media_stream_url,omitempty"`
	Timestamp         time.Time  `json:"timestamp"`
	CreatedAt         time.Time  `json:"created_at"`
}
