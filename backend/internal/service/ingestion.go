package service

import (
	"context"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// EnrichSendInput preenche campos vazios de PII, GeoIP e identificadores de clique/cookie
// (incluindo unificação cross-device pelo group_id restrito ao accountID).
func EnrichSendInput(ctx context.Context, users *repo.UserRepo, accountID int64, in *domain.SendEventInput) {
	if users == nil || in == nil || in.ExternalID == "" {
		return
	}

	if u, err := users.FindByExternalIDByAccount(ctx, in.ExternalID, accountID); err == nil && u != nil {
		if accountID == 0 {
			accountID = u.AccountID
		}
		fillStr(&in.Email, u.Email)
		fillStr(&in.Phone, u.Phone)
		fillStr(&in.FirstName, u.FirstName)
		fillStr(&in.LastName, u.LastName)
		fillStr(&in.IDWpp, u.IDWpp)
		fillStr(&in.ClientIP, u.ClientIP)
		fillStr(&in.ClientUserAgent, u.ClientUserAgent)
		fillStr(&in.Country, u.Country)
		fillStr(&in.State, u.State)
		fillStr(&in.City, u.City)
		fillStr(&in.PostalCode, u.PostalCode)
		fillStr(&in.SourceURL, u.URL)
		fillStr(&in.FBP, u.FBP)
		fillStr(&in.FBC, u.FBC)
		fillStr(&in.GCLID, u.GCLID)
		fillStr(&in.WBraid, u.WBraid)
		fillStr(&in.GBraid, u.GBraid)
		fillStr(&in.TTClid, u.TTClid)
		fillStr(&in.TTP, u.TTP)
		fillStr(&in.GA4ClientID, u.GA4ClientID)
		fillStr(&in.GA4SessionID, u.GA4SessionID)
	}

	if linked, err := users.LinkedFieldsByAccount(ctx, accountID, in.ExternalID); err == nil {
		fillStr(&in.FBP, linked.FBP)
		fillStr(&in.FBC, linked.FBC)
		fillStr(&in.GCLID, linked.GCLID)
		fillStr(&in.WBraid, linked.WBraid)
		fillStr(&in.GBraid, linked.GBraid)
		fillStr(&in.TTClid, linked.TTClid)
		fillStr(&in.TTP, linked.TTP)
		fillStr(&in.GA4ClientID, linked.GA4ClientID)
		fillStr(&in.GA4SessionID, linked.GA4SessionID)
	}
}

func fillStr(dst *string, val string) {
	if *dst == "" && val != "" {
		*dst = val
	}
}

