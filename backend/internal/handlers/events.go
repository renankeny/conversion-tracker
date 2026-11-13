package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/geoip"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// Events expõe os endpoints públicos chamados pelo pixel.js.
type Events struct {
	pixels      *repo.PixelRepo
	users       *repo.UserRepo
	sentEvents  *repo.SentEventRepo
	geo         *geoip.Lookup
	queue       *queue.Enqueuer
	urls        *util.URLMapper
	crmRepo            *repo.CRMRepo
	rdb                *redis.Client
	trackingTargetHost string
	recentDedup        sync.Map // fallback local caso o Redis esteja temporariamente indisponível
}

func NewEvents(
	p *repo.PixelRepo,
	u *repo.UserRepo,
	s *repo.SentEventRepo,
	g *geoip.Lookup,
	q *queue.Enqueuer,
	urls *util.URLMapper,
	crm *repo.CRMRepo,
) *Events {
	return &Events{pixels: p, users: u, sentEvents: s, geo: g, queue: q, urls: urls, crmRepo: crm}
}

func (h *Events) SetRedis(rdb *redis.Client) {
	h.rdb = rdb
}

func (h *Events) SetTrackingTargetHost(host string) {
	h.trackingTargetHost = strings.TrimSpace(host)
}

// sendRequest reflete o payload enviado pelo pixel.js legado.
// Os nomes de campo são intencionalmente compatíveis (mistura de camelCase e snake_case)
// para que o script `script.js` continue funcionando contra esta API sem alterações.
type sendRequest struct {
	EventType    string   `json:"eventType"`
	UserID       string   `json:"userId"`
	IDWpp        string   `json:"id_wpp"`
	PixelName    string   `json:"pixelName"`
	ContentID    string   `json:"contentId"`
	ContentName  string   `json:"content_name"`
	OrderID      string   `json:"order_id"`
	OrderIDAlt   string   `json:"orderId"`
	EventID      string   `json:"eventId"`
	Price        *float64 `json:"price"`
	Currency     string   `json:"currency"`
	URL          string   `json:"URL"`
	Referrer     string   `json:"referrer"`
	Webdriver    *bool    `json:"webdriver,omitempty"`
	FBC          string   `json:"fbc"`
	FBP          string   `json:"fbp"`
	GCLID        string   `json:"gclid"`
	WBraid       string   `json:"wbraid"`
	GBraid       string   `json:"gbraid"`
	TTClid       string   `json:"ttclid"`
	TTP          string   `json:"ttp"`
	GA4ClientID  string   `json:"ga4_client_id"`
	GA4SessionID string   `json:"ga4_session_id"`

	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	PostalCode string `json:"postal_code"`

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
