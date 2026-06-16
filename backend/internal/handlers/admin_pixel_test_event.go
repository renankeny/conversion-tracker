package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/meta"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/tiktok"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// AdminPixelTestEvent dispara um evento de teste sincronamente pra Meta CAPI
// e/ou TikTok Events API usando os test_event_codes configurados no pixel.
// Útil pra validar a integração pelo painel sem precisar abrir o site real.
type AdminPixelTestEvent struct {
	pixels         *repo.PixelRepo
	metaAPIVersion string
	metaPartner    string
}

func NewAdminPixelTestEvent(pixels *repo.PixelRepo, apiVersion, partnerAgent string) *AdminPixelTestEvent {
	return &AdminPixelTestEvent{
		pixels:         pixels,
		metaAPIVersion: apiVersion,
		metaPartner:    partnerAgent,
	}
}

type testEventInput struct {
	EventName string  `json:"event_name"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Value     float64 `json:"value"`
	Currency  string  `json:"currency"`
	// Channel = "meta" (default), "tiktok" ou "all"
	Channel string `json:"channel"`
}

type testEventResult struct {
	Channel    string `json:"channel"` // "meta" ou "tiktok"
	PixelID    string `json:"pixel_id"`
	StatusCode int    `json:"status_code"`
	Response   string `json:"response,omitempty"`
	Error      string `json:"error,omitempty"`
}

// SendTestEvent dispara o evento sincronamente em todos os pixel_ids do pixel
// e retorna o resultado de cada um (status, body, erro).
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

	// Body opcional: usuário pode customizar evento e PII.
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

	channel := strings.ToLower(strings.TrimSpace(in.Channel))
	if channel == "" {
		channel = "meta"
	}
	wantMeta := channel == "meta" || channel == "all"
	wantTikTok := channel == "tiktok" || channel == "all"

	if wantMeta && (len(p.PixelIDs) == 0 || len(p.AccessTokens) == 0) {
		writeError(w, http.StatusBadRequest, "pixel sem pixel_ids ou access_tokens Meta configurados")
		return
	}
	if wantTikTok && !p.HasTikTok() {
		writeError(w, http.StatusBadRequest, "pixel sem tiktok_pixel_ids ou tiktok_access_tokens configurados")
		return
	}

	// Monta o payload com test_event_code e dados de PII normalizados.
	eventID := "TEST_" + util.GenerateID10()
	now := time.Now().Unix()

	// Resolve IP do cliente. Prioridade: X-Forwarded-For (atrás de proxy)
	// > X-Real-IP > RemoteAddr. Em todos os casos, descarta a porta (TikTok
	// e Meta esperam só o endereco IP, sem :porta).
	clientIP := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(r.Header.Get("X-Real-IP"))
	}
	if clientIP == "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			clientIP = host
		} else {
			clientIP = r.RemoteAddr
		}
	}
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
		PixelName:       p.Name,
		EventName:       in.EventName,
		EventID:         eventID,
		EventTime:       now,
		SourceURL:       sourceURL,
		ContentID:       p.Name, // mesmo padrao do /events/send: pixelName como content_id quando o cliente nao informa
		ContentName:     p.Name,
		ContentType:     "product",
		Value:           valuePtr,
		Currency:        in.Currency,
		ExternalID:      "test-" + util.GenerateID10(),
		ClientIP:        clientIP,
		ClientUserAgent: clientUA,
		Email:           in.Email,
		Phone:           in.Phone,
		FirstName:       in.FirstName,
		LastName:        in.LastName,
	}

	// Envia sincronamente pra cada pixel_id usando os clientes correspondentes.
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	results := make([]testEventResult, 0, len(p.PixelIDs)+len(p.TikTokPixelIDs))

	if wantMeta {
		client := meta.New(h.metaAPIVersion, h.metaPartner)
		for _, r := range client.Send(ctx, p, input) {
			res := testEventResult{
				Channel:    "meta",
				PixelID:    r.PixelID,
				StatusCode: r.StatusCode,
				Response:   util.RedactSecrets(r.Body),
			}
			if r.Err != nil {
				res.Error = r.Err.Error()
			}
			results = append(results, res)
		}
	}

	if wantTikTok {
		client := tiktok.New(h.metaPartner)
		for _, r := range client.Send(ctx, p, input) {
			res := testEventResult{
				Channel:    "tiktok",
				PixelID:    r.PixelID,
				StatusCode: r.StatusCode,
				Response:   util.RedactSecrets(r.Body),
			}
			if r.Err != nil {
				res.Error = r.Err.Error()
			}
			results = append(results, res)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"event_id":   eventID,
		"event_name": in.EventName,
		"channel":    channel,
		"results":    results,
	})
}
