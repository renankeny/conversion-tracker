package queue

import (
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func buildAggregatedItem(in domain.SendEventInput, val float64) domain.AggregatedItem {
	itemTitle := in.ContentName
	if itemTitle == "" {
		itemTitle = in.PixelName
	}
	itemType := in.ItemType
	if itemType == "" {
		itemType = "main"
	}
	return domain.AggregatedItem{
		OrderID:      in.OrderID,
		ProductTitle: itemTitle,
		ItemType:     itemType,
		Value:        val,
		Gateway:      util.FirstNonEmpty(in.Gateway, in.ActionSource),
		CreatedAt:    time.Now().Unix(),
	}
}

func newAggregatedSession(pixelName string, in domain.SendEventInput, val float64, currentItem domain.AggregatedItem) AggregatedSession {
	session := AggregatedSession{
		PixelName:             pixelName,
		EventName:             in.EventName,
		FirstEventID:          in.EventID,
		FirstEventTime:        in.EventTime,
		SourceURL:             in.SourceURL,
		TotalValue:            val,
		Currency:              util.FirstNonEmpty(in.Currency, "BRL"),
		Count:                 1,
		OrderIDs:              make([]string, 0, 4),
		ContentIDs:            make([]string, 0, 4),
		ContentNames:          make([]string, 0, 4),
		Items:                 []domain.AggregatedItem{currentItem},
		ExternalID:            in.ExternalID,
		IDWpp:                 in.IDWpp,
		ClientIP:              in.ClientIP,
		ClientUserAgent:       in.ClientUserAgent,
		FBP:                   in.FBP,
		FBC:                   in.FBC,
		Email:                 in.Email,
		Phone:                 in.Phone,
		FirstName:             in.FirstName,
		LastName:              in.LastName,
		Country:               in.Country,
		State:                 in.State,
		City:                  in.City,
		PostalCode:            in.PostalCode,
		TTClid:                in.TTClid,
		TTP:                   in.TTP,
		GCLID:                 in.GCLID,
		WBraid:                in.WBraid,
		GBraid:                in.GBraid,
		GA4ClientID:           in.GA4ClientID,
		GA4SessionID:          in.GA4SessionID,
		OverrideGadsAccountID: in.OverrideGadsAccountID,
		OverrideGadsActionID:  in.OverrideGadsActionID,
	}
	if in.OrderID != "" {
		session.OrderIDs = append(session.OrderIDs, in.OrderID)
	}
	if in.ContentID != "" {
		session.ContentIDs = append(session.ContentIDs, in.ContentID)
	}
	if in.ContentName != "" {
		session.ContentNames = append(session.ContentNames, in.ContentName)
	}
	return session
}

func mergeSessionPII(session *AggregatedSession, in domain.SendEventInput) {
	session.ExternalID = util.FirstNonEmpty(session.ExternalID, in.ExternalID)
	session.SourceURL = util.FirstNonEmpty(session.SourceURL, in.SourceURL)
	session.Email = util.FirstNonEmpty(in.Email, session.Email)
	session.Phone = util.FirstNonEmpty(in.Phone, session.Phone)
	session.FirstName = util.FirstNonEmpty(in.FirstName, session.FirstName)
	session.LastName = util.FirstNonEmpty(in.LastName, session.LastName)
	session.Country = util.FirstNonEmpty(in.Country, session.Country)
	session.State = util.FirstNonEmpty(in.State, session.State)
	session.City = util.FirstNonEmpty(in.City, session.City)
	session.PostalCode = util.FirstNonEmpty(in.PostalCode, session.PostalCode)
	session.FBP = util.FirstNonEmpty(in.FBP, session.FBP)
	session.FBC = util.FirstNonEmpty(in.FBC, session.FBC)
	session.GCLID = util.FirstNonEmpty(in.GCLID, session.GCLID)
	session.WBraid = util.FirstNonEmpty(in.WBraid, session.WBraid)
	session.GBraid = util.FirstNonEmpty(in.GBraid, session.GBraid)
	session.TTClid = util.FirstNonEmpty(in.TTClid, session.TTClid)
	session.TTP = util.FirstNonEmpty(in.TTP, session.TTP)
	session.GA4ClientID = util.FirstNonEmpty(in.GA4ClientID, session.GA4ClientID)
	session.GA4SessionID = util.FirstNonEmpty(in.GA4SessionID, session.GA4SessionID)
	session.ClientIP = util.FirstNonEmpty(in.ClientIP, session.ClientIP)
	session.ClientUserAgent = util.FirstNonEmpty(in.ClientUserAgent, session.ClientUserAgent)
}

func appendSubsequentSessionItem(session *AggregatedSession, in domain.SendEventInput, val float64, currentItem domain.AggregatedItem) {
	if currentItem.ItemType == "" || currentItem.ItemType == "main" {
		if in.ItemType != "" && in.ItemType != "main" {
			currentItem.ItemType = in.ItemType
		} else {
			currentItem.ItemType = "upsell"
		}
	}

	session.TotalValue += val
	session.Count++
	session.Items = append(session.Items, currentItem)

	if in.OrderID != "" && !sliceContains(session.OrderIDs, in.OrderID) {
		session.OrderIDs = append(session.OrderIDs, in.OrderID)
	}
	if in.ContentID != "" && !sliceContains(session.ContentIDs, in.ContentID) {
		session.ContentIDs = append(session.ContentIDs, in.ContentID)
	}
	if in.ContentName != "" && !sliceContains(session.ContentNames, in.ContentName) {
		session.ContentNames = append(session.ContentNames, in.ContentName)
	}

	mergeSessionPII(session, in)
}

func sliceContains(s []string, val string) bool {
	for _, item := range s {
		if item == val {
			return true
		}
	}
	return false
}

