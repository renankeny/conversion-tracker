package whatsappai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
)

const aiChatWorkerConcurrency = 4

type Analyzer struct {
	AI       *repo.WhatsAppAIRepo
	WhatsApp *repo.WhatsAppRepo
	Pixels   *repo.PixelRepo
	Users    *repo.UserRepo
	CRM      *repo.CRMRepo
	MetaAcc  *repo.MetaAccountRepo
	Storage  *storage.MediaStorage
	Enqueuer EventEnqueuer
	Notify   *notify.Service
	Redis    *redis.Client
}

type RunSummary struct {
	AnalyzedChats  int    `json:"analyzed_chats"`
	Qualifications int    `json:"qualifications"`
	Sales          int    `json:"sales"`
	Errors         int    `json:"errors"`
	SkippedMedia   int    `json:"skipped_media"`
	LastError      string `json:"last_error,omitempty"`
}

func (a *Analyzer) AnalyzeAccount(ctx context.Context, accountID, fallbackUserID int64, force bool) (RunSummary, error) {
	var sum RunSummary
	if a == nil || a.AI == nil {
		return sum, nil
	}
	if a.Redis != nil && accountID > 0 {
		lockKey := fmt.Sprintf("lock:whatsapp_ai_account:%d", accountID)
		ok, lerr := a.Redis.SetNX(ctx, lockKey, "1", 5*time.Minute).Result()
		if lerr == nil && !ok {
			if force {
				return sum, fmt.Errorf("uma análise da IA já está em andamento para esta conta; aguarde alguns segundos")
			}
			return sum, nil
		}
		defer a.Redis.Del(context.Background(), lockKey)
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
		msg := "Chave da OpenAI não configurada (adicione em Configurações > Meu Perfil)."
		slog.Warn("whatsapp ai skipped: missing openai api key", "account_id", accountID)
		_ = a.AI.RecordRunResult(ctx, accountID, 0, 0, 0, msg)
		if force {
			return sum, errors.New(msg)
		}
		return sum, nil
	}

	if a.CRM != nil {
		_ = a.CRM.EnsureDefaultStages(ctx, accountID)
	}
	qualStageID, saleStageID := a.AI.ResolveTargetStages(ctx, accountID, settings.QualifiedStageID, settings.SaleStageID)
	var qualStage, saleStage *domain.CRMStage
	if a.CRM != nil && qualStageID > 0 {
		qualStage, _ = a.CRM.GetStageByID(ctx, qualStageID, accountID)
	}
	if a.CRM != nil && saleStageID > 0 {
		saleStage, _ = a.CRM.GetStageByID(ctx, saleStageID, accountID)
	}

	candidates, err := a.AI.ListCandidateChats(ctx, accountID, 40)
	if err != nil {
		_ = a.AI.RecordRunResult(ctx, accountID, 0, 0, 0, "Falha ao listar conversas para IA: "+err.Error())
		return sum, err
	}

	var (
		mu         sync.Mutex
		wg         sync.WaitGroup
		fatalStop  atomic.Bool
		sem        = make(chan struct{}, aiChatWorkerConcurrency)
	)

	for _, cand := range candidates {
		if cand.HasRecentPendingMedia && !force {
			mu.Lock()
			sum.SkippedMedia++
			mu.Unlock()
			continue
		}
		if ctx.Err() != nil || fatalStop.Load() {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(c repo.AIChatCandidate) {
			defer wg.Done()
			defer func() { <-sem }()
			if ctx.Err() != nil || fatalStop.Load() {
				return
			}
			qInc, sInc, analyzed, callErr := a.analyzeSingleChat(ctx, apiKey, settings, qualStage, saleStage, qualStageID, saleStageID, c)
			mu.Lock()
			defer mu.Unlock()
			if callErr != nil {
				sum.Errors++
				sum.LastError = humanizeOpenAIError(callErr)
				var httpErr *openAIHTTPError
				if errors.As(callErr, &httpErr) && httpErr.isFatalAccountError() {
					fatalStop.Store(true)
				}
				return
			}
			if analyzed {
				sum.AnalyzedChats++
				sum.Qualifications += qInc
				sum.Sales += sInc
			}
		}(cand)
	}
	wg.Wait()

	recordedErr := ""
	if sum.Errors > 0 {
		recordedErr = sum.LastError
	}
	_ = a.AI.RecordRunResult(ctx, accountID, sum.AnalyzedChats, sum.Qualifications, sum.Sales, recordedErr)
	if force && sum.AnalyzedChats == 0 && sum.Errors > 0 && sum.LastError != "" {
		return sum, errors.New(sum.LastError)
	}
	return sum, nil
}

func (a *Analyzer) analyzeSingleChat(
	ctx context.Context,
	apiKey string,
	settings *domain.AccountAISettings,
	qualStage, saleStage *domain.CRMStage,
	qualStageID, saleStageID int64,
	cand repo.AIChatCandidate,
) (qCount, sCount int, analyzed bool, callErr error) {
	msgs, err := a.AI.ListChatMessagesForAI(ctx, cand.ChatID, 50)
	if err != nil || len(msgs) == 0 {
		return 0, 0, false, nil
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
		return 0, 0, false, nil
	}

	u, _ := a.Users.FindByID(ctx, cand.TrackedUserID)
	if u == nil {
		return 0, 0, false, nil
	}
	pixel, _ := a.Pixels.FindByID(ctx, cand.PixelID)

	priorSales := a.AI.ListPriorPurchasesSummary(ctx, cand.AccountID, cand.ChatID, cand.TrackedUserID)
	sysPrompt := BuildSystemPrompt(settings.BasePrompt, settings.LearnedRules, cand.AIQualified, priorSales, cand.AILastAnalyzedMsgID)
	built := buildMultimodalUserParts(ctx, apiKey, a.Storage, a.WhatsApp, msgs, cand.AILastAnalyzedMsgID)

	res, err := callOpenAIAnalysis(ctx, apiKey, settings.Model, sysPrompt, built.Parts)
	if err != nil && built.HasPDF {
		var httpErr *openAIHTTPError
		if errors.As(err, &httpErr) && httpErr.isBadRequest() {
			slog.Warn("whatsapp ai pdf attachment rejected by openai, retrying without pdf", "account_id", cand.AccountID, "chat_id", cand.ChatID)
			res, err = callOpenAIAnalysis(ctx, apiKey, settings.Model, sysPrompt, built.NoPDFParts)
		}
	}
	if err != nil {
		slog.Warn("whatsapp ai analysis failed", "account_id", cand.AccountID, "chat_id", cand.ChatID, "error", err)
		return 0, 0, false, err
	}

	for _, ms := range res.MediaSummaries {
		if m, ok := msgByID[ms.MessageID]; ok && m.MessageType != "audio" && strings.TrimSpace(ms.Summary) != "" {
			_ = a.WhatsApp.SaveMessageAITranscript(ctx, ms.MessageID, strings.TrimSpace(ms.Summary))
		}
	}

	markQualified := cand.AIQualified
	if !cand.AIQualified && res.Qualified.IsQualified && res.Qualified.Confidence == "high" {
		if a.applyQualification(ctx, pixel, u, cand, qualStage, qualStageID, saleStageID, res.Qualified, msgByID) {
			qCount++
			markQualified = true
		} else if a.AI.HasActiveChatAIQualification(ctx, cand.ChatID) {
			markQualified = true
		}
	}

	for _, sale := range res.NewSales {
		if sale.Confidence != "certain" {
			continue
		}
		if a.applySale(ctx, pixel, u, cand, saleStage, saleStageID, sale, msgByID) {
			sCount++
			markQualified = true
		}
	}

	effectiveCursor := cand.NewestMsgID
	if built.RetryCursorCap > 0 && built.RetryCursorCap < effectiveCursor {
		effectiveCursor = built.RetryCursorCap
	}
	_ = a.AI.UpdateChatAICursor(ctx, cand.ChatID, effectiveCursor, markQualified)
	return qCount, sCount, true, nil
}
