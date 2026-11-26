package domain

import "time"

const (
	// Categorias configuráveis pelo usuário (com checkbox + botão de teste)
	NotifWhatsAppConnection        = "whatsapp_connection"
	NotifLeadCreated               = "lead_created"
	NotifAILeadQualified           = "ai_lead_qualified"
	NotifSaleApproved              = "sale_approved"
	NotifSaleRefunded              = "sale_refunded"
	NotifMetaAccountLifecycle      = "meta_account_lifecycle"
	NotifGoogleAdsAccountLifecycle = "google_ads_account_lifecycle"
	NotifPixelLifecycle            = "pixel_lifecycle"
	NotifEventDispatchError        = "event_dispatch_error"

	// Categorias fixas de auditoria/segurança (sempre ativas, ocultas dos checkboxes)
	NotifCompanyLifecycle = "company_lifecycle"
	NotifAPIKeyLifecycle  = "api_key_lifecycle"

	AccountScopeAll    = "all"
	AccountScopeCustom = "custom"
	AccountScopeNone   = "none"
)

func IsMandatoryNotificationCategory(category string) bool {
	return category == NotifCompanyLifecycle || category == NotifAPIKeyLifecycle
}

func IsConfigurableNotificationCategory(category string) bool {
	switch category {
	case NotifWhatsAppConnection,
		NotifLeadCreated,
		NotifAILeadQualified,
		NotifSaleApproved,
		NotifSaleRefunded,
		NotifMetaAccountLifecycle,
		NotifGoogleAdsAccountLifecycle,
		NotifPixelLifecycle,
		NotifEventDispatchError:
		return true
	default:
		return false
	}
}

func NormalizeAccountScopeMode(mode string) string {
	switch mode {
	case AccountScopeCustom:
		return AccountScopeCustom
	case AccountScopeNone:
		return AccountScopeNone
	default:
		return AccountScopeAll
	}
}

type NotificationPreferences struct {
	UserID                    int64     `json:"user_id"`
	AccountScopeMode          string    `json:"account_scope_mode"`
	EnabledAccountIDs         []int64   `json:"enabled_account_ids"`
	WhatsAppConnection        bool      `json:"whatsapp_connection"`
	LeadCreated               bool      `json:"lead_created"`
	AILeadQualified           bool      `json:"ai_lead_qualified"`
	SaleApproved              bool      `json:"sale_approved"`
	SaleRefunded              bool      `json:"sale_refunded"`
	MetaAccountLifecycle      bool      `json:"meta_account_lifecycle"`
	GoogleAdsAccountLifecycle bool      `json:"google_ads_account_lifecycle"`
	PixelLifecycle            bool      `json:"pixel_lifecycle"`
	EventDispatchError        bool      `json:"event_dispatch_error"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

func DefaultNotificationPreferences(userID int64) NotificationPreferences {
	return NotificationPreferences{
		UserID:                    userID,
		AccountScopeMode:          AccountScopeAll,
		EnabledAccountIDs:         []int64{},
		WhatsAppConnection:        true,
		LeadCreated:               true,
		AILeadQualified:           true,
		SaleApproved:              true,
		SaleRefunded:              true,
		MetaAccountLifecycle:      true,
		GoogleAdsAccountLifecycle: true,
		PixelLifecycle:            true,
		EventDispatchError:        false,
	}
}

func (p NotificationPreferences) AllowsCategory(category string) bool {
	if IsMandatoryNotificationCategory(category) {
		return true
	}
	switch category {
	case NotifWhatsAppConnection:
		return p.WhatsAppConnection
	case NotifLeadCreated:
		return p.LeadCreated
	case NotifAILeadQualified:
		return p.AILeadQualified
	case NotifSaleApproved:
		return p.SaleApproved
	case NotifSaleRefunded:
		return p.SaleRefunded
	case NotifMetaAccountLifecycle:
		return p.MetaAccountLifecycle
	case NotifGoogleAdsAccountLifecycle:
		return p.GoogleAdsAccountLifecycle
	case NotifPixelLifecycle:
		return p.PixelLifecycle
	case NotifEventDispatchError:
		return p.EventDispatchError
	default:
		return false
	}
}

func (p NotificationPreferences) AllowsAccount(accountID int64, category string) bool {
	if IsMandatoryNotificationCategory(category) || accountID <= 0 {
		return true
	}
	switch NormalizeAccountScopeMode(p.AccountScopeMode) {
	case AccountScopeNone:
		return false
	case AccountScopeCustom:
		for _, id := range p.EnabledAccountIDs {
			if id == accountID {
				return true
			}
		}
		return false
	default:
		return true
	}
}

type PushSubscription struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Endpoint    string    `json:"endpoint"`
	P256dh      string    `json:"-"`
	AuthSecret  string    `json:"-"`
	DeviceLabel string    `json:"device_label"`
	UserAgent   string    `json:"user_agent,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

type UserNotification struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	AccountID   *int64    `json:"account_id,omitempty"`
	AccountName string    `json:"account_name,omitempty"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	ActionURL   string    `json:"action_url"`
	IsRead      bool      `json:"is_read"`
	CreatedAt   time.Time `json:"created_at"`
}

type NotificationsOverview struct {
	VAPIDPublicKey string                  `json:"vapid_public_key"`
	UnreadCount    int                     `json:"unread_count"`
	Preferences    NotificationPreferences `json:"preferences"`
	Subscriptions  []PushSubscription      `json:"subscriptions"`
	Notifications  []UserNotification      `json:"notifications"`
}
