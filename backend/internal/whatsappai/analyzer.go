package whatsappai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
)

type Analyzer struct {
	AI       *repo.WhatsAppAIRepo
	WhatsApp *repo.WhatsAppRepo
	Pixels   *repo.PixelRepo
	Users    *repo.UserRepo
	CRM      *repo.CRMRepo
	MetaAcc  *repo.MetaAccountRepo
	Storage  *storage.MediaStorage
	Enqueuer EventEnqueuer
	Redis    *redis.Client
}

type RunSummary struct {
	AnalyzedChats  int `json:"analyzed_chats"`
	Qualifications int `json:"qualifications"`
	Sales          int `json:"sales"`
}

func (a *Analyzer) AnalyzeAccount(ctx context.Context, accountID, fallbackUserID int64, force bool) (RunSummary, error) {
	var sum RunSummary
	if a == nil || a.AI == nil {
		return sum, nil
	}
	settings, err := a.AI.GetSettings(ctx, accountID)
	if err != nil {
		return sum, err
	}
	if !force && !settings.Enabled {
		return sum, nil
	}

	apiKey := a.AI.ResolveOpenAIKey(ctx, accountID, fallbackUserID)
	if strings.TrimSpace(apiKey) == "" {
		if force {
			return sum, fmt.Errorf("nenhuma chave da OpenAI configurada (adicione em Configurações > Meu Perfil)")
		}
		return sum, nil
	}

	qualStageID, saleStageID := a.AI.ResolveTargetStages(ctx, accountID, settings.QualifiedStageID, settings.SaleStageID)
	candidates, err := a.AI.ListCandidateChats(ctx, accountID, 40)
	if err != nil {
		return sum, err
	}

	for _, cand := range candidates {
		if cand.HasRecentPendingMedia && !force {
			continue
		}
		qInc, sInc, analyzed := a.analyzeSingleChat(ctx, apiKey, settings, qualStageID, saleStageID, cand)
		if analyzed {
			sum.AnalyzedChats++
			sum.Qualifications += qInc
			sum.Sales += sInc
		}
	}
	_ = a.AI.TouchLastRun(ctx, accountID)
	return sum, nil
}

func (a *Analyzer) analyzeSingleChat(
	ctx context.Context,
	apiKey string,
	settings *domain.AccountAISettings,
	qualStageID, saleStageID int64,
	cand repo.AIChatCandidate,
) (qCount, sCount int, analyzed bool) {
	msgs, err := a.AI.ListChatMessagesForAI(ctx, cand.ChatID, 50)
	if err != nil || len(msgs) == 0 {
		return 0, 0, false
	}
	hasInbound := false
	msgByID := make(map[int64]*domain.WhatsAppMessage, len(msgs))
	for _, m := range msgs {
		msgByID[m.ID] = m
		if !m.IsFromMe {
			hasInbound = true
		}
	}
	if !hasInbound {
		_ = a.AI.UpdateChatAICursor(ctx, cand.ChatID, cand.NewestMsgID, false)
		return 0, 0, false
	}

	u, _ := a.Users.FindByID(ctx, cand.TrackedUserID)
	if u == nil {
		return 0, 0, false
	}
	pixel, _ := a.Pixels.FindByID(ctx, cand.PixelID)

	priorSales := a.AI.ListPriorPurchasesSummary(ctx, cand.AccountID, cand.ChatID, cand.TrackedUserID)
	sysPrompt := BuildSystemPrompt(settings.BasePrompt, settings.LearnedRules, cand.AIQualified, priorSales, cand.AILastAnalyzedMsgID)
	userParts := buildMultimodalUserParts(ctx, apiKey, a.Storage, a.WhatsApp, msgs, cand.AILastAnalyzedMsgID)

	res, err := callOpenAIAnalysis(ctx, apiKey, settings.Model, sysPrompt, userParts)
	if err != nil {
		slog.Warn("whatsapp ai analysis failed", "account_id", cand.AccountID, "chat_id", cand.ChatID, "error", err)
		return 0, 0, false
	}

	for _, ms := range res.MediaSummaries {
		if m, ok := msgByID[ms.MessageID]; ok && m.MessageType != "audio" && strings.TrimSpace(ms.Summary) != "" {
			_ = a.WhatsApp.SaveMessageAITranscript(ctx, ms.MessageID, strings.TrimSpace(ms.Summary))
		}
	}

	markQualified := cand.AIQualified
	if !cand.AIQualified && res.Qualified.IsQualified && res.Qualified.Confidence == "high" {
		if a.applyQualification(ctx, pixel, u, cand, qualStageID, saleStageID, res.Qualified, msgByID) {
			qCount++
		}
		markQualified = true
	}

	for _, sale := range res.NewSales {
		if sale.Confidence != "certain" {
			continue
		}
		if a.applySale(ctx, pixel, u, cand, saleStageID, sale, msgByID) {
			sCount++
			markQualified = true
		}
	}

	_ = a.AI.UpdateChatAICursor(ctx, cand.ChatID, cand.NewestMsgID, markQualified)
	return qCount, sCount, true
}
