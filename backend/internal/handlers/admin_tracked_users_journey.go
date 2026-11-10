package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type SessionSummary struct {
	ExternalID       string    `json:"external_id"`
	CreatedAt        time.Time `json:"created_at"`
	DeviceType       string    `json:"device_type"` // "Celular", "Computador", "Tablet"
	Browser          string    `json:"browser"`     // "Google Chrome", "Apple Safari", etc.
	URL              string    `json:"url,omitempty"`
	FirstUTMSource   string    `json:"first_utm_source,omitempty"`
	FirstUTMCampaign string    `json:"first_utm_campaign,omitempty"`
	PageViewCount    int       `json:"pageview_count"`
	CheckoutCount    int       `json:"checkout_count"`
	IsInitial        bool      `json:"is_initial"`
}

func inferDeviceInfo(ua string) (deviceType string, browserName string) {
	if ua == "" {
		return "Computador", "Navegador Web"
	}
	uaLower := strings.ToLower(ua)

	// Verificação de robôs/crawlers
	if strings.Contains(uaLower, "meta-external") ||
		strings.Contains(uaLower, "facebookexternalhit") ||
		strings.Contains(uaLower, "facebot") ||
		strings.Contains(uaLower, "googlebot") ||
		strings.Contains(uaLower, "adsbot") ||
		strings.Contains(uaLower, "crawler") ||
		strings.Contains(uaLower, "spider") ||
		strings.Contains(uaLower, "bot") ||
		strings.Contains(uaLower, "scraper") {
		deviceType = "Servidor / Robô"
		switch {
		case strings.Contains(uaLower, "meta-external") || strings.Contains(uaLower, "facebookexternalhit") || strings.Contains(uaLower, "facebot"):
			browserName = "Meta Crawler"
		case strings.Contains(uaLower, "googlebot") || strings.Contains(uaLower, "adsbot"):
			browserName = "Google Crawler"
		case strings.Contains(uaLower, "tiktokbot") || strings.Contains(uaLower, "bytespider"):
			browserName = "TikTok Bot"
		default:
			browserName = "Robô Automatizado"
		}
		return
	}

	if strings.Contains(uaLower, "mobile") || strings.Contains(uaLower, "android") || strings.Contains(uaLower, "iphone") {
		deviceType = "Celular"
	} else if strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "tablet") {
		deviceType = "Tablet"
	} else {
		deviceType = "Computador"
	}

	switch {
	case strings.Contains(uaLower, "whatsapp"):
		browserName = "WhatsApp Browser"
	case strings.Contains(uaLower, "instagram"):
		browserName = "Instagram Browser"
	case strings.Contains(uaLower, "edg"):
		browserName = "Microsoft Edge"
	case strings.Contains(uaLower, "chrome") || strings.Contains(uaLower, "crios"):
		browserName = "Google Chrome"
	case strings.Contains(uaLower, "firefox") || strings.Contains(uaLower, "fxios"):
		browserName = "Mozilla Firefox"
	case strings.Contains(uaLower, "safari"):
		browserName = "Apple Safari"
	case strings.Contains(uaLower, "opera") || strings.Contains(uaLower, "opr"):
		browserName = "Opera"
	default:
		browserName = "Navegador Web"
	}
	return
}

// Journey GET /admin/tracked-users/journey?external_id=...
// Devolve a jornada unificada (cross-device/cross-browser) com histórico de todas as sessões e eventos vinculados.
func (h *AdminTrackedUsers) Journey(w http.ResponseWriter, r *http.Request) {
	externalID := strings.TrimSpace(r.URL.Query().Get("external_id"))
	if externalID == "" {
		writeError(w, http.StatusBadRequest, "external_id required")
		return
	}

	ctx := r.Context()
	scope := TenantFromContext(ctx)
	lookupAccID := int64(0)
	if !scope.IsOwner {
		lookupAccID = scope.AccountID
	}

	user, err := h.repo.FindByExternalIDByAccount(ctx, externalID, lookupAccID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "lookup error: "+err.Error())
		return
	}

	if !scope.IsOwner && user.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	sessions, err := h.repo.GetGroupSessions(ctx, scope.AccountID, externalID)
	if err != nil || len(sessions) == 0 {
		sessions = []domain.TrackedUser{*user}
	}

	extIDs := make([]string, 0, len(sessions))
	sessionSummaries := make([]SessionSummary, 0, len(sessions))
	var totalPageviews, totalCheckouts int

	for i, s := range sessions {
		extIDs = append(extIDs, s.ExternalID)
		pv := s.PageViewCount
		if pv == 0 {
			pv = 1
		}
		totalPageviews += pv
		totalCheckouts += s.CheckoutCount

		devType, browser := inferDeviceInfo(s.ClientUserAgent)
		sessionSummaries = append(sessionSummaries, SessionSummary{
			ExternalID:       s.ExternalID,
			CreatedAt:        s.CreatedAt,
			DeviceType:       devType,
			Browser:          browser,
			URL:              s.URL,
			FirstUTMSource:   s.FirstUTMSource,
			FirstUTMCampaign: s.FirstUTMCampaign,
			PageViewCount:    pv,
			CheckoutCount:    s.CheckoutCount,
			IsInitial:        i == 0,
		})
	}

	events, _, err := h.sentEvents.ListGrouped(ctx, repo.ListFilter{
		AccountID:   scope.AccountID,
		ExternalIDs: extIDs,
		Limit:       200,
	})
	if err != nil {
		slog.Warn("journey events fetch error", "err", err)
		events = []repo.GroupedEvent{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"primary_user":     user,
		"first_touch":      sessions[0],
		"sessions_count":   len(sessions),
		"sessions":         sessionSummaries,
		"total_pageviews":  totalPageviews,
		"total_checkouts":  totalCheckouts,
		"time_to_purchase": user.TimeToPurchase,
		"events":           events,
	})
}
