package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// unwrapOnixPayPayloads aceita objeto JSON direto na raiz ({...}) ou lista ([{...}] / envelope de exportação [{"body": {...}}]).
func unwrapOnixPayPayloads(raw []byte) ([][]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	if trimmed[0] == '{' {
		return [][]byte{unwrapOnixPayEnvelopeObject(trimmed)}, nil
	}
	if trimmed[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(trimmed, &arr); err != nil {
			return nil, err
		}
		out := make([][]byte, 0, len(arr))
		for _, elem := range arr {
			et := bytes.TrimSpace(elem)
			if len(et) > 0 && et[0] == '{' {
				out = append(out, unwrapOnixPayEnvelopeObject(et))
			}
		}
		return out, nil
	}
	return nil, io.ErrUnexpectedEOF
}

func unwrapOnixPayEnvelopeObject(objBytes []byte) []byte {
	var probe struct {
		Event string          `json:"event"`
		Body  json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(objBytes, &probe); err == nil && strings.TrimSpace(probe.Event) == "" {
		bt := bytes.TrimSpace(probe.Body)
		if len(bt) > 0 && bt[0] == '{' {
			return bt
		}
	}
	return objBytes
}

func mergeOnixPayTrackProps(root, nested map[string]any) map[string]any {
	if len(root) == 0 && len(nested) == 0 {
		return nil
	}
	merged := make(map[string]any, len(nested)+len(root))
	for k, v := range nested {
		merged[k] = v
	}
	for k, v := range root {
		if v != nil && anyString(v) != "" {
			merged[k] = v
		} else if _, exists := merged[k]; !exists {
			merged[k] = v
		}
	}
	return merged
}

func mapBool(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch b := v.(type) {
		case bool:
			return b
		case string:
			s := strings.ToLower(strings.TrimSpace(b))
			if s == "true" || s == "1" {
				return true
			}
		}
	}
	return false
}

func mapStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			if s := strings.TrimSpace(anyString(v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func parseCheckoutURLParams(rawURL string) url.Values {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	return u.Query()
}

func buildOnixPayPurchaseInput(p onixPayPayload, eventName string, accID int64, pixelQuery string, itemRaw []byte) purchaseInput {
	first, last := util.SplitFullName(p.Client.Name)
	items := p.OrderItems
	if len(items) == 0 {
		items = p.Transaction.OrderItems
	}
	tp := mergeOnixPayTrackProps(p.TrackProps, p.Transaction.TrackProps)
	urlQuery := parseCheckoutURLParams(p.CheckoutURL)

	value := util.ParseMoneyAny(p.Transaction.Amount)
	if value == 0 {
		for _, it := range items {
			value += util.ParseMoneyAny(it.Price)
		}
	}
	if value == 0 {
		value = util.ParseMoneyAny(p.Transaction.OriginalAmount)
	}

	var prodID, prodName string
	if len(items) > 0 {
		prodID = util.FirstNonEmpty(anyString(items[0].Product.ID), items[0].Product.ExternalID, anyString(items[0].ID))
		prodName = strings.TrimSpace(items[0].Product.Name)
	}
	prodID = util.FirstNonEmpty(prodID, strings.TrimSpace(p.OfferCode))

	var addrCountry, addrState, addrCity, addrZip string
	if p.Client.Address != nil {
		addrCountry = p.Client.Address.Country
		addrState = p.Client.Address.State
		addrCity = p.Client.Address.City
		addrZip = p.Client.Address.ZipCode
	}

	directExtID := mapStr(tp, "external_id", "externalId", "user_id", "userId")
	if extracted := util.ExtractUserIDFromUTM(directExtID); extracted != "" {
		directExtID = extracted
	}

	externalID := util.FirstNonEmpty(
		directExtID,
		util.ExtractUserIDFromMap(tp),
		util.ExtractUserIDFromAny(
			urlQuery.Get("utm_source"),
			urlQuery.Get("sck"),
			urlQuery.Get("src"),
			urlQuery.Get("utm_content"),
			urlQuery.Get("utm_term"),
			urlQuery.Get("utm_campaign"),
		),
		util.ExtractUserIDDeep(itemRaw),
	)

	isUpsell := mapBool(tp, "isUpsell", "is_upsell")
	isOrderBump := mapBool(tp, "isOrderBump", "is_order_bump")

	return purchaseInput{
		GatewayName:       "onixpay",
		EventName:         eventName,
		ExternalID:        externalID,
		OrderID:           util.FirstNonEmpty(p.OrderID, p.Transaction.ID, p.Transaction.Identifier),
		Value:             value,
		Currency:          util.FirstNonEmpty(p.Transaction.Currency, p.Transaction.OriginalCurrency, "BRL"),
		FirstName:         first,
		LastName:          last,
		Email:             p.Client.Email,
		Phone:             p.Client.Phone,
		FBC:               util.FirstNonEmpty(mapStr(tp, "fbc"), strings.TrimSpace(urlQuery.Get("fbc"))),
		FBP:               util.FirstNonEmpty(mapStr(tp, "fbp"), strings.TrimSpace(urlQuery.Get("fbp"))),
		ProductID:         prodID,
		ProductName:       prodName,
		IsUpsell:          isUpsell,
		IsOrderBump:       isOrderBump,
		ClientIP:          mapStr(tp, "ip", "client_ip"),
		ClientUserAgent:   mapStr(tp, "user_agent", "userAgent"),
		Country:           util.FirstNonEmpty(addrCountry, mapStr(tp, "country")),
		State:             util.FirstNonEmpty(addrState, mapStr(tp, "state")),
		City:              util.FirstNonEmpty(addrCity, mapStr(tp, "city")),
		PostalCode:        util.FirstNonEmpty(addrZip, mapStr(tp, "zip_code", "zipCode", "postal_code")),
		SourceURL:         util.FirstNonEmpty(p.CheckoutURL, p.OrderURL),
		UTMSource:         util.FirstNonEmpty(mapStr(tp, "utm_source"), urlQuery.Get("utm_source")),
		UTMMedium:         util.FirstNonEmpty(mapStr(tp, "utm_medium"), urlQuery.Get("utm_medium")),
		UTMCampaign:       util.FirstNonEmpty(mapStr(tp, "utm_campaign"), urlQuery.Get("utm_campaign")),
		UTMContent:        util.FirstNonEmpty(mapStr(tp, "utm_content"), urlQuery.Get("utm_content")),
		UTMTerm:           util.FirstNonEmpty(mapStr(tp, "utm_term"), urlQuery.Get("utm_term")),
		GCLID:             util.FirstNonEmpty(mapStr(tp, "gclid"), urlQuery.Get("gclid")),
		TTClid:            util.FirstNonEmpty(mapStr(tp, "ttclid"), urlQuery.Get("ttclid")),
		AccountID:         accID,
		PixelQuery:        pixelQuery,
		AlsoSearchByEmail: true,
		RawPayload:        itemRaw,
	}
}
