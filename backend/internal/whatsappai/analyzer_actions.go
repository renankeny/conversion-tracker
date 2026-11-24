package whatsappai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func (a *Analyzer) applyQualification(
	ctx context.Context,
	pixel *domain.Pixel,
	u *domain.TrackedUser,
	cand repo.AIChatCandidate,
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

	deduplicated := false
	shouldEnqueue := false
	dispatchedEventID := ""
	if pixel != nil && pixel.Enabled && a.Enqueuer != nil {
		alreadySent, _ := a.AI.HasLeadConversionSent(ctx, cand.AccountID, cand.TrackedUserID, pixel.Name)
		if alreadySent {
			deduplicated = true
		} else {
			thresholdOK := true
			if (cand.CtwaClid != "" || u.CtwaClid != "") && a.WhatsApp != nil {
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
		return false
	}

	if shouldEnqueue {
		in := buildAISendEventInput(ctx, a.Users, a.MetaAcc, pixel, u, cand, domain.EventLead, dispatchedEventID, evTime, nil, cand.Currency)
		if err := a.Enqueuer.EnqueueSendEvent(ctx, in); err != nil {
			slog.Error("enqueue ai qualification event failed", "pixel_name", pixel.Name, "event_id", dispatchedEventID, "error", err)
		}
	}

	a.publishUpdates(ctx, cand.AccountID, evRec, newStagePtr != nil)
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

	snap := a.AI.GetCRMLeadSnapshot(ctx, cand.AccountID, cand.TrackedUserID)
	saleAmount := s.Amount
	if saleAmount <= 0 && snap.Value > 0 {
		saleAmount = snap.Value
	}
	if saleAmount <= 0 && a.WhatsApp != nil {
		if avg := a.WhatsApp.GetChatInstanceAverageTicket(ctx, cand.ChatID); avg > 0 {
			saleAmount = avg
		}
	}
	if saleAmount <= 0 && pixel != nil && len(pixel.TrackerConfig.ProductValue) > 0 && pixel.TrackerConfig.ProductValue[0] > 0 {
		saleAmount = pixel.TrackerConfig.ProductValue[0]
	}
	if a.AI.HasRecentActiveChatAISale(ctx, cand.ChatID, saleAmount, evTime) {
		return false
	}
	effCurrency := domain.NormalizeWhatsAppCurrency(firstStr(s.Currency, cand.Currency))

	deduplicated := false
	shouldEnqueue := false
	dispatchedEventID := ""
	if pixel != nil && pixel.Enabled {
		dupInFlight := cache.HasInFlightMatchingPurchase(ctx, a.Redis, cand.AccountID, pixel.Name, u.GroupID, cand.TrackedUserID, u.ExternalID, firstStr(cand.Phone, u.Phone), u.Email, "", true, saleAmount, evTime)
		alreadySent := false
		if !dupInFlight {
			alreadySent, _ = a.AI.HasMatchingPurchaseSent(ctx, cand.AccountID, cand.TrackedUserID, pixel.Name, saleAmount, evTime)
		}
		if dupInFlight || alreadySent {
			deduplicated = true
		} else if a.Enqueuer != nil && saleAmount > 0 {
			dispatchedEventID = fmt.Sprintf("ai_sale:%d:%d", cand.ChatID, evMsgID)
			shouldEnqueue = true
		} else if saleAmount <= 0 {
			slog.Warn("skipping ai Purchase CAPI dispatch without positive value", "chat_id", cand.ChatID)
		}
	}

	var prevStagePtr, newStagePtr *int64
	if snap.StageID > 0 {
		st := snap.StageID
		prevStagePtr = &st
	}
	targetStage := saleStageID
	if targetStage == 0 {
		targetStage = snap.StageID
	}
	if targetStage > 0 && targetStage != snap.StageID {
		ns := targetStage
		newStagePtr = &ns
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
		EventType: domain.AIEventPurchase, EvidenceMessageID: &evMsgID, Amount: saleAmount, Currency: effCurrency,
		AIReason: strings.TrimSpace(s.Reason), CRMPrevStageID: prevStagePtr, CRMNewStageID: newStagePtr,
		CRMPrevValue: snap.Value, DispatchedEventID: dispatchedEventID, Deduplicated: deduplicated,
	}
	inserted, err := a.AI.CreateAIEvent(ctx, evRec)
	if err != nil || !inserted {
		return false
	}

	crmChanged := false
	if a.CRM != nil && targetStage > 0 {
		nextCRMVal := a.CRM.ComputeNextPurchaseLeadValue(ctx, cand.AccountID, cand.TrackedUserID, snap.Value, saleAmount, !deduplicated)
		if snap.LeadID == 0 {
			if id, _, cerr := a.CRM.CreateOrUpdateLeadAPI(ctx, cand.AccountID, cand.TrackedUserID, targetStage, nextCRMVal); cerr == nil && id > 0 {
				evRec.CRMLeadID = &id
				crmChanged = true
			}
		} else if snap.StageID != targetStage || nextCRMVal != snap.Value {
			if a.CRM.UpdateLeadStageAndValue(ctx, snap.LeadID, cand.AccountID, targetStage, nextCRMVal) == nil {
				crmChanged = true
			}
		}
	}

	if shouldEnqueue {
		v := saleAmount
		in := buildAISendEventInput(ctx, a.Users, a.MetaAcc, pixel, u, cand, domain.EventPurchase, dispatchedEventID, evTime, &v, effCurrency)
		cache.RecordInFlightPurchase(ctx, a.Redis, cand.AccountID, pixel.Name, u.GroupID, cand.TrackedUserID, u.ExternalID, firstStr(cand.Phone, u.Phone), u.Email, cache.InFlightPurchase{
			EventID: dispatchedEventID, IsNonOrder: true, Value: saleAmount, AtUnix: evTime.Unix(),
		})
		if err := a.Enqueuer.EnqueueSendEvent(ctx, in); err != nil {
			slog.Error("enqueue ai purchase event failed", "pixel_name", pixel.Name, "event_id", dispatchedEventID, "error", err)
		}
	}

	a.publishUpdates(ctx, cand.AccountID, evRec, crmChanged)
	return true
}
