package googleads

import (
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func resolveTargetOperatingAndAction(account *domain.GoogleAdsAccount, action *domain.GoogleAdsConversionAction) (string, string) {
	rawAction := strings.TrimSpace(action.ConversionActionID)
	ownerID := digitsOnlyStr(action.OwnerCustomerID)
	actionID := rawAction
	if strings.Contains(rawAction, "/") {
		parts := strings.Split(rawAction, "/")
		actionID = parts[len(parts)-1]
		if ownerID == "" && len(parts) >= 2 && parts[0] == "customers" {
			ownerID = digitsOnlyStr(parts[1])
		}
	}
	if cleanAct := digitsOnlyStr(actionID); cleanAct != "" {
		actionID = cleanAct
	}
	if ownerID == "" {
		ownerID = digitsOnlyStr(account.CustomerID)
	}
	return ownerID, actionID
}

func buildDataManagerPayload(operatingID, loginID, actionID string, in domain.SendEventInput, userData []map[string]any, validateOnly bool) map[string]any {
	destination := map[string]any{
		"reference":            "dest_0",
		"operatingAccount":     map[string]any{"accountType": "GOOGLE_ADS", "accountId": operatingID},
		"productDestinationId": actionID,
	}
	if loginID != "" && loginID != operatingID {
		destination["loginAccount"] = map[string]any{"accountType": "GOOGLE_ADS", "accountId": loginID}
	}

	t := time.Unix(in.EventTime, 0)
	if in.EventTime <= 0 {
		t = time.Now()
	}
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil || loc == nil {
		loc = time.FixedZone("BRT", -3*3600)
	}
	eventTimestamp := t.In(loc).Format(time.RFC3339)

	adIdentifiers := map[string]any{}
	switch {
	case in.GCLID != "":
		adIdentifiers["gclid"] = in.GCLID
	case in.GBraid != "":
		adIdentifiers["gbraid"] = in.GBraid
	case in.WBraid != "":
		adIdentifiers["wbraid"] = in.WBraid
	}

	event := map[string]any{
		"destinationReferences": []string{"dest_0"},
		"eventTimestamp":        eventTimestamp,
		"eventSource":           "WEB",
	}
	if len(adIdentifiers) > 0 {
		event["adIdentifiers"] = adIdentifiers
	}
	if in.Value != nil {
		event["conversionValue"] = *in.Value
		event["currency"] = orDefault(in.Currency, "BRL")
	}
	if txID := orDefault(in.OrderID, in.EventID); txID != "" {
		event["transactionId"] = txID
	}
	if len(userData) > 0 {
		event["userData"] = map[string]any{"userIdentifiers": userData}
	}

	return map[string]any{
		"destinations": []map[string]any{destination},
		"encoding":     "HEX",
		"events":       []map[string]any{event},
		"validateOnly": validateOnly,
	}
}

func buildUserData(in domain.SendEventInput) []map[string]any {
	out := make([]map[string]any, 0, 4)
	if h := util.SHA256Hex(util.NormalizeEmail(in.Email)); h != "" {
		out = append(out, map[string]any{"emailAddress": h})
	}
	if h := util.SHA256Hex(util.NormalizePhoneE164(in.Phone)); h != "" {
		out = append(out, map[string]any{"phoneNumber": h})
	}
	first := util.NormalizeName(in.FirstName)
	last := util.NormalizeName(in.LastName)
	zip := util.NormalizeZip(in.PostalCode)
	if first != "" && last != "" && zip != "" {
		country := strings.ToUpper(util.NormalizeCountry(in.Country))
		if len(country) != 2 {
			country = "BR"
		}
		addr := map[string]any{
			"givenName":  util.SHA256Hex(first),
			"familyName": util.SHA256Hex(last),
			"postalCode": zip,
			"regionCode": country,
		}
		out = append(out, map[string]any{"address": addr})
	}
	return out
}

func digitsOnlyStr(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
