package whatsappai

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

func TestBuildSystemPrompt_IncludesContestRulesAndQualificationGuard(t *testing.T) {
	rules := []domain.AILearnedRule{
		{ID: "r_1", EventType: domain.AIEventPurchase, Rule: "Print de carrinho sem comprovante pago não é venda."},
	}
	priorSales := []repo.PriorPurchaseRef{
		{Source: "ia", MessageID: 42, Amount: 297.00, CreatedAt: time.Date(2026, 11, 13, 10, 30, 0, 0, time.UTC)},
	}
	out := BuildSystemPrompt("", rules, true, priorSales, 45)

	if !strings.Contains(out, "ESTE LEAD JÁ FOI QUALIFICADO ANTERIORMENTE") {
		t.Fatal("expected already-qualified guard in system prompt")
	}
	if !strings.Contains(out, "Print de carrinho sem comprovante pago não é venda.") {
		t.Fatal("expected contested rule to be injected into system prompt")
	}
	if !strings.Contains(out, "297.00") {
		t.Fatal("expected prior sale reference in system prompt")
	}
}

func TestAudioFilenameForMime(t *testing.T) {
	if got := audioFilenameForMime("audio/ogg; codecs=opus", ""); got != "audio.ogg" {
		t.Fatalf("expected audio.ogg, got %q", got)
	}
	if got := audioFilenameForMime("audio/mp4", "voice.bin"); got != "audio.m4a" {
		t.Fatalf("expected audio.m4a, got %q", got)
	}
	if got := audioFilenameForMime("", "nota.mp3"); got != "audio.mp3" {
		t.Fatalf("expected audio.mp3, got %q", got)
	}
}

func TestIsPDFMessage(t *testing.T) {
	pdfMime := "application/pdf"
	imgMime := "image/jpeg"
	if !isPDFMessage(&domain.WhatsAppMessage{MessageType: "document", MediaMimeType: &pdfMime}) {
		t.Fatal("expected application/pdf mime to be detected as PDF")
	}
	if !isPDFMessage(&domain.WhatsAppMessage{MessageType: "document", MediaFilename: "Comprovante_PIX.PDF"}) {
		t.Fatal("expected .PDF filename to be detected as PDF")
	}
	if isPDFMessage(&domain.WhatsAppMessage{MessageType: "document", MediaMimeType: &imgMime, MediaFilename: "planilha.xlsx"}) {
		t.Fatal("expected xlsx not to be detected as PDF")
	}
}

func TestResponseJSONSchema_SerializesCleanly(t *testing.T) {
	raw, err := json.Marshal(responseJSONSchema())
	if err != nil || len(raw) == 0 {
		t.Fatalf("failed to marshal responseJSONSchema: %v", err)
	}
	if !strings.Contains(string(raw), `"strict":true`) {
		t.Fatal("expected strict json_schema enabled")
	}
}

func TestResolveAIQualificationEvent(t *testing.T) {
	if got := resolveAIQualificationEvent(nil, true); got != domain.EventQualifiedLead {
		t.Fatalf("expected QualifiedLead for CTWA without stage trigger, got %q", got)
	}
	if got := resolveAIQualificationEvent(nil, false); got != domain.EventLead {
		t.Fatalf("expected Lead for web without stage trigger, got %q", got)
	}
	schedStage := &domain.CRMStage{TriggerEvent: "Schedule", TriggerEventCTWA: "Schedule"}
	if got := resolveAIQualificationEvent(schedStage, true); got != "Schedule" {
		t.Fatalf("expected Schedule from stage trigger, got %q", got)
	}
	purchaseStage := &domain.CRMStage{TriggerEvent: "Purchase", TriggerEventCTWA: "Purchase"}
	if got := resolveAIQualificationEvent(purchaseStage, true); got != domain.EventQualifiedLead {
		t.Fatalf("expected Purchase stage trigger to fall back to QualifiedLead, got %q", got)
	}
}

func TestOpenAIErrorClassificationAndModels(t *testing.T) {
	if !(&openAIHTTPError{StatusCode: 401, Body: "invalid_api_key"}).isFatalAccountError() {
		t.Fatal("expected 401 to be fatal")
	}
	if !(&openAIHTTPError{StatusCode: 429, Body: `{"error":{"code":"insufficient_quota"}}`}).isFatalAccountError() {
		t.Fatal("expected 429 insufficient_quota to be fatal")
	}
	if (&openAIHTTPError{StatusCode: 429, Body: `{"error":{"code":"rate_limit_exceeded"}}`}).isFatalAccountError() {
		t.Fatal("expected transient 429 rate limit not to be fatal")
	}
	if !(&openAIHTTPError{StatusCode: 400, Body: "invalid file"}).isBadRequest() {
		t.Fatal("expected 400 to be bad request")
	}
	if got := normalizeModelName("gpt-4.1-mini"); got != "gpt-4.1-mini" {
		t.Fatalf("expected gpt-4.1-mini, got %q", got)
	}
	if got := normalizeModelName("unknown-model"); got != "gpt-4o-mini" {
		t.Fatalf("expected fallback gpt-4o-mini, got %q", got)
	}
}

func TestShouldAttachMedia_DelayedReceiptRecovery(t *testing.T) {
	recent := &domain.WhatsAppMessage{
		MediaStatus:    domain.MediaStatusReady,
		MediaObjectKey: "k/1.jpg",
		CreatedAt:      time.Now().Add(-20 * time.Minute),
	}
	if !shouldAttachMedia(recent, false) {
		t.Fatal("expected recent ready media without transcript to be reattached")
	}
	recent.AITranscript = "Comprovante PIX R$ 297,00"
	if shouldAttachMedia(recent, false) {
		t.Fatal("expected already-summarized old media not to be reattached")
	}
}

