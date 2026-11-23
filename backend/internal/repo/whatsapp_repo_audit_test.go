package repo

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestBuildAuditBaseWhere_PlaceholdersMatchArgs(t *testing.T) {
	now := time.Now()
	f := domain.ConversationAuditFilter{
		AccountID:  7,
		PixelID:    12,
		PixelName:  "clinica-odonto",
		InstanceID: 3,
		From:       now.Add(-24 * time.Hour),
		To:         now,
		Search:     "Maria (11) 98888-7777",
		SLAStatus:  "pending",
		StageID:    5,
		TagID:      9,
		Campaign:   "Campanha Implante",
		UTM:        "facebook",
	}
	whereSQL, args := buildAuditBaseWhere(f)
	if len(args) != 12 {
		t.Fatalf("expected 12 bound args, got %d (%v)", len(args), args)
	}
	for i := 1; i <= len(args); i++ {
		tok := fmt.Sprintf("$%d", i)
		if !strings.Contains(whereSQL, tok) {
			t.Errorf("expected placeholder %s in SQL: %s", tok, whereSQL)
		}
	}
	if !strings.Contains(whereSQL, "c.sla_status IN ('unanswered', 'waiting_reply')") {
		t.Errorf("expected pending SLA clause, got %s", whereSQL)
	}
}

func TestResolveAuditOrderBy_RejectsArbitraryInput(t *testing.T) {
	got := resolveAuditOrderBy("DROP TABLE whatsapp_chats;--", "INVALID")
	if got != "last_message_time DESC, chat_id DESC" {
		t.Fatalf("unexpected fallback order: %q", got)
	}
	gotAsc := resolveAuditOrderBy("first_response_sec", "asc")
	if gotAsc != "first_response_sec ASC NULLS LAST, last_message_time DESC" {
		t.Fatalf("unexpected asc order: %q", gotAsc)
	}
}
