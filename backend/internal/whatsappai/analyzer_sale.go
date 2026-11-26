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

func (a *Analyzer) applySale(
	ctx context.Context,
	pixel *domain.Pixel,
	u *domain.TrackedUser,
	cand repo.AIChatCandidate,
	saleStage *domain.CRMStage,
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
	reason := strings.TrimSpace(s.Reason)
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
			slog.Warn("skipping ai Purchase CAPI dispatch without positive value", "account_id", cand.AccountID, "chat_id", cand.ChatID)
			reason = strings.TrimSpace(reason + " (Valor não identificado na conversa — configure o Ticket Médio na instância WhatsApp para disparar o evento Purchase.)")
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
		AIReason: reason, CRMPrevStageID: prevStagePtr, CRMNewStageID: newStagePtr,
		CRMPrevValue: snap.Value, DispatchedEventID: dispatchedEventID, Deduplicated: deduplicated,
	}
	inserted, err := a.AI.CreateAIEvent(ctx, evRec)
	if err != nil || !inserted {
		if err != nil {
			slog.Warn("whatsapp ai insert sale event failed", "chat_id", cand.ChatID, "error", err)
		}
		return false
	}

	crmChanged := false
	if a.CRM != nil && targetStage > 0 {
		nextCRMVal := a.CRM.ComputeNextPurchaseLeadValue(ctx, cand.AccountID, cand.TrackedUserID, snap.Value, saleAmount, !deduplicated)
		if snap.LeadID == 0 {
			if id, _, cerr := a.CRM.CreateOrUpdateLeadAPI(ctx, cand.AccountID, cand.TrackedUserID, targetStage, nextCRMVal); cerr != nil {
				slog.Warn("whatsapp ai create crm lead on sale failed", "account_id", cand.AccountID, "chat_id", cand.ChatID, "error", cerr)
			} else if id > 0 {
				evRec.CRMLeadID = &id
				crmChanged = true
				_ = a.AI.LinkChatToCRMLead(ctx, cand.ChatID, id)
			}
		} else if snap.StageID != targetStage || nextCRMVal != snap.Value {
			if cerr := a.CRM.UpdateLeadStageAndValue(ctx, snap.LeadID, cand.AccountID, targetStage, nextCRMVal); cerr != nil {
				slog.Warn("whatsapp ai move crm lead on sale failed", "account_id", cand.AccountID, "lead_id", snap.LeadID, "stage_id", targetStage, "error", cerr)
			} else {
				crmChanged = true
			}
		}
	}

	if shouldEnqueue {
		v := saleAmount
		in := buildAISendEventInput(ctx, a.Users, a.MetaAcc, pixel, saleStage, u, cand, domain.EventPurchase, dispatchedEventID, evTime, &v, effCurrency)
		cache.RecordInFlightPurchase(ctx, a.Redis, cand.AccountID, pixel.Name, u.GroupID, cand.TrackedUserID, u.ExternalID, firstStr(cand.Phone, u.Phone), u.Email, cache.InFlightPurchase{
			EventID: dispatchedEventID, IsNonOrder: true, Value: saleAmount, AtUnix: evTime.Unix(),
		})
		if err := a.Enqueuer.EnqueueSendEvent(ctx, in); err != nil {
			slog.Error("enqueue ai purchase event failed", "pixel_name", pixel.Name, "event_id", dispatchedEventID, "error", err)
		}
	}

	a.publishUpdates(ctx, cand.AccountID, evRec, crmChanged)
	if a.Notify != nil && !deduplicated && !shouldEnqueue {
		who := firstStr(strings.TrimSpace(u.FirstName+" "+u.LastName), u.Phone, "Lead #"+fmt.Sprint(u.ID))
		a.Notify.DispatchAsync(notify.Event{
			AccountID: cand.AccountID, Category: domain.NotifSaleApproved,
			Title: "Venda Identificada pela IA", Body: fmt.Sprintf("A IA confirmou venda para %s no WhatsApp.", who), ActionURL: "/crm",
		})
	}
	return true
}
