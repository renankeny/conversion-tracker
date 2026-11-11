package queue

import (
	"context"
	"log/slog"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func resolveOrCreatePortalLead(
	ctx context.Context,
	deps Deps,
	inst *domain.WhatsAppInstance,
	parsed *ParsedWhatsAppMessage,
	protocolCode, phone, lidDigits string,
) (*domain.TrackedUser, string) {
	first, last := splitFullName(parsed.PushName)

	// Portal 1: Protocolo injetado pelo tracker.js (Site / LP / Link na Bio)
	if protocolCode != "" {
		u, _ := deps.Users.FindByProtocolByAccount(ctx, protocolCode, inst.AccountID)
		if u == nil && phone != "" {
			u, _ = deps.Users.FindByPhoneVariantsByAccount(ctx, util.PhoneVariants(phone), inst.AccountID)
		}
		if u == nil && lidDigits != "" {
			u, _ = deps.Users.FindByWaLIDByAccount(ctx, lidDigits, inst.AccountID)
		}
		if u == nil {
			pixel, err := deps.Pixels.FindByID(ctx, inst.PixelID)
			if err != nil || pixel == nil {
				return nil, ""
			}
			u = &domain.TrackedUser{
				ExternalID: protocolCode,
				IDWpp:      protocolCode,
				AccountID:  inst.AccountID,
				PixelName:  pixel.Name,
			}
		}
		hydrateWhatsAppUserFields(u, phone, lidDigits, first, last, parsed)
		_ = deps.Users.Upsert(ctx, u)
		if phone != "" {
			if stitched, lerr := deps.Users.ResolveAndLink(ctx, inst.AccountID, u.ExternalID, phone, u.Email); lerr == nil && stitched != nil {
				u = stitched
			}
		}
		return u, ClassifyPortalOrigin(parsed.CtwaClid, parsed.SourceID, u)
	}

	// Portal 2: Meta Ads CTWA Direto (ctwaClid ou referral.sourceID)
	if parsed.CtwaClid != "" || parsed.SourceID != "" {
		pixel, err := deps.Pixels.FindByID(ctx, inst.PixelID)
		if err != nil || pixel == nil {
			return nil, ""
		}

		var u *domain.TrackedUser
		if phone != "" {
			u, _ = deps.Users.FindByPhoneVariantsByAccount(ctx, util.PhoneVariants(phone), inst.AccountID)
		}
		if u == nil && lidDigits != "" {
			u, _ = deps.Users.FindByWaLIDByAccount(ctx, lidDigits, inst.AccountID)
		}
		if u == nil {
			extKey := "wpp:" + util.FirstNonEmpty(phone, lidDigits, util.GenerateID10())
			u = &domain.TrackedUser{
				ExternalID:     extKey,
				IDWpp:          util.GenerateID10(),
				AccountID:      inst.AccountID,
				PixelName:      pixel.Name,
				FirstUTMSource: util.FirstNonEmpty(parsed.SourceApp, "facebook"),
			}
		}
		hydrateWhatsAppUserFields(u, phone, lidDigits, first, last, parsed)
		if parsed.SourceID != "" {
			u.WaAdID = parsed.SourceID
		}
		if parsed.SourceApp != "" {
			u.WaSourceApp = parsed.SourceApp
			u.LastUTMSource = parsed.SourceApp
		} else if u.LastUTMSource == "" {
			u.LastUTMSource = "facebook"
		}
		if err := deps.Users.Upsert(ctx, u); err != nil {
			slog.Warn("upsert ctwa tracked user failed", "err", err)
			return nil, ""
		}
		if phone != "" {
			if stitched, lerr := deps.Users.ResolveAndLink(ctx, inst.AccountID, u.ExternalID, phone, u.Email); lerr == nil && stitched != nil {
				u = stitched
			}
		}
		return u, domain.WhatsAppOriginMetaCTWA
	}

	// Portal 3: Visitante já rastreado na conta pelo telefone ou @lid (preencheu form/checkout na LP e chamou sem protocolo)
	var existingUser *domain.TrackedUser
	if phone != "" {
		existingUser, _ = deps.Users.FindByPhoneVariantsByAccount(ctx, util.PhoneVariants(phone), inst.AccountID)
	}
	if existingUser == nil && lidDigits != "" {
		existingUser, _ = deps.Users.FindByWaLIDByAccount(ctx, lidDigits, inst.AccountID)
	}
	if existingUser != nil {
		hydrateWhatsAppUserFields(existingUser, phone, lidDigits, first, last, parsed)
		_ = deps.Users.Upsert(ctx, existingUser)
		return existingUser, ClassifyPortalOrigin(parsed.CtwaClid, parsed.SourceID, existingUser)
	}

	return nil, ""
}

func hydrateWhatsAppUserFields(u *domain.TrackedUser, phone, lidDigits, first, last string, parsed *ParsedWhatsAppMessage) {
	if phone != "" {
		u.Phone = phone
	}
	if lidDigits != "" {
		u.WaLid = lidDigits
	}
	if !parsed.IsFromMe {
		if first != "" && u.FirstName == "" {
			u.FirstName = first
		}
		if last != "" && u.LastName == "" {
			u.LastName = last
		}
	}
	if parsed.CtwaClid != "" {
		u.CtwaClid = parsed.CtwaClid
	}
}

func ensureCRMLeadForWhatsApp(ctx context.Context, deps Deps, inst *domain.WhatsAppInstance, trackedUserID int64) int64 {
	if deps.CRM == nil || trackedUserID <= 0 {
		return 0
	}
	var stageID int64
	if inst.DefaultCRMStageID != nil && *inst.DefaultCRMStageID > 0 {
		if st, err := deps.CRM.GetStageByID(ctx, *inst.DefaultCRMStageID, inst.AccountID); err == nil && st != nil {
			stageID = st.ID
		}
	}
	if stageID == 0 {
		_ = deps.CRM.EnsureDefaultStages(ctx, inst.AccountID)
		if firstStage, err := deps.CRM.GetFirstStage(ctx, inst.AccountID); err == nil && firstStage != nil {
			stageID = firstStage.ID
		}
	}
	if stageID == 0 {
		return 0
	}
	id, err := deps.CRM.EnsureLeadReturningID(ctx, inst.AccountID, trackedUserID, stageID)
	if err != nil {
		slog.Warn("ensure crm lead from whatsapp failed", "err", err)
		return 0
	}
	if deps.Redis != nil && id > 0 {
		wshub.PublishCustom(ctx, deps.Redis, inst.AccountID, "crm_updated", map[string]any{"action": "whatsapp_lead_ensured", "lead_id": id})
	}
	return id
}

func ClassifyPortalOrigin(ctwaClid, sourceID string, u *domain.TrackedUser) string {
	if ctwaClid != "" || sourceID != "" {
		return domain.WhatsAppOriginMetaCTWA
	}
	if u == nil {
		return domain.WhatsAppOriginWebsiteOrganic
	}
	if u.CtwaClid != "" {
		return domain.WhatsAppOriginMetaCTWA
	}
	if u.FBC != "" || isPaidMetaUTM(u.FirstUTMSource, u.FirstUTMMedium) || isPaidMetaUTM(u.LastUTMSource, u.LastUTMMedium) {
		return domain.WhatsAppOriginMetaLP
	}
	if u.GCLID != "" || u.GBraid != "" || u.WBraid != "" {
		return domain.WhatsAppOriginGoogleAds
	}
	if u.TTClid != "" {
		return domain.WhatsAppOriginTikTokAds
	}
	if isBioLinkSession(u) {
		return domain.WhatsAppOriginBioLinkOrganic
	}
	return domain.WhatsAppOriginWebsiteOrganic
}

func isPaidMetaUTM(src, med string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	m := strings.ToLower(strings.TrimSpace(med))
	isMeta := s == "facebook" || s == "fb" || s == "instagram" || s == "ig" || s == "meta" || strings.Contains(s, "facebook") || strings.Contains(s, "instagram")
	if !isMeta {
		return false
	}
	return m == "cpc" || m == "ppc" || m == "paid" || m == "ads" || m == "cpm" || m == "paid_social" || strings.Contains(m, "ad") || strings.Contains(m, "pago") || strings.Contains(m, "trafego")
}

func isBioLinkSession(u *domain.TrackedUser) bool {
	if u == nil {
		return false
	}
	med := strings.ToLower(u.FirstUTMMedium + " " + u.LastUTMMedium + " " + u.FirstUTMSource + " " + u.LastUTMSource)
	if strings.Contains(med, "bio") || strings.Contains(med, "linktree") || strings.Contains(med, "perfil") {
		return true
	}
	urls := strings.ToLower(u.FirstURL + " " + u.URL + " " + u.Referrer)
	for _, token := range []string{"linktr.ee", "beacons.ai", "bio.site", "lnk.bio", "l.instagram.com", "instagram.com", "/bio"} {
		if strings.Contains(urls, token) {
			return true
		}
	}
	return false
}

func splitFullName(pushName string) (string, string) {
	pushName = strings.TrimSpace(pushName)
	if pushName == "" {
		return "", ""
	}
	parts := strings.SplitN(pushName, " ", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
