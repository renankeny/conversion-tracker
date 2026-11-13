package whatsappai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

func resolveValidMsgID(msgID, fallbackID int64, msgByID map[int64]*domain.WhatsAppMessage) int64 {
	if msgID > 0 {
		if _, ok := msgByID[msgID]; ok {
			return msgID
		}
	}
	return fallbackID
}

func (a *Analyzer) applyQualification(
	ctx context.Context,
	pixel *domain.Pixel,
	u *domain.TrackedUser,
	cand repo.AIChatCandidate,
	qualStageID, saleStageID int64,
	q QualifiedDecision,
	msgByID map[int64]*domain.WhatsAppMessage,
) bool {
	evMsgID := resolveValidMsgID(q.TriggerMessageID, cand.NewestMsgID, msgByID)
	snap := a.AI.GetCRMLeadSnapshot(ctx, cand.AccountID, cand.TrackedUserID)

	var prevStagePtr, newStagePtr *int64
	if snap.StageID > 0 {
		s := snap.StageID
		prevStagePtr = &s
	}

	if qualStageID > 0 && a.CRM != nil {
		if snap.LeadID == 0 {
			if id, _, err := a.CRM.CreateOrUpdateLeadAPI(ctx, cand.AccountID, cand.TrackedUserID, qualStageID, 0); err == nil && id > 0 {
				snap.LeadID = id
				ns := qualStageID
				newStagePtr = &ns
			}
		} else if snap.StageID != qualStageID && snap.StageID != saleStageID {
			qualOrder := a.AI.GetStageOrder(ctx, cand.AccountID, qualStageID)
			if snap.StageOrder < qualOrder {
				if err := a.CRM.UpdateLeadStageAndValue(ctx, snap.LeadID, cand.AccountID, qualStageID, snap.Value); err == nil {
					ns := qualStageID
					newStagePtr = &ns
				}
			}
		}
	}

	deduplicated := false
	dispatchedEventID := ""
	if pixel != nil && pixel.Enabled && a.Enqueuer != nil {
		alreadySent, _ := a.AI.HasLeadConversionSent(ctx, cand.AccountID, cand.TrackedUserID, pixel.Name)
		if alreadySent {
			deduplicated = true
		} else {
			evID := fmt.Sprintf("ai_qual:%d", cand.ChatID)
			in := buildAISendEventInput(ctx, a.Users, a.MetaAcc, pixel, u, cand, domain.EventLead, evID, nil)
			if err := a.Enqueuer.EnqueueSendEvent(ctx, in); err != nil {
				slog.Error("enqueue ai qualification event failed", "pixel_name", pixel.Name, "event_id", evID, "error", err)
			} else {
				dispatchedEventID = evID
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
		AccountID:         cand.AccountID,
		ChatID:            cand.ChatID,
		CRMLeadID:         leadPtr,
		TrackedUserID:     &tuID,
		EventType:         domain.AIEventQualified,
		EvidenceMessageID: &evMsgID,
		Currency:          cand.Currency,
		AIReason:          strings.TrimSpace(q.Reason),
		CRMPrevStageID:    prevStagePtr,
		CRMNewStageID:     newStagePtr,
		CRMPrevValue:      snap.Value,
		DispatchedEventID: dispatchedEventID,
		Deduplicated:      deduplicated,
	}
	inserted, err := a.AI.CreateAIEvent(ctx, evRec)
	if err != nil || !inserted {
		return false
	}
	a.publishUpdates(ctx, cand.AccountID, evRec, newStagePtr != nil)
	return true
}

func (a *Analyzer) applySale(
	ctx context.Context,
	pixel *domain.Pixel,
	u *domain.TrackedUser,
	cand repo.AIChatCandidate,
	saleStageID int64,
	s SaleDecision,
	msgByID map[int64]*domain.WhatsAppMessage,
) bool {
	evMsgID := resolveValidMsgID(s.EvidenceMessageID, cand.NewestMsgID, msgByID)
	evTime := time.Now()
	if m, ok := msgByID[evMsgID]; ok && !m.Timestamp.IsZero() {
		evTime = m.Timestamp
	}

	saleAmount := s.Amount
	if saleAmount <= 0 && pixel != nil && len(pixel.TrackerConfig.ProductValue) > 0 && pixel.TrackerConfig.ProductValue[0] > 0 {
		saleAmount = pixel.TrackerConfig.ProductValue[0]
	}

	deduplicated := false
	dispatchedEventID := ""
	if pixel != nil && pixel.Enabled {
		alreadySent, _ := a.AI.HasMatchingPurchaseSent(ctx, cand.AccountID, cand.TrackedUserID, pixel.Name, saleAmount, evTime)
		if alreadySent {
			deduplicated = true
		} else if a.Enqueuer != nil {
			evID := fmt.Sprintf("ai_sale:%d:%d", cand.ChatID, evMsgID)
			var valPtr *float64
			if saleAmount > 0 {
				v := saleAmount
				valPtr = &v
			}
			in := buildAISendEventInput(ctx, a.Users, a.MetaAcc, pixel, u, cand, domain.EventPurchase, evID, valPtr)
			if err := a.Enqueuer.EnqueueSendEvent(ctx, in); err != nil {
				slog.Error("enqueue ai purchase event failed", "pixel_name", pixel.Name, "event_id", evID, "error", err)
			} else {
				dispatchedEventID = evID
			}
		}
	}

	snap := a.AI.GetCRMLeadSnapshot(ctx, cand.AccountID, cand.TrackedUserID)
	var prevStagePtr, newStagePtr *int64
	if snap.StageID > 0 {
		st := snap.StageID
		prevStagePtr = &st
	}
	newCRMVal := snap.Value
	if !deduplicated && saleAmount > 0 {
		newCRMVal = snap.Value + saleAmount
	} else if snap.Value <= 0 && saleAmount > 0 {
		newCRMVal = saleAmount
	}

	if a.CRM != nil {
		targetStage := saleStageID
		if targetStage == 0 {
			targetStage = snap.StageID
		}
		if snap.LeadID == 0 && targetStage > 0 {
			if id, _, err := a.CRM.CreateOrUpdateLeadAPI(ctx, cand.AccountID, cand.TrackedUserID, targetStage, newCRMVal); err == nil && id > 0 {
				snap.LeadID = id
				ns := targetStage
				newStagePtr = &ns
			}
		} else if snap.LeadID > 0 && targetStage > 0 && (snap.StageID != targetStage || newCRMVal != snap.Value) {
			if err := a.CRM.UpdateLeadStageAndValue(ctx, snap.LeadID, cand.AccountID, targetStage, newCRMVal); err == nil {
				ns := targetStage
				newStagePtr = &ns
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
		AccountID:         cand.AccountID,
		ChatID:            cand.ChatID,
		CRMLeadID:         leadPtr,
		TrackedUserID:     &tuID,
		EventType:         domain.AIEventPurchase,
		EvidenceMessageID: &evMsgID,
		Amount:            saleAmount,
		Currency:          firstStr(s.Currency, cand.Currency),
		AIReason:          strings.TrimSpace(s.Reason),
		CRMPrevStageID:    prevStagePtr,
		CRMNewStageID:     newStagePtr,
		CRMPrevValue:      snap.Value,
		DispatchedEventID: dispatchedEventID,
		Deduplicated:      deduplicated,
	}
	inserted, err := a.AI.CreateAIEvent(ctx, evRec)
	if err != nil || !inserted {
		return false
	}
	a.publishUpdates(ctx, cand.AccountID, evRec, newStagePtr != nil)
	return true
}

func (a *Analyzer) publishUpdates(ctx context.Context, accountID int64, ev *domain.WhatsAppAIEvent, crmChanged bool) {
	if a.Redis == nil {
		return
	}
	wshub.PublishCustom(ctx, a.Redis, accountID, "whatsapp_ai_event", map[string]any{
		"chat_id":         ev.ChatID,
		"crm_lead_id":     ev.CRMLeadID,
		"tracked_user_id": ev.TrackedUserID,
		"ai_event":        ev,
	})
	if crmChanged {
		wshub.PublishCustom(ctx, a.Redis, accountID, "crm_updated", map[string]any{
			"action":  "whatsapp_ai_synced",
			"lead_id": ev.CRMLeadID,
			"event":   ev.EventType,
		})
	}
}

