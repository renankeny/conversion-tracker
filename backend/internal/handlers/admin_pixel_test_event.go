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

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
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
	EventName     string  `json:"event_name"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	Value         float64 `json:"value"`
	Currency      string  `json:"currency"`
	Channel       string  `json:"channel"`
	TestEventCode string  `json:"test_event_code"`
}

type testEventResult struct {
	Channel      string `json:"channel"`
	PixelID      string `json:"pixel_id"`
	StatusCode   int    `json:"status_code"`
	UsedTestCode bool   `json:"used_test_code"`
	Response     string `json:"response,omitempty"`
	Error        string `json:"error,omitempty"`
}

func (h *AdminPixelTestEvent) SendTestEvent(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para disparar eventos de teste")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return
	}
	p, err := h.pixels.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pixel não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar pixel")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && p.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso a este pixel")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "este pixel está desativado; ative-o antes de testar")
		return
	}

	var in testEventInput
	_ = json.NewDecoder(r.Body).Decode(&in)
	if strings.TrimSpace(in.EventName) == "" {
		in.EventName = domain.EventLead
	}
	if !domain.IsValidEventType(in.EventName) || domain.IsInternalEvent(in.EventName) {
		writeError(w, http.StatusBadRequest, "evento inválido para teste")
		return
	}
	if in.Currency == "" {
		in.Currency = p.TrackerConfig.ProductCurrency
		if in.Currency == "" {
			in.Currency = "BRL"
		}
	}

	oauthAcc, oauthReason := h.resolveTestMetaOAuth(r.Context(), p)
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
		if oauthReason != "" {
			writeError(w, http.StatusBadRequest, oauthReason)
			return
		}
		writeError(w, http.StatusBadRequest, "pixel sem Pixel ID ou credenciais da Meta configuradas")
		return
	}
	if wantTikTok && !hasTikTok {
		writeError(w, http.StatusBadRequest, "pixel sem Pixel ID ou Access Token do TikTok configurados")
		return
	}

	input, eventID := buildTestSendInput(r, p, in)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	overrideCode := strings.TrimSpace(in.TestEventCode)
	results := make([]testEventResult, 0, len(p.PixelIDs)+len(p.TikTokPixelIDs))
	if wantMeta {
		results = append(results, h.sendTestMeta(ctx, p, oauthAcc, input, overrideCode)...)
	}
	if wantTikTok {
		results = append(results, h.sendTestTikTok(ctx, p, input, overrideCode)...)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"event_id": eventID, "event_name": in.EventName,
		"channel": channel, "results": results,
	})
}
