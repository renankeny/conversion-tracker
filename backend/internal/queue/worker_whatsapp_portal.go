package queue

import (
	"context"
	"log/slog"
	"regexp"
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
		if u, err := deps.Users.FindByProtocolByAccount(ctx, protocolCode, inst.AccountID); err == nil && u != nil {
			if phone != "" {
				u.Phone = phone
			}
			if lidDigits != "" {
				u.WaLid = lidDigits
			}
			if first != "" && u.FirstName == "" {
				u.FirstName = first
			}
			if last != "" && u.LastName == "" {
				u.LastName = last
			}
			if parsed.CtwaClid != "" {
				u.CtwaClid = parsed.CtwaClid
			}
			_ = deps.Users.Upsert(ctx, u)
			if phone != "" {
				if stitched, lerr := deps.Users.ResolveAndLink(ctx, inst.AccountID, u.ExternalID, phone, u.Email); lerr == nil && stitched != nil {
					u = stitched
				}
			}
			return u, ClassifyPortalOrigin(parsed.CtwaClid, parsed.SourceID, u)
		}
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
		if phone != "" {
			u.Phone = phone
		}
		if lidDigits != "" {
			u.WaLid = lidDigits
		}
		if first != "" && u.FirstName == "" {
			u.FirstName = first
		}
		if last != "" && u.LastName == "" {
			u.LastName = last
		}
		if parsed.CtwaClid != "" {
			u.CtwaClid = parsed.CtwaClid
		}
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

	return nil, ""
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

var rxMoneyExplicit = regexp.MustCompile(`(?i)(?:R\$\s*\d{1,3}(?:[.,]\d{3})*(?:[.,]\d{2})?|R\$\s*\d+(?:[.,]\d{2})?|\b\d{1,3}(?:\.\d{3})+,\d{2}\b|\b\d+,\d{2}\b|\b\d{1,3}(?:,\d{3})+\.\d{2}\b)`)

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
