package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func parseConversationAuditFilter(r *http.Request, accountID int64) domain.ConversationAuditFilter {
	q := r.URL.Query()
	from, to := parseDashboardTimeBounds(r)
	if from.IsZero() && q.Get("all_time") != "1" {
		from = time.Now().AddDate(0, 0, -7)
	}
	if to.IsZero() && q.Get("all_time") != "1" {
		to = time.Now()
	}

	limit := 50
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 5000 {
		limit = n
	}
	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n >= 0 {
		offset = n
	}
	instanceID, _ := strconv.ParseInt(q.Get("instance_id"), 10, 64)
	stageID, _ := strconv.ParseInt(q.Get("stage_id"), 10, 64)
	tagID, _ := strconv.ParseInt(q.Get("tag_id"), 10, 64)
	pixelID, _ := strconv.ParseInt(q.Get("pixel_id"), 10, 64)

	return domain.ConversationAuditFilter{
		AccountID:      accountID,
		PixelName:      strings.TrimSpace(q.Get("pixel")),
		PixelID:        pixelID,
		InstanceID:     instanceID,
		From:           from,
		To:             to,
		Search:         strings.TrimSpace(q.Get("search")),
		SLAStatus:      strings.TrimSpace(q.Get("sla_status")),
		StageID:        stageID,
		TagID:          tagID,
		OriginCategory: strings.TrimSpace(q.Get("origin")),
		Campaign:       strings.TrimSpace(q.Get("campaign")),
		UTM:            strings.TrimSpace(q.Get("utm")),
		SortBy:         strings.TrimSpace(q.Get("sort_by")),
		SortDir:        strings.TrimSpace(q.Get("sort_dir")),
		Limit:          limit,
		Offset:         offset,
	}
}

// ListAuditConversations GET /admin/whatsapp/conversations
func (h *AdminWhatsAppChats) ListAuditConversations(w http.ResponseWriter, r *http.Request) {
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	filter := parseConversationAuditFilter(r, accID)
	res, err := h.repo.ListAuditConversations(r.Context(), filter)
	if err != nil {
		slog.Error("whatsapp list audit conversations failed", "account_id", accID, "error", err)
		writeError(w, http.StatusInternalServerError, "erro ao carregar conversas")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// WaitingCount GET /admin/whatsapp/conversations/waiting-count
func (h *AdminWhatsAppChats) WaitingCount(w http.ResponseWriter, r *http.Request) {
	accID := TenantFromContext(r.Context()).EffectiveAccountID(0)
	count, err := h.repo.CountWaitingConversations(r.Context(), accID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]int{"waiting_count": 0})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"waiting_count": count})
}
