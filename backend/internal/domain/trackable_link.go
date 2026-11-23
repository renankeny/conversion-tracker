package domain

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// GenerateTrackableLinkSlug gera um identificador numérico único de 10 dígitos,
// começando pelo ID da empresa (account_id) completado com dígitos aleatórios.
func GenerateTrackableLinkSlug(accountID int64) string {
	const totalLen = 10
	if accountID <= 0 {
		accountID = 1
	}
	prefix := strconv.FormatInt(accountID, 10)
	rem := totalLen - len(prefix)
	if rem <= 0 {
		rem = 2
	}
	digits := make([]byte, rem)
	ten := big.NewInt(10)
	for i := range digits {
		n, err := rand.Int(rand.Reader, ten)
		if err != nil {
			digits[i] = '0'
			continue
		}
		digits[i] = byte('0') + byte(n.Int64())
	}
	return prefix + string(digits)
}

const (
	LinkRoutingAutoPixel        = "auto_pixel"
	LinkRoutingSpecificInstance = "specific_instance"
	LinkRoutingManualPhone      = "manual_phone"

	LinkTriggerOnFirstMessage = "on_first_message"
	LinkTriggerOnClick        = "on_click"
)

func NormalizeLinkRoutingMode(m string) string {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case LinkRoutingSpecificInstance:
		return LinkRoutingSpecificInstance
	case LinkRoutingManualPhone:
		return LinkRoutingManualPhone
	default:
		return LinkRoutingAutoPixel
	}
}

func NormalizeLinkTriggerMode(m string) string {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case LinkTriggerOnClick:
		return LinkTriggerOnClick
	default:
		return LinkTriggerOnFirstMessage
	}
}

func NormalizeLinkEventName(ev string) string {
	switch strings.TrimSpace(ev) {
	case EventLead, EventContact, EventPageView, EventViewContent, EventInitiateCheckout, EventSchedule, EventQualifiedLead:
		return strings.TrimSpace(ev)
	default:
		return EventLead
	}
}

type TrackableLink struct {
	ID                 int64     `json:"id"`
	AccountID          int64     `json:"account_id"`
	PixelID            int64     `json:"pixel_id"`
	PixelName          string    `json:"pixel_name,omitempty"`
	PixelLabel         string    `json:"pixel_label,omitempty"`
	TrackingHost       string    `json:"tracking_host,omitempty"`
	Slug               string    `json:"slug"`
	Name               string    `json:"name"`
	PublicURL          string    `json:"public_url,omitempty"`
	RoutingMode        string    `json:"routing_mode"`
	WhatsAppInstanceID *int64    `json:"whatsapp_instance_id"`
	InstanceName       string    `json:"instance_name,omitempty"`
	InstanceProvider   string    `json:"instance_provider,omitempty"`
	InstanceStatus     string    `json:"instance_status,omitempty"`
	FallbackPhone      string    `json:"fallback_phone"`
	PrefillMessage     string    `json:"prefill_message"`
	UTMSource          string    `json:"utm_source"`
	UTMMedium          string    `json:"utm_medium"`
	UTMCampaign        string    `json:"utm_campaign"`
	UTMContent         string    `json:"utm_content"`
	UTMTerm            string    `json:"utm_term"`
	SendEventsEnabled  bool      `json:"send_events_enabled"`
	EventTriggerMode   string    `json:"event_trigger_mode"`
	EventName          string    `json:"event_name"`
	SendMeta           bool      `json:"send_meta"`
	SendTikTok         bool      `json:"send_tiktok"`
	SendGoogleAds      bool      `json:"send_google_ads"`
	SendGA4            bool      `json:"send_ga4"`
	Enabled            bool      `json:"enabled"`
	ClicksCount        int64     `json:"clicks_count"`
	ChatsCount         int64     `json:"chats_count"`
	SalesCount         int64     `json:"sales_count"`
	SalesRevenue       float64   `json:"sales_revenue"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (l *TrackableLink) AllowedChannels() []string {
	if l == nil || !l.SendEventsEnabled {
		return nil
	}
	out := make([]string, 0, 4)
	if l.SendMeta {
		out = append(out, "meta")
	}
	if l.SendTikTok {
		out = append(out, "tiktok")
	}
	if l.SendGoogleAds {
		out = append(out, "google_ads")
	}
	if l.SendGA4 {
		out = append(out, "ga4")
	}
	return out
}

func (l *TrackableLink) BuildPublicURL(fallbackBaseURL string) string {
	if l == nil || l.Slug == "" {
		return ""
	}
	host := strings.TrimSpace(l.TrackingHost)
	if host != "" {
		host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
		host = strings.TrimRight(host, "/")
		return "https://" + host + "/w/" + l.Slug
	}
	base := strings.TrimRight(strings.TrimSpace(fallbackBaseURL), "/")
	base = strings.TrimSuffix(base, "/api/v1")
	if base == "" {
		return "/w/" + l.Slug
	}
	return base + "/w/" + l.Slug
}

type TrackableLinkClick struct {
	ID                 int64     `json:"id"`
	AccountID          int64     `json:"account_id"`
	PixelID            int64     `json:"pixel_id"`
	LinkID             int64     `json:"link_id"`
	TrackedUserID      *int64    `json:"tracked_user_id,omitempty"`
	ResolvedInstanceID *int64    `json:"resolved_instance_id,omitempty"`
	ProtocolCode       string    `json:"protocol_code"`
	ClientIP           string    `json:"client_ip"`
	IsBot              bool      `json:"is_bot"`
	MatchedChatID      *int64    `json:"matched_chat_id,omitempty"`
	ClickEventSent     bool      `json:"click_event_sent"`
	MessageEventSent   bool      `json:"message_event_sent"`
	CreatedAt          time.Time `json:"created_at"`
}

type TrackableLinksOverview struct {
	TotalLinks   int             `json:"total_links"`
	TotalClicks  int64           `json:"total_clicks"`
	TotalChats   int64           `json:"total_chats"`
	TotalSales   int64           `json:"total_sales"`
	TotalRevenue float64         `json:"total_revenue"`
	Links        []TrackableLink `json:"links"`
}
