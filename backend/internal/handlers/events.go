package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/geoip"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// Events expõe os endpoints públicos chamados pelo pixel.js.
//
// Endpoint principal: POST /events/send. Recebe TODOS os tipos de evento
// (Init, UpdateUserData, PageView, Purchase, Custom...) e despacha conforme
// `eventType`. Mantemos também POST /events/init como atalho do handshake
// inicial pra clients novos que preferirem esse formato.
type Events struct {
	pixels      *repo.PixelRepo
	users       *repo.UserRepo
	sentEvents  *repo.SentEventRepo
	geo         *geoip.Lookup
	queue       *queue.Enqueuer
	urls        *util.URLMapper
}

func NewEvents(
	p *repo.PixelRepo,
	u *repo.UserRepo,
	s *repo.SentEventRepo,
	g *geoip.Lookup,
	q *queue.Enqueuer,
	urls *util.URLMapper,
) *Events {
	return &Events{pixels: p, users: u, sentEvents: s, geo: g, queue: q, urls: urls}
}

// sendRequest reflete o payload enviado pelo pixel.js legado.
// Os nomes de campo são intencionalmente compatíveis (mistura de camelCase e snake_case)
// para que o script `script.js` continue funcionando contra esta API sem alterações.
type sendRequest struct {
	EventType string   `json:"eventType"`
	UserID    string   `json:"userId"`
	IDWpp     string   `json:"id_wpp"`
	PixelName string   `json:"pixelName"`
	ContentID string   `json:"contentId"`
	EventID   string   `json:"eventId"`
	Price     *float64 `json:"price"`
	Currency  string   `json:"currency"`
	URL       string   `json:"URL"`
	Referrer  string   `json:"referrer"`
	FBC       string   `json:"fbc"`
	FBP       string   `json:"fbp"`
	GCLID     string   `json:"gclid"`
	TTClid    string   `json:"ttclid"`
	TTP       string   `json:"ttp"`

	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`

	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
	UTMContent  string `json:"utm_content"`
	UTMTerm     string `json:"utm_term"`
}

// Init é um atalho que aceita um body simples e devolve a mesma resposta
// que `Send` devolveria para `eventType=Init`. Existe pra clients novos
// que queiram um endpoint dedicado ao handshake.
func (h *Events) Init(w http.ResponseWriter, r *http.Request) {
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.EventType = domain.EventInit
	h.handle(w, r, req)
}

// Send é o endpoint principal: recebe qualquer eventType e despacha.
func (h *Events) Send(w http.ResponseWriter, r *http.Request) {
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	h.handle(w, r, req)
}

func (h *Events) handle(w http.ResponseWriter, r *http.Request, req sendRequest) {
	if req.EventType == "" {
		writeError(w, http.StatusBadRequest, "eventType required")
		return
	}
	if !domain.IsValidEventType(req.EventType) {
		writeError(w, http.StatusBadRequest, "invalid eventType: "+req.EventType)
		return
	}

	ctx := r.Context()
	ip := clientIP(r)
	ua := r.Header.Get("User-Agent")
	geo, _ := h.geo.Resolve(ip)

	// Normaliza GeoIP em ASCII lowercase (mesmo formato do legado).
	country := strings.ToLower(geo.Country)
	state := strings.ToLower(geo.State)
	city := util.NormalizeCity(geo.City)
	postal := geo.PostalCode

	// Init: só responde com o contexto. Não toca em tracked_users e não envia
	// nada pra Meta.
	if req.EventType == domain.EventInit {
		writeJSON(w, http.StatusOK, map[string]any{
			"city":              city,
			"state":             state,
			"postal_code":       postal,
			"country":           country,
			"client_ip_address": ip,
			"client_user_agent": ua,
			"fbc":               req.FBC,
			"fbp":               req.FBP,
			"external_id":       req.UserID,
		})
		return
	}

	// Daqui pra frente todos os eventos exigem userId e pixelName.
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "userId required")
		return
	}
	if req.PixelName == "" {
		writeError(w, http.StatusBadRequest, "pixelName required")
		return
	}

	// Resolve o pixel cadastrado.
	pixel, err := h.pixels.FindByName(ctx, req.PixelName)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "unknown pixelName: "+req.PixelName)
			return
		}
		slog.Error("find pixel", "err", err, "pixel_name", req.PixelName)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !pixel.Enabled {
		writeError(w, http.StatusForbidden, "pixel disabled")
		return
	}

	phone := util.NormalizePhone(req.Phone)
	email := util.NormalizeEmail(req.Email)
	mappedURL := h.urls.Apply(req.URL)

	// Persiste / atualiza o tracked user de acordo com o evento.
	if err := h.upsertTrackedUser(ctx, req, pixel.Name, pixel.AccountID, ip, ua, mappedURL, country, state, city, postal, email, phone); err != nil {
		slog.Error("upsert tracked user", "err", err, "external_id", req.UserID)
		// não interrompe o fluxo; continua e tenta enfileirar mesmo assim
	}

	// UpdateUserData não envia evento, só persiste fbp/fbc/gclid (já feito acima).
	if req.EventType == domain.EventUpdateUserData {
		writeJSON(w, http.StatusOK, map[string]any{"updated": true})
		return
	}

	// Recarrega o tracked user pra usar os dados de geo persistidos (capturados no Init,
	// quando o IP do client era o real e não de um CDN/proxy). Se não houver no banco,
	// mantém os dados do lookup atual.
	// Também recupera ttclid/ttp persistidos pra ir junto no envio TikTok mesmo
	// quando o evento atual não os trouxe (mantém atribuição).
	ttclid := req.TTClid
	ttp := req.TTP
	if existing, err := h.users.FindByExternalID(ctx, req.UserID); err == nil && existing != nil {
		if existing.Country != "" {
			country = existing.Country
		}
		if existing.State != "" {
			state = existing.State
		}
		if existing.City != "" {
			city = existing.City
		}
		if existing.PostalCode != "" {
			postal = existing.PostalCode
		}
		if ttclid == "" && existing.TTClid != "" {
			ttclid = existing.TTClid
		}
		if ttp == "" && existing.TTP != "" {
			ttp = existing.TTP
		}
	}

	// Determina o eventID: usa o que veio do client (compat com dedup do pixel.js)
	// ou gera um novo de 10 chars.
	eventID := req.EventID
	if eventID == "" {
		eventID = util.GenerateID10()
	}

	// Dedup por (event_id, pixel_name): se já enviamos com sucesso, evita reprocessar.
	// Em caso de erro no banco, NÃO bloqueia o fluxo (warn + segue).
	if exists, err := h.sentEvents.ExistsSent(ctx, eventID, pixel.Name); err != nil {
		slog.Warn("dedup check failed", "err", err, "event_id", eventID)
	} else if exists {
		writeJSON(w, http.StatusOK, map[string]any{
			"queued":       false,
			"deduplicated": true,
			"event_id":     eventID,
		})
		return
	}

	// content_id: mantém compat com o legado — usa o pixelName quando o client
	// não informa contentId (que é o caso default do script.js).
	contentID := req.ContentID
	if contentID == "" {
		contentID = pixel.Name
	}

	in := domain.SendEventInput{
		PixelName: pixel.Name,
		EventName: req.EventType,
		EventID:   eventID,
		EventTime: time.Now().Unix(),
		SourceURL: mappedURL,

		ContentID:   contentID,
		ContentType: "product",
		Value:       req.Price,
		Currency:    req.Currency,

		ExternalID:      req.UserID,
		ClientIP:        ip,
		ClientUserAgent: ua,
		FBP:             req.FBP,
		FBC:             req.FBC,
		Email:           email,
		Phone:           phone,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		Country:         country,
		State:           state,
		City:            city,
		PostalCode:      postal,

		TTClid:   ttclid,
		TTP:      ttp,
		Referrer: req.Referrer,
	}
	if err := h.queue.EnqueueSendEvent(ctx, in); err != nil {
		slog.Error("enqueue send", "err", err, "event_id", eventID)
		writeError(w, http.StatusInternalServerError, "queue error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"queued":   true,
		"event_id": eventID,
	})
}

// upsertTrackedUser cria/atualiza o tracked user de acordo com o eventType.
// PageView faz upsert completo; Lead/InitiateCheckout/Purchase atualizam PII;
// UpdateUserData atualiza só fbp/fbc/gclid; demais eventos não tocam o user.
func (h *Events) upsertTrackedUser(
	ctx context.Context,
	req sendRequest,
	pixelName string,
	accountID int64,
	ip, ua, mappedURL, country, state, city, postal, email, phone string,
) error {
	switch req.EventType {
	case domain.EventPageView, domain.EventViewContent:
		// Snapshot completo do contexto da visita.
		tu := &domain.TrackedUser{
			ExternalID:      req.UserID,
			IDWpp:           req.IDWpp,
			PixelName:       pixelName,
			AccountID:       accountID,
			FBP:             req.FBP,
			FBC:             req.FBC,
			GCLID:           req.GCLID,
			TTClid:          req.TTClid,
			TTP:             req.TTP,
			ClientIP:        ip,
			ClientUserAgent: ua,
			Country:         country,
			State:           state,
			City:            city,
			PostalCode:      postal,
			FirstName:       req.FirstName,
			LastName:        req.LastName,
			Email:           email,
			Phone:           phone,
			URL:              mappedURL,
			FirstUTMSource:   req.UTMSource,
			FirstUTMMedium:   req.UTMMedium,
			FirstUTMCampaign: req.UTMCampaign,
			FirstUTMContent:  req.UTMContent,
			FirstUTMTerm:     req.UTMTerm,
		}
		return h.users.Upsert(ctx, tu)

	case domain.EventLead, domain.EventInitiateCheckout, domain.EventPurchase,
		domain.EventContact, domain.EventCompleteRegistration, domain.EventAddPaymentInfo:
		// Atualiza PII e pixel/url, sem mexer nas UTMs nem geo.
		tu := &domain.TrackedUser{
			ExternalID:      req.UserID,
			IDWpp:           req.IDWpp,
			PixelName:       pixelName,
			AccountID:       accountID,
			FBP:             req.FBP,
			FBC:             req.FBC,
			GCLID:           req.GCLID,
			TTClid:          req.TTClid,
			TTP:             req.TTP,
			ClientIP:        ip,
			ClientUserAgent: ua,
			FirstName:       req.FirstName,
			LastName:        req.LastName,
			Email:           email,
			Phone:           phone,
			URL:             mappedURL,
		}
		return h.users.Upsert(ctx, tu)

	case domain.EventUpdateUserData:
		tu := &domain.TrackedUser{
			ExternalID: req.UserID,
			AccountID:  accountID,
			FBP:        req.FBP,
			FBC:        req.FBC,
			GCLID:      req.GCLID,
			TTClid:     req.TTClid,
			TTP:        req.TTP,
		}
		return h.users.Upsert(ctx, tu)

	default:
		// Custom events (Scroll_*, Timer_1min, ViewVideo_*, PlayVideo, AddToCart...)
		// não modificam o tracked user.
		return nil
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return real
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
