package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/tiktok"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type AdminPixelTestEvent struct {
	pixels         *repo.PixelRepo
	metaAccs       *repo.MetaAccountRepo
	metaOAuth      *metaoauth.Client
	metaAPIVersion string
	metaPartner    string
	tiktokPartner  string
}

func NewAdminPixelTestEvent(pixels *repo.PixelRepo, apiVersion, metaPartnerAgent, tiktokPartnerName string) *AdminPixelTestEvent {
	return &AdminPixelTestEvent{
		pixels:         pixels,
		metaAPIVersion: apiVersion,
		metaPartner:    metaPartnerAgent,
		tiktokPartner:  tiktokPartnerName,
	}
}

func (h *AdminPixelTestEvent) SetMetaOAuth(metaAccs *repo.MetaAccountRepo, metaOAuth *metaoauth.Client) {
	h.metaAccs = metaAccs
	h.metaOAuth = metaOAuth
}

type testEventInput struct {
	EventName string  `json:"event_name"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Value     float64 `json:"value"`
	Currency  string  `json:"currency"`
	Channel   string  `json:"channel"`
}

type testEventResult struct {
	Channel    string `json:"channel"`
	PixelID    string `json:"pixel_id"`
	StatusCode int    `json:"status_code"`
	Response   string `json:"response,omitempty"`
	Error      string `json:"error,omitempty"`
}

func (h *AdminPixelTestEvent) SendTestEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.pixels.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "pixel disabled")
		return
	}

	var in testEventInput
	_ = json.NewDecoder(r.Body).Decode(&in)
	if strings.TrimSpace(in.EventName) == "" {
		in.EventName = domain.EventLead
	}
	if !domain.IsValidEventType(in.EventName) || domain.IsInternalEvent(in.EventName) {
		writeError(w, http.StatusBadRequest, "evento invalido para teste")
		return
	}
	if in.Currency == "" {
		in.Currency = p.TrackerConfig.ProductCurrency
		if in.Currency == "" {
			in.Currency = "BRL"
		}
	}

	oauthAcc := h.resolveTestMetaOAuth(r.Context(), p)
	hasMeta := len(p.PixelIDs) > 0 && (hasNonEmptyToken(p.AccessTokens) || oauthAcc != nil)
	hasTikTok := p.HasTikTok()
	channel := strings.ToLower(strings.TrimSpace(in.Channel))
	if channel == "" {
		if hasMeta {
			channel = "meta"
		} else if hasTikTok {
			channel = "tiktok"
		} else {
			channel = "meta"
		}
	}
	wantMeta := channel == "meta" || (channel == "all" && hasMeta)
	wantTikTok := channel == "tiktok" || (channel == "all" && hasTikTok)

	if !wantMeta && !wantTikTok {
		writeError(w, http.StatusBadRequest, "pixel sem credenciais Meta ou TikTok configuradas para teste")
		return
	}
	if wantMeta && !hasMeta {
		writeError(w, http.StatusBadRequest, "pixel sem pixel_ids ou credenciais Meta configuradas")
		return
	}
	if wantTikTok && !hasTikTok {
		writeError(w, http.StatusBadRequest, "pixel sem tiktok_pixel_ids ou tiktok_access_tokens configurados")
		return
	}

	eventID := "TEST_" + util.GenerateID10()
	clientUA := r.UserAgent()
	if clientUA == "" {
		clientUA = "TrackFind-TestEvent/1.0"
	}
	var valuePtr *float64
	if in.Value > 0 {
		v := in.Value
		valuePtr = &v
	}
	sourceURL := "https://" + r.Host + "/admin/test-event"
	if p.TrackingHost != "" {
		sourceURL = "https://" + p.TrackingHost + "/admin/test-event"
	}

	input := domain.SendEventInput{
		PixelName: p.Name, EventName: in.EventName, EventID: eventID,
		EventTime: time.Now().Unix(), SourceURL: sourceURL,
		ContentID: p.Name, ContentName: p.Name, ContentType: "product",
		Value: valuePtr, Currency: in.Currency,
		ExternalID: "test-" + util.GenerateID10(), ClientIP: extractTestClientIP(r),
		ClientUserAgent: clientUA, Email: in.Email, Phone: in.Phone,
		FirstName: in.FirstName, LastName: in.LastName,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	results := make([]testEventResult, 0, len(p.PixelIDs)+len(p.TikTokPixelIDs))
	if wantMeta {
		results = append(results, h.sendTestMeta(ctx, p, oauthAcc, input)...)
	}
	if wantTikTok {
		client := tiktok.New(h.tiktokPartner)
		for _, r := range client.Send(ctx, p, input) {
			errStr := ""
			if r.Err != nil {
				errStr = r.Err.Error()
			}
			results = append(results, testEventResult{
				Channel: "tiktok", PixelID: r.PixelID, StatusCode: r.StatusCode,
				Response: util.RedactSecrets(r.Body), Error: errStr,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"event_id": eventID, "event_name": in.EventName,
		"channel": channel, "results": results,
	})
}
