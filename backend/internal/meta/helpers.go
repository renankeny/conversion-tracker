package meta

import (
	"fmt"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func isWebEvent(name string) bool {
	switch name {
	case domain.EventPageView, domain.EventViewContent, domain.EventLead,
		domain.EventContact, domain.EventInitiateCheckout, domain.EventAddShippingInfo, domain.EventAddPaymentInfo,
		domain.EventAddToCart, domain.EventAddToWishlist,
		domain.EventCompleteRegistration, domain.EventPurchase,
		domain.EventSearch, domain.EventSubscribe, domain.EventStartTrial, domain.EventSchedule,
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

func setPhoneHashed(m map[string]any, rawPhone, rawCountry string) {
	norm := util.NormalizePhoneWithCountry(rawPhone, rawCountry)
	if norm == "" {
		return
	}
	if !util.IsSHA256Hex(norm) && len(norm) < 8 {
		return
	}
	setHashed(m, "ph", norm)
}

func setIfNotEmpty(m map[string]any, key, val string) {
	if trimmed := strings.TrimSpace(val); trimmed != "" {
		m[key] = trimmed
	}
}

func isBogusTokenString(s string) bool {
	switch strings.ToLower(s) {
	case "", "null", "undefined", "nan", "none", "[object object]":
		return true
	}
	return false
}

func cleanFBP(raw string) string {
	s := strings.TrimSpace(raw)
	if isBogusTokenString(s) || !strings.HasPrefix(s, "fb.") {
		return ""
	}
	return s
}

func cleanFBC(raw string, eventTimeSec int64) string {
	s := strings.TrimSpace(raw)
	if isBogusTokenString(s) || strings.ContainsAny(s, " \t\r\n") {
		return ""
	}
	if strings.HasPrefix(s, "fb.") {
		return s
	}
	if len(s) >= 12 {
		tsMs := time.Now().UnixMilli()
		if eventTimeSec > 0 {
			tsMs = eventTimeSec * 1000
		}
		return fmt.Sprintf("fb.1.%d.%s", tsMs, s)
	}
	return ""
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

// ResolveEffectiveEvent calcula o action_source e o event_name efetivos aceitos pela Meta CAPI,
// garantindo que website sem URL vire system_generated e que eventos de mensageria/web sejam compatíveis.
func ResolveEffectiveEvent(in domain.SendEventInput) (eventName, actionSrc string, isMessaging bool) {
	actionSrc = validActionSource(in.ActionSource)
	isMessaging = in.CtwaClid != "" && in.PageID != "" &&
		(actionSrc == "business_messaging" || in.ActionSource == "")
	if isMessaging {
		actionSrc = "business_messaging"
	} else {
		if actionSrc == "business_messaging" || (actionSrc == "website" && strings.TrimSpace(in.SourceURL) == "") {
			actionSrc = "system_generated"
		}
	}

	eventName = in.EventName
	if isMessaging {
		if eventName == domain.EventLead {
			eventName = domain.EventLeadSubmitted
		}
		return eventName, actionSrc, true
	}
	switch eventName {
	case domain.EventLeadSubmitted, domain.EventQualifiedLead:
		eventName = domain.EventLead
	case domain.EventOrderCreated:
		eventName = domain.EventInitiateCheckout
	case domain.EventCartAbandoned:
		eventName = domain.EventAddToCart
	}
	return eventName, actionSrc, false
}
