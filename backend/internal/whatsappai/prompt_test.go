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

