package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func (h *AdminCRM) resolveOrCreateCRMTrackedUser(
	ctx context.Context,
	accountID int64,
	firstName, lastName, email, phone, externalID, pixelName string,
) *domain.TrackedUser {
	var trackedUser *domain.TrackedUser
	if email != "" {
		trackedUser, _ = h.users.FindByEmailByAccount(ctx, email, accountID)
	}
	if trackedUser == nil && phone != "" {
		trackedUser, _ = h.users.FindByPhoneVariantsByAccount(ctx, util.PhoneVariants(phone), accountID)
	}
	if trackedUser == nil && strings.TrimSpace(externalID) != "" {
		trackedUser, _ = h.users.FindByExternalIDByAccount(ctx, strings.TrimSpace(externalID), accountID)
	}

	if trackedUser == nil {
		extID := strings.TrimSpace(externalID)
		if extID == "" {
			extID = "api_" + util.GenerateID10()
		}
		pxName := strings.TrimSpace(pixelName)
		if pxName == "" {
			pxName = "api_import"
		}
		tu := &domain.TrackedUser{
			ExternalID: extID,
			PixelName:  pxName,
			AccountID:  accountID,
			FirstName:  firstName,
			LastName:   lastName,
			Email:      email,
			Phone:      phone,
		}
		if err := h.users.Create(ctx, tu); err != nil {
			return nil
		}
		return tu
	}

	if firstName != "" {
		trackedUser.FirstName = firstName
	}
	if lastName != "" {
		trackedUser.LastName = lastName
	}
	if email != "" && trackedUser.Email == "" {
		trackedUser.Email = email
	}
	if phone != "" && trackedUser.Phone == "" {
		trackedUser.Phone = phone
	}
	if pixelName != "" && pixelName != "api_import" && (trackedUser.PixelName == "" || trackedUser.PixelName == "api_import") {
		trackedUser.PixelName = pixelName
	}
	_ = h.users.Upsert(ctx, trackedUser)
	return trackedUser
}

// dispatchCRMStageTrigger enfileira o evento de conversão da etapa do CRM enriquecendo Pixel padrão,
// chaves geográficas, URL de origem e metadados CTWA/Meta OAuth.
func (h *AdminCRM) dispatchCRMStageTrigger(
	ctx context.Context,
	accountID int64,
	stage *domain.CRMStage,
	tu *domain.TrackedUser,
	eventVal float64,
) {
	if h.queue == nil || stage == nil || stage.TriggerEvent == "" || tu == nil {
		return
	}
	if stage.TriggerEvent == domain.EventPurchase && eventVal <= 0 {
		return
	}

	pixelName := tu.PixelName
	var resolvedPixel *domain.Pixel
	if h.pixels != nil {
		if pixelName != "" && pixelName != "api_import" {
			if px, err := RequireAccountPixel(ctx, h.pixels, accountID, pixelName); err == nil {
				resolvedPixel = px
			} else {
				pixelName = ""
			}
		}
		if pixelName == "" || pixelName == "api_import" {
			if list, err := h.pixels.ListByAccount(ctx, accountID); err == nil {
				for i := range list {
					if list[i].Enabled {
						resolvedPixel = &list[i]
						pixelName = list[i].Name
						break
					}
				}
			}
		}
	}
	if pixelName == "" || pixelName == "api_import" {
		return
	}

	var valuePtr *float64
	if eventVal > 0 {
		v := eventVal
		valuePtr = &v
	}

	linked, err := h.users.LinkedFieldsByAccount(ctx, accountID, tu.ExternalID)
	if err != nil {
		linked = domain.LinkedTrackingFields{
			FBP: tu.FBP, FBC: tu.FBC, GCLID: tu.GCLID, WBraid: tu.WBraid, GBraid: tu.GBraid,
			TTClid: tu.TTClid, TTP: tu.TTP, GA4ClientID: tu.GA4ClientID, GA4SessionID: tu.GA4SessionID,
		}
	}

	actionSource := "website"
	msgChannel := ""
	pageID := ""
	sourceURL := util.FirstNonEmpty(tu.URL, tu.FirstURL)
	if tu.CtwaClid != "" {
		actionSource = "business_messaging"
		msgChannel = "whatsapp"
		sourceURL = ""
		if h.whatsapp != nil {
			pageID = h.whatsapp.FindPageIDByCtwaClid(ctx, accountID, tu.CtwaClid)
		}
		if pageID == "" && h.metaAcc != nil {
			if resolvedPixel != nil && resolvedPixel.MetaAccountID != nil && *resolvedPixel.MetaAccountID > 0 {
				if mAcc, merr := h.metaAcc.FindByID(ctx, *resolvedPixel.MetaAccountID); merr == nil && mAcc != nil {
					pageID = mAcc.DefaultPageID
				}
			}
			if pageID == "" {
				if mAcc, merr := h.metaAcc.FindFirstConnectedByAccount(ctx, accountID); merr == nil && mAcc != nil {
					pageID = mAcc.DefaultPageID
				}
			}
		}
	}

	currency := "BRL"
	if h.whatsapp != nil {
		var pxID int64
		if resolvedPixel != nil {
			pxID = resolvedPixel.ID
		}
		currency = h.whatsapp.ResolveCurrencyForLead(ctx, accountID, tu.ID, pxID)
	}

	in := domain.SendEventInput{
		PixelName:             pixelName,
		EventName:             stage.TriggerEvent,
		EventID:               util.GenerateID10(),
		EventTime:             time.Now().Unix(),
		SourceURL:             sourceURL,
		ActionSource:          actionSource,
		MessagingChannel:      msgChannel,
		CtwaClid:              tu.CtwaClid,
		PageID:                pageID,
		ExternalID:            tu.ExternalID,
		ClientIP:              tu.ClientIP,
		ClientUserAgent:       tu.ClientUserAgent,
		FBP:                   linked.FBP,
		FBC:                   linked.FBC,
		GCLID:                 linked.GCLID,
		WBraid:                linked.WBraid,
		GBraid:                linked.GBraid,
		TTClid:                linked.TTClid,
		TTP:                   linked.TTP,
		GA4ClientID:           linked.GA4ClientID,
		GA4SessionID:          linked.GA4SessionID,
		Email:                 tu.Email,
		Phone:                 tu.Phone,
		FirstName:             tu.FirstName,
		LastName:              tu.LastName,
		IDWpp:                 tu.IDWpp,
		Country:               tu.Country,
		State:                 tu.State,
		City:                  tu.City,
		PostalCode:            tu.PostalCode,
		Value:                 valuePtr,
		Currency:              currency,
		OverrideGadsAccountID: stage.GoogleAdsAccountID,
		OverrideGadsActionID:  stage.GoogleAdsActionID,
	}

	_ = h.queue.EnqueueSendEvent(ctx, in)
}
