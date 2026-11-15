package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

const (
	wsWriteWait  = 10 * time.Second
	wsPongWait   = 60 * time.Second
	wsPingPeriod = (wsPongWait * 9) / 10
	wsMaxMsgSize = 512
)

// WSEvents expõe o endpoint WebSocket de eventos em tempo real.
type WSEvents struct {
	hub       *wshub.Hub
	jwt       *auth.JWT
	tickets   *cache.WSTicketStore
	sessions  *cache.SessionStore
	appOrigin string
	isProd    bool
}

func NewWSEvents(hub *wshub.Hub, j *auth.JWT) *WSEvents {
	return &WSEvents{hub: hub, jwt: j}
}

func (h *WSEvents) SetSessionDeps(t *cache.WSTicketStore, s *cache.SessionStore) {
	h.tickets = t
	h.sessions = s
}

func (h *WSEvents) SetOriginPolicy(appURL string, isProd bool) {
	if u, err := url.Parse(strings.TrimSpace(appURL)); err == nil && u.Scheme != "" && u.Host != "" {
		h.appOrigin = strings.ToLower(u.Scheme + "://" + u.Host)
	}
	h.isProd = isProd
}

func (h *WSEvents) checkOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	normalized := strings.ToLower(u.Scheme + "://" + u.Host)
	if h.appOrigin != "" && normalized == h.appOrigin {
		return true
	}
	if !h.isProd && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
		return true
	}
	return false
}

// Handle faz upgrade HTTP para WebSocket, autentica e registra o cliente no hub.
func (h *WSEvents) Handle(w http.ResponseWriter, r *http.Request) {
	var (
		userID    int64
		accountID int64
		isOwner   bool
		tokenID   string
		issuedAt  int64
	)

	if h.tickets != nil {
		ticketID := strings.TrimSpace(r.URL.Query().Get("ticket"))
		if ticketID == "" {
			http.Error(w, "ticket obrigatório", http.StatusUnauthorized)
			return
		}
		t, ok := h.tickets.Consume(r.Context(), ticketID)
		if !ok {
			http.Error(w, "ticket inválido ou expirado", http.StatusUnauthorized)
			return
		}
		if h.sessions != nil && h.sessions.IsRevoked(r.Context(), t.TokenID, t.UserID, t.IssuedAt) {
			http.Error(w, "sessão revogada", http.StatusUnauthorized)
			return
		}
		userID = t.UserID
		accountID = t.AccountID
		isOwner = t.IsOwner
		tokenID = t.TokenID
		issuedAt = t.IssuedAt
	} else {
		tok := strings.TrimSpace(r.URL.Query().Get("token"))
		if tok == "" || h.jwt == nil {
			http.Error(w, "credencial obrigatória", http.StatusUnauthorized)
			return
		}
		claims, err := h.jwt.Parse(tok)
		if err != nil || claims.IsAPIKey() {
			http.Error(w, "token inválido", http.StatusUnauthorized)
			return
		}
		if claims.IssuedAt != nil {
			issuedAt = claims.IssuedAt.Unix()
		}
		if h.sessions != nil && h.sessions.IsRevoked(r.Context(), claims.ID, claims.UserID, issuedAt) {
			http.Error(w, "sessão revogada", http.StatusUnauthorized)
			return
		}
		userID = claims.UserID
		accountID = claims.AccountID
		isOwner = claims.IsOwner()
		tokenID = claims.ID
	}

	if !isOwner && accountID <= 0 {
		accountID = -1
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin:     h.checkOrigin,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("ws: upgrade failed", "err", err)
		return
	}

	client := &wshub.Client{
		UserID:      userID,
		AccountID:   accountID,
		IsOwner:     isOwner,
		TokenID:     tokenID,
		IssuedAt:    issuedAt,
		PixelFilter: strings.TrimSpace(r.URL.Query().Get("pixel")),
		Send:        make(chan []byte, 256),
	}

	h.hub.Register(client)
	go wsWritePump(conn, client, h.sessions)
	wsReadPump(conn, client, h.hub)
}

func wsReadPump(conn *websocket.Conn, client *wshub.Client, hub *wshub.Hub) {
	defer func() {
		hub.Unregister(client)
		conn.Close()
	}()

	conn.SetReadLimit(wsMaxMsgSize)
	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func wsWritePump(conn *websocket.Conn, client *wshub.Client, sessions *cache.SessionStore) {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case msg, ok := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			if sessions != nil {
				checkCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				revoked := sessions.IsRevoked(checkCtx, client.TokenID, client.UserID, client.IssuedAt)
				cancel()
				if revoked {
					_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
					_ = conn.WriteMessage(
						websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "sessão encerrada"),
					)
					return
				}
			}
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
