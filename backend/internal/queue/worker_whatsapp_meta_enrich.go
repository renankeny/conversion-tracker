package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// resolvePixelMetaOAuth respeita estritamente o vínculo explícito pixel.MetaAccountID.
// Quando pixel.MetaAccountID == nil ("Nenhuma — usar apenas Access Token manual"), NUNCA faz fallback
// para contas OAuth de outras operações/clientes do mesmo workspace.
func resolvePixelMetaOAuth(ctx context.Context, deps Deps, pixel *domain.Pixel) (*domain.MetaAccount, bool) {
	if deps.MetaAcc == nil || deps.MetaOAuth == nil || pixel == nil {
		return nil, false
	}
	if pixel.MetaAccountID == nil || *pixel.MetaAccountID <= 0 {
		return nil, false
	}
	acc, err := deps.MetaAcc.FindByID(ctx, *pixel.MetaAccountID)
	if err != nil || acc == nil || acc.AccountID != pixel.AccountID {
		return nil, false
	}
	if !acc.Enabled {
		return nil, true
	}
	if acc.IsConnected() {
		return acc, false
	}
	return nil, false
}

func dispatchMetaWithOAuthFallback(ctx context.Context, deps Deps, pixel *domain.Pixel, in domain.SendEventInput, skip func(string) bool) ([]meta.SendResult, bool) {
	oauthAcc, oauthPaused := resolvePixelMetaOAuth(ctx, deps, pixel)
	if oauthAcc != nil && in.PageID == "" && oauthAcc.DefaultPageID != "" && (in.ActionSource == "business_messaging" || in.CtwaClid != "") {
		in.PageID = oauthAcc.DefaultPageID
	}
	hasOAuthLink := pixel.MetaAccountID != nil && *pixel.MetaAccountID > 0
	results := make([]meta.SendResult, 0, len(pixel.PixelIDs))
	skippedPaused := false

	for i, pid := range pixel.PixelIDs {
		if skip != nil && skip(pid) {
			continue
		}
		tok := ""
		if i < len(pixel.AccessTokens) {
			tok = util.CleanTokenInput(pixel.AccessTokens[i])
		}
		hasManual := util.IsPlausibleMetaToken(tok)

		// Se o pixel possui conta OAuth conectada, prioriza OAuth com fallback para token manual se houver.
		if oauthAcc != nil {
			rawBody, err := meta.BuildCAPIPayload(in, "", deps.Meta.PartnerAgent())
			if err != nil {
				results = append(results, meta.SendResult{PixelID: pid, Err: err})
				continue
			}
			code, respBody, perr := deps.MetaOAuth.SendCAPIEvent(ctx, oauthAcc, pid, rawBody)
			res := meta.SendResult{PixelID: pid, StatusCode: code, Body: respBody, RequestBody: string(rawBody), Err: perr}
			if perr == nil && code >= 300 {
				res.Err = fmt.Errorf("meta oauth capi returned %d: %s", code, respBody)
			}
			if res.Err != nil && hasManual && util.IsMetaAuthFailure(res.StatusCode, res.Body) {
				slog.Warn("meta oauth rejected, falling back to manual token", "pixel_name", pixel.Name, "pixel_id", pid)
				sub := *pixel
				sub.PixelIDs = []string{pid}
				sub.AccessTokens = []string{tok}
				sub.TestCodes = nil
				results = append(results, deps.Meta.Send(ctx, &sub, in)...)
				continue
			}
			results = append(results, res)
			continue
		}

		// Sem conta OAuth ativa: usa token manual se existir.
		if tok != "" && (!hasOAuthLink || hasManual) {
			sub := *pixel
			sub.PixelIDs = []string{pid}
			sub.AccessTokens = []string{tok}
			sub.TestCodes = nil
			results = append(results, deps.Meta.Send(ctx, &sub, in)...)
			continue
		}

		if oauthPaused {
			skippedPaused = true
			continue
		}

		rawBody, _ := meta.BuildCAPIPayload(in, "", deps.Meta.PartnerAgent())
		errMsg := "missing access token for pixel " + pid
		if hasOAuthLink {
			errMsg = "conta Meta Ads OAuth desconectada; reconecte em Integrações > Meta Ads"
		}
		results = append(results, meta.SendResult{
			PixelID:     pid,
			StatusCode:  401,
			RequestBody: string(rawBody),
			Err:         errors.New(errMsg),
		})
	}
	return results, skippedPaused && len(results) == 0
}

func enrichCTWAAdMetadata(
	ctx context.Context,
	deps Deps,
	inst *domain.WhatsAppInstance,
	u *domain.TrackedUser,
	chatID int64,
	sourceID string,
) (*domain.Pixel, string) {
	pixel, err := deps.Pixels.FindByID(ctx, inst.PixelID)
	if err != nil || pixel == nil || !pixel.Enabled {
		return nil, ""
	}

	oauthAcc, _ := resolvePixelMetaOAuth(ctx, deps, pixel)
	var metaInfo cachedAdMetadata
	if sourceID != "" {
		metaInfo = fetchAdMetadataCached(ctx, deps, inst.AccountID, sourceID, oauthAcc, pixel.MetaPermanentToken)
		if metaInfo.PageID == "" && oauthAcc != nil && oauthAcc.DefaultPageID != "" {
			metaInfo.PageID = oauthAcc.DefaultPageID
		}
		if metaInfo.CampaignName != "" || metaInfo.AdsetName != "" || metaInfo.AdName != "" || metaInfo.PageID != "" {
			_ = deps.WhatsApp.UpdateChatAdEnrichment(
				ctx, chatID,
				metaInfo.CampaignID, metaInfo.CampaignName,
				metaInfo.AdsetID, metaInfo.AdsetName,
				metaInfo.AdName, metaInfo.PageID, metaInfo.ThumbnailURL,
			)
			u.LastUTMCampaign = util.FirstNonEmpty(metaInfo.CampaignName, u.LastUTMCampaign)
			u.LastUTMMedium = util.FirstNonEmpty(metaInfo.AdsetName, u.LastUTMMedium)
			u.LastUTMContent = util.FirstNonEmpty(metaInfo.AdName, u.LastUTMContent)
			if u.FirstUTMCampaign == "" {
				u.FirstUTMCampaign = metaInfo.CampaignName
				u.FirstUTMMedium = metaInfo.AdsetName
				u.FirstUTMContent = metaInfo.AdName
			}
			_ = deps.Users.Upsert(ctx, u)
		}
	} else if oauthAcc != nil && oauthAcc.DefaultPageID != "" {
		metaInfo.PageID = oauthAcc.DefaultPageID
	}

	return pixel, metaInfo.PageID
}
