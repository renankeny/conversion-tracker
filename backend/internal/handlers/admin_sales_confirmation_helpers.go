package handlers

import (
	"context"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func (h *AdminSalesConfirmation) resolveConfirmationAttribution(
	ctx context.Context,
	effAccountID int64,
	pixel *domain.Pixel,
	user *domain.TrackedUser,
) (sourceURL, actionSource, msgChannel, ctwaClid, pageID string) {
	sourceURL = util.FirstNonEmpty(user.URL, user.FirstURL)
	actionSource = "website"
	if user.CtwaClid != "" && user.FBC == "" && h.whatsappRepo != nil {
		pageID = h.whatsappRepo.FindPageIDByCtwaClid(ctx, effAccountID, user.CtwaClid)
		if pageID == "" && user.ID > 0 {
			pageID = h.whatsappRepo.FindPageIDByTrackedUserID(ctx, effAccountID, user.ID)
		}
		if pageID == "" && h.metaAccRepo != nil && pixel != nil && pixel.MetaAccountID != nil && *pixel.MetaAccountID > 0 {
			if mAcc, merr := h.metaAccRepo.FindByID(ctx, *pixel.MetaAccountID); merr == nil && mAcc != nil && mAcc.AccountID == effAccountID && mAcc.Enabled && mAcc.IsConnected() {
				pageID = mAcc.DefaultPageID
			}
		}
		if pageID != "" {
			actionSource = "business_messaging"
			msgChannel = "whatsapp"
			ctwaClid = user.CtwaClid
			sourceURL = ""
		}
	}
	if actionSource == "website" && sourceURL == "" {
		actionSource = "system_generated"
	}
	return sourceURL, actionSource, msgChannel, ctwaClid, pageID
}
