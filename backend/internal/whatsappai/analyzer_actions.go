package whatsappai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func (a *Analyzer) applyQualification(
	ctx context.Context,
	pixel *domain.Pixel,
	u *domain.TrackedUser,
	cand repo.AIChatCandidate,
	qualStage *domain.CRMStage,
	qualStageID, saleStageID int64,
	q QualifiedDecision,
	msgByID map[int64]*domain.WhatsAppMessage,
) bool {
	if a.AI.HasActiveChatAIQualification(ctx, cand.ChatID) {
		return false
	}
	evMsgID := resolveValidMsgID(q.TriggerMessageID, cand.NewestMsgID, msgByID)
	evTime := time.Now()
	if m, ok := msgByID[evMsgID]; ok && !m.Timestamp.IsZero() {
		evTime = m.Timestamp
	}
	snap := a.AI.GetCRMLeadSnapshot(ctx, cand.AccountID, cand.TrackedUserID)
	isCTWA := cand.CtwaClid != "" || u.CtwaClid != ""
	targetEvent := resolveAIQualificationEvent(qualStage, isCTWA)

	deduplicated := false
	shouldEnqueue := false
	dispatchedEventID := ""
	if pixel != nil && pixel.Enabled && a.Enqueuer != nil {
		alreadySent := false
		if targetEvent == domain.EventLead || targetEvent == domain.EventLeadSubmitted {
			alreadySent, _ = a.AI.HasLeadConversionSent(ctx, cand.AccountID, cand.TrackedUserID, pixel.Name)
		} else {
			alreadySent, _ = a.AI.HasEventNameSent(ctx, cand.AccountID, cand.TrackedUserID, pixel.Name, targetEvent)
		}
		if alreadySent {
			deduplicated = true
		} else {
			thresholdOK := true
			if isCTWA && targetEvent == domain.EventLeadSubmitted && a.WhatsApp != nil {
				if hasChat, reached := a.WhatsApp.CheckCtwaInboundThreshold(ctx, cand.AccountID, cand.TrackedUserID); hasChat && !reached {
					thresholdOK = false
				}
			}
			if thresholdOK {
				dispatchedEventID = fmt.Sprintf("ai_qual:%d", cand.ChatID)
				shouldEnqueue = true
			}
		}
	}

	var prevStagePtr, newStagePtr *int64
	if snap.StageID > 0 {
		s := snap.StageID
		prevStagePtr = &s
	}
	crmChanged := false
	if qualStageID > 0 && a.CRM != nil {
		if snap.LeadID == 0 {
			if id, _, err := a.CRM.CreateOrUpdateLeadAPI(ctx, cand.AccountID, cand.TrackedUserID, qualStageID, 0); err != nil {
				slog.Warn("whatsapp ai create crm lead on qualification failed", "account_id", cand.AccountID, "chat_id", cand.ChatID, "error", err)
			} else if id > 0 {
				snap.LeadID = id
				ns := qualStageID
				newStagePtr = &ns
				crmChanged = true
				_ = a.AI.LinkChatToCRMLead(ctx, cand.ChatID, id)
			}
		} else if !snap.IsWonStage && snap.StageID != qualStageID && snap.StageID != saleStageID {
			qualOrder := a.AI.GetStageOrder(ctx, cand.AccountID, qualStageID)
			if snap.StageID == 0 || snap.StageOrder < qualOrder {
				if err := a.CRM.UpdateLeadStageAndValue(ctx, snap.LeadID, cand.AccountID, qualStageID, snap.Value); err != nil {
					slog.Warn("whatsapp ai move crm lead on qualification failed", "account_id", cand.AccountID, "lead_id", snap.LeadID, "stage_id", qualStageID, "error", err)
				} else {
					ns := qualStageID
					newStagePtr = &ns
					crmChanged = true
				}
			}
		}
	}

	var leadPtr *int64
	if snap.LeadID > 0 {
		id := snap.LeadID
		leadPtr = &id
	} else if cand.CRMLeadID != nil {
		leadPtr = cand.CRMLeadID
	}
	tuID := cand.TrackedUserID
	evRec := &domain.WhatsAppAIEvent{
		AccountID: cand.AccountID, ChatID: cand.ChatID, CRMLeadID: leadPtr, TrackedUserID: &tuID,
		EventType: domain.AIEventQualified, EvidenceMessageID: &evMsgID, Currency: cand.Currency,
		AIReason: strings.TrimSpace(q.Reason), CRMPrevStageID: prevStagePtr, CRMNewStageID: newStagePtr,
		CRMPrevValue: snap.Value, DispatchedEventID: dispatchedEventID, Deduplicated: deduplicated,
	}
	inserted, err := a.AI.CreateAIEvent(ctx, evRec)
	if err != nil || !inserted {
		if err != nil {
			slog.Warn("whatsapp ai insert qualification event failed", "chat_id", cand.ChatID, "error", err)
		}
		return false
	}

	if shouldEnqueue {
		in := buildAISendEventInput(ctx, a.Users, a.MetaAcc, pixel, qualStage, u, cand, targetEvent, dispatchedEventID, evTime, nil, cand.Currency)
		if err := a.Enqueuer.EnqueueSendEvent(ctx, in); err != nil {
			slog.Error("enqueue ai qualification event failed", "pixel_name", pixel.Name, "event_id", dispatchedEventID, "error", err)
		}
	}

	a.publishUpdates(ctx, cand.AccountID, evRec, crmChanged)
	if a.Notify != nil {
		who := strings.TrimSpace(u.FirstName + " " + u.LastName)
		if who == "" {
			who = firstStr(u.Phone, "Lead #"+fmt.Sprint(u.ID))
		}
		body := fmt.Sprintf("A IA qualificou %s na conversa de WhatsApp.", who)
		if r := strings.TrimSpace(q.Reason); r != "" {
			body += " Motivo: " + r
		}
		a.Notify.DispatchAsync(notify.Event{
			AccountID: cand.AccountID, Category: domain.NotifAILeadQualified,
			Title: "Lead Qualificado pela IA", Body: body, ActionURL: "/crm",
		})
	}
	return true
}
