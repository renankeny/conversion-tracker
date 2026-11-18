package handlers

import (
	"context"
	"strings"

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

func (h *AdminCRM) resolveCRMDispatchPixel(ctx context.Context, accountID int64, tu *domain.TrackedUser) (*domain.Pixel, string) {
	pixelName := tu.PixelName
	var resolvedPixel *domain.Pixel
	if h.pixels == nil {
		return nil, pixelName
	}
	if pixelName != "" && pixelName != "api_import" {
		if px, err := RequireAccountPixel(ctx, h.pixels, accountID, pixelName); err == nil {
			resolvedPixel = px
		} else {
			pixelName = ""
		}
	}
	if (pixelName == "" || pixelName == "api_import") && h.whatsapp != nil && tu.ID > 0 {
		if pxName := h.whatsapp.FindPixelNameByTrackedUserID(ctx, accountID, tu.ID); pxName != "" {
			if px, err := RequireAccountPixel(ctx, h.pixels, accountID, pxName); err == nil {
				resolvedPixel = px
				pixelName = pxName
			}
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
	return resolvedPixel, pixelName
}

// resolveCRMPageID respeita estritamente o MetaAccountID explícito do pixel (sem vazar conta de outra operação).
func (h *AdminCRM) resolveCRMPageID(ctx context.Context, accountID int64, tu *domain.TrackedUser, pixel *domain.Pixel) string {
	pageID := ""
	if tu.CtwaClid != "" && h.whatsapp != nil {
		pageID = h.whatsapp.FindPageIDByCtwaClid(ctx, accountID, tu.CtwaClid)
	}
	if pageID == "" && h.whatsapp != nil && tu.ID > 0 {
		pageID = h.whatsapp.FindPageIDByTrackedUserID(ctx, accountID, tu.ID)
	}
	if pageID == "" && h.metaAcc != nil && pixel != nil && pixel.MetaAccountID != nil && *pixel.MetaAccountID > 0 {
		if mAcc, merr := h.metaAcc.FindByID(ctx, *pixel.MetaAccountID); merr == nil && mAcc != nil && mAcc.AccountID == accountID && mAcc.Enabled && mAcc.IsConnected() {
			pageID = mAcc.DefaultPageID
		}
	}
	return pageID
}
