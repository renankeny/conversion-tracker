package tiktok

import (
	"log/slog"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func buildPayload(pixelID, testCode, partnerName, eventName string, in domain.SendEventInput) map[string]any {
	user := map[string]any{}

	country := util.NormalizeCountry(in.Country)

	// Campos com hash SHA-256 obrigatório na TikTok Events API 2.0 (v1.3)
	setHashed(user, "email", util.NormalizeEmail(in.Email))
	setHashed(user, "phone", util.NormalizePhoneE164(in.Phone))
	setHashed(user, "external_id", strings.ToLower(strings.TrimSpace(in.ExternalID)))
	setHashed(user, "first_name", util.NormalizeName(in.FirstName))
	setHashed(user, "last_name", util.NormalizeName(in.LastName))
	setHashed(user, "zip_code", normalizeTikTokZip(in.PostalCode, country))

	// Campos em texto plano (NÃO hashear na TikTok Events API 2.0)
	setIfNotEmpty(user, "city", util.NormalizeCity(in.City))
	setIfNotEmpty(user, "state", util.NormalizeStateISO(in.State))
	setIfNotEmpty(user, "country", country)
	setIfNotEmpty(user, "ttclid", in.TTClid)
	setIfNotEmpty(user, "ttp", in.TTP)
	setIfNotEmpty(user, "ip", in.ClientIP)
	setIfNotEmpty(user, "user_agent", in.ClientUserAgent)
	user["locale"] = "pt-BR"

	page := map[string]any{}
	if in.SourceURL != "" {
		page["url"] = in.SourceURL
	}
	if in.Referrer != "" {
		page["referrer"] = in.Referrer
	}

	properties := buildProperties(eventName, in)

	data := map[string]any{
		"event":      eventName,
		"event_time": in.EventTime,
		"event_id":   in.EventID,
		"user":       user,
	}
	if len(page) > 0 {
		data["page"] = page
	}
	if len(properties) > 0 {
		data["properties"] = properties
	}

	payload := map[string]any{
		"event_source":    eventSrc,
		"event_source_id": pixelID,
		"data":            []any{data},
	}
	if testCode != "" {
		payload["test_event_code"] = testCode
	}
	if partnerName != "" {
		payload["partner_name"] = partnerName
	}
	return payload
}

func buildProperties(eventName string, in domain.SendEventInput) map[string]any {
	properties := map[string]any{}
	contentType := orDefault(in.ContentType, "product")

	if len(in.AggregatedItems) > 0 {
		contents := make([]map[string]any, 0, len(in.AggregatedItems))
		firstID := ""
		for i, agg := range in.AggregatedItems {
			cid := firstNonEmpty(agg.ProductID, agg.OrderID, in.ContentID, in.PixelName)
			if i == 0 {
				firstID = cid
			}
			contents = append(contents, map[string]any{
				"content_id":   cid,
				"content_type": contentType,
				"content_name": firstNonEmpty(agg.ProductTitle, in.ContentName, in.PixelName),
				"price":        agg.Value,
				"quantity":     1,
			})
		}
		properties["content_id"] = firstID
		properties["content_type"] = contentType
		properties["contents"] = contents
	} else if in.ContentID != "" && eventName != "Pageview" {
		rawIDs := strings.Split(in.ContentID, ",")
		ids := make([]string, 0, len(rawIDs))
		for _, id := range rawIDs {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				ids = append(ids, trimmed)
			}
		}
		if len(ids) == 0 {
			ids = []string{in.ContentID}
		}
		properties["content_id"] = ids[0]
		properties["content_type"] = contentType

		names := strings.Split(in.ContentName, ",")
		contents := make([]map[string]any, 0, len(ids))
		qty := 1
		if len(ids) == 1 && in.AggregatedCount > 1 {
			qty = in.AggregatedCount
		}
		unitPrice := 0.0
		if in.Value != nil {
			unitPrice = *in.Value / float64(len(ids)*qty)
		}
		for idx, cid := range ids {
			cName := in.ContentName
			if idx < len(names) && strings.TrimSpace(names[idx]) != "" {
				cName = strings.TrimSpace(names[idx])
			}
			item := map[string]any{
				"content_id":   cid,
				"content_type": contentType,
				"content_name": firstNonEmpty(cName, in.PixelName),
				"quantity":     qty,
			}
			if in.Value != nil {
				item["price"] = unitPrice
			}
			contents = append(contents, item)
		}
		properties["contents"] = contents
	}

	if in.ContentName != "" && eventName != "Pageview" {
		properties["content_name"] = in.ContentName
	}
	if in.Value != nil {
		properties["value"] = *in.Value
		properties["currency"] = orDefault(in.Currency, "BRL")
	} else if eventName == "Purchase" {
		properties["value"] = 0.0
		properties["currency"] = orDefault(in.Currency, "BRL")
	}
	if in.OrderID != "" {
		properties["order_id"] = in.OrderID
	}
	return properties
}

func normalizeTikTokZip(postal, country string) string {
	zip := util.NormalizeZip(postal)
	if zip == "" {
		return ""
	}
	if country == "us" && len(zip) > 5 {
		return zip[:5]
	}
	return zip
}

func mapEventName(name string) string {
	switch name {
	case domain.EventPageView:
		return "Pageview"
	case domain.EventViewContent:
		return "ViewContent"
	case domain.EventAddToCart:
		return "AddToCart"
	case domain.EventAddToWishlist:
		return "AddToWishlist"
	case domain.EventInitiateCheckout:
		return "InitiateCheckout"
	case domain.EventAddPaymentInfo:
		return "AddPaymentInfo"
	case domain.EventAddShippingInfo:
		return "AddShippingInfo"
	case domain.EventCompleteRegistration:
		return "CompleteRegistration"
	case domain.EventContact:
		return "Contact"
	case domain.EventLead, domain.EventLeadSubmitted:
		return "Lead"
	case domain.EventPurchase:
		return "Purchase"
	case "CustomizeProduct", "FindLocation", "Schedule", "Search", "StartTrial", "SubmitApplication", "Subscribe", "Download":
		return name
	case "Donate",
		domain.EventInit, domain.EventUpdateUserData,
		domain.EventScroll25, domain.EventScroll50, domain.EventScroll75, domain.EventScroll90,
		domain.EventTimer1Min, domain.EventPlayVideo,
		domain.EventViewVideo25, domain.EventViewVideo50, domain.EventViewVideo75, domain.EventViewVideo90:
		return ""
	}
	slog.Debug("tiktok: evento sem mapeamento conhecido", "event", name)
	return ""
}

func setHashed(m map[string]any, key, val string) {
	if h := util.SHA256Hex(val); h != "" {
		m[key] = h
	}
}

func setIfNotEmpty(m map[string]any, key, val string) {
	if v := strings.TrimSpace(val); v != "" {
		m[key] = v
	}
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func firstNonEmpty(opts ...string) string {
	for _, o := range opts {
		if strings.TrimSpace(o) != "" {
			return o
		}
	}
	return ""
}
