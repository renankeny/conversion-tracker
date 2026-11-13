package whatsappai

import (
	"context"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type EventEnqueuer interface {
	EnqueueSendEvent(ctx context.Context, in domain.SendEventInput) error
}

func resolveMetaDefaultPageID(ctx context.Context, metaRepo *repo.MetaAccountRepo, pixel *domain.Pixel) string {
	if metaRepo == nil || pixel == nil {
		return ""
	}
	if pixel.MetaAccountID != nil && *pixel.MetaAccountID > 0 {
		if acc, err := metaRepo.FindByID(ctx, *pixel.MetaAccountID); err == nil && acc != nil && acc.Enabled && acc.IsConnected() && acc.DefaultPageID != "" {
			return acc.DefaultPageID
		}
	}
	if acc, err := metaRepo.FindFirstConnectedByAccount(ctx, pixel.AccountID); err == nil && acc != nil && acc.Enabled && acc.IsConnected() {
		return acc.DefaultPageID
	}
	return ""
}

func buildAISendEventInput(
	ctx context.Context,
	usersRepo *repo.UserRepo,
	metaRepo *repo.MetaAccountRepo,
	pixel *domain.Pixel,
	u *domain.TrackedUser,
	cand repo.AIChatCandidate,
	eventName string,
	eventID string,
	value *float64,
) domain.SendEventInput {
	ctwaClid := util.FirstNonEmpty(cand.CtwaClid, u.CtwaClid)
	pageID := cand.PageID
	if ctwaClid != "" && pageID == "" {
		pageID = resolveMetaDefaultPageID(ctx, metaRepo, pixel)
	}

	actionSource := "website"
	msgChannel := ""
	if ctwaClid != "" {
		if pageID != "" {
			actionSource = "business_messaging"
			msgChannel = "whatsapp"
		} else {
			actionSource = "system_generated"
			ctwaClid = ""
		}
	} else if u.URL == "" && u.FirstURL == "" && u.FBP == "" && u.FBC == "" && u.GCLID == "" && u.TTClid == "" {
		actionSource = "system_generated"
	}

	if actionSource == "business_messaging" && eventName == domain.EventLead {
		eventName = domain.EventLeadSubmitted
	}

	var linked domain.LinkedTrackingFields
	if usersRepo != nil {
		linked, _ = usersRepo.LinkedFieldsByAccount(ctx, cand.AccountID, u.ExternalID)
	}

	sourceURL := util.FirstNonEmpty(u.URL, u.FirstURL)
	if actionSource == "business_messaging" {
		sourceURL = ""
	}

	return domain.SendEventInput{
		PixelName:        pixel.Name,
		EventName:        eventName,
		EventID:          eventID,
		EventTime:        time.Now().Unix(),
		SourceURL:        sourceURL,
		Gateway:          "whatsapp_ai",
		ActionSource:     actionSource,
		MessagingChannel: msgChannel,
		CtwaClid:         ctwaClid,
		PageID:           pageID,
		ExternalID:       u.ExternalID,
		IDWpp:            u.IDWpp,
		ClientIP:         u.ClientIP,
		ClientUserAgent:  u.ClientUserAgent,
		Email:            u.Email,
		Phone:            util.FirstNonEmpty(cand.Phone, u.Phone),
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		Country:          u.Country,
		State:            u.State,
		City:             u.City,
		PostalCode:       u.PostalCode,
		FBP:              util.FirstNonEmpty(u.FBP, linked.FBP),
		FBC:              util.FirstNonEmpty(u.FBC, linked.FBC),
		GCLID:            util.FirstNonEmpty(u.GCLID, linked.GCLID),
		WBraid:           util.FirstNonEmpty(u.WBraid, linked.WBraid),
		GBraid:           util.FirstNonEmpty(u.GBraid, linked.GBraid),
		TTClid:           util.FirstNonEmpty(u.TTClid, linked.TTClid),
		TTP:              util.FirstNonEmpty(u.TTP, linked.TTP),
		GA4ClientID:      util.FirstNonEmpty(u.GA4ClientID, linked.GA4ClientID),
		GA4SessionID:     util.FirstNonEmpty(u.GA4SessionID, linked.GA4SessionID),
		Value:            value,
		Currency:         domain.NormalizeWhatsAppCurrency(cand.Currency),
	}
}

