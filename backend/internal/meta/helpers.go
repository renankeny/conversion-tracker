package meta

import (
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func isWebEvent(name string) bool {
	switch name {
	case domain.EventPageView, domain.EventViewContent, domain.EventLead,
		domain.EventContact, domain.EventInitiateCheckout, domain.EventAddShippingInfo, domain.EventAddPaymentInfo,
		domain.EventAddToCart, domain.EventAddToWishlist,
		domain.EventCompleteRegistration, domain.EventPurchase,
		domain.EventScroll25, domain.EventScroll50, domain.EventScroll75, domain.EventScroll90,
		domain.EventTimer1Min, domain.EventPlayVideo,
		domain.EventViewVideo25, domain.EventViewVideo50, domain.EventViewVideo75, domain.EventViewVideo90:
		return true
	}
	return false
}

func setHashed(m map[string]any, key, val string) {
	if h := util.SHA256Hex(val); h != "" {
		m[key] = []string{h}
	}
}

func setIfNotEmpty(m map[string]any, key, val string) {
	if val != "" {
		m[key] = val
	}
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func validActionSource(s string) string {
	switch s {
	case "website", "app", "phone_call", "chat", "email", "other", "system_generated", "business_messaging":
		return s
	default:
		return "website"
	}
}

