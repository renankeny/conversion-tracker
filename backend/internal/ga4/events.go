package ga4

import (
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// mapEventName traduz o nome de evento do TrackFind para o padrao GA4.
// Eventos customizados nao listados sao higienizados para snake_case valido.
func mapEventName(eventName string) string {
	switch eventName {
	case domain.EventPurchase:
		return "purchase"
	case domain.EventRefund, domain.EventChargeback:
		return "refund"
	case domain.EventLead, domain.EventLeadSubmitted:
		return "generate_lead"
	case "QualifiedLead":
		return "qualified_lead"
	case domain.EventPageView:
		return "page_view"
	case domain.EventViewContent:
		return "view_item"
	case domain.EventAddToCart:
		return "add_to_cart"
	case domain.EventInitiateCheckout:
		return "begin_checkout"
	case domain.EventAddShippingInfo:
		return "add_shipping_info"
	case domain.EventAddPaymentInfo:
		return "add_payment_info"
	case domain.EventAddToWishlist:
		return "add_to_wishlist"
	case domain.EventCompleteRegistration:
		return "sign_up"
	case "Search":
		return "search"
	case "SubmitApplication":
		return "submit_application"
	case domain.EventContact:
		return "contact"
	case "Subscribe":
		return "subscribe"
	case "FindLocation":
		return "find_location"
	case "Schedule":
		return "schedule"
	case "StartTrial":
		return "start_trial"
	case "Donate":
		return "donate"
	case "CustomizeProduct":
		return "customize_product"
	case domain.EventScroll25:
		return "scroll_25"
	case domain.EventScroll50:
		return "scroll_50"
	case domain.EventScroll75:
		return "scroll_75"
	case domain.EventScroll90:
		return "scroll_90"
	case domain.EventTimer1Min:
		return "timer_1min"
	case domain.EventPlayVideo:
		return "play_video"
	case domain.EventViewVideo25:
		return "view_video_25"
	case domain.EventViewVideo50:
		return "view_video_50"
	case domain.EventViewVideo75:
		return "view_video_75"
	case domain.EventViewVideo90:
		return "view_video_90"
	}
	return sanitizeGA4EventName(eventName)
}

var ga4AccentReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a", "Á", "A", "À", "A", "Ã", "A", "Â", "A", "Ä", "A",
	"é", "e", "è", "e", "ê", "e", "ë", "e", "É", "E", "È", "E", "Ê", "E", "Ë", "E",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o", "Ó", "O", "Ò", "O", "Ô", "O", "Õ", "O", "Ö", "O",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
	"ç", "c", "Ç", "C", "ñ", "n", "Ñ", "N",
)

func sanitizeGA4EventName(raw string) string {
	s := ga4AccentReplacer.Replace(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	var b strings.Builder
	var prevLowerOrDigit bool
	lastUnderscore := false
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			if prevLowerOrDigit && !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			prevLowerOrDigit = false
			lastUnderscore = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevLowerOrDigit = true
			lastUnderscore = false
		default:
			if b.Len() > 0 && !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
			prevLowerOrDigit = false
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return ""
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "ev_" + out
	}
	if len(out) > 40 {
		out = strings.TrimRight(out[:40], "_")
	}
	return out
}
