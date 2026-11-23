package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/webpush"
	"github.com/renankeny/conversion-tracker/backend/internal/wshub"
)

type Event struct {
	AccountID   int64
	AccountName string // opcional (útil quando a empresa acabou de ser excluída)
	Category    string
	Title       string
	Body        string
	ActionURL   string
	CooldownKey string
	CooldownTTL time.Duration
	OwnersOnly  bool
}

type pushWirePayload struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	URL         string `json:"url"`
	Category    string `json:"category"`
	AccountName string `json:"account_name,omitempty"`
	Tag         string `json:"tag"`
}

type Service struct {
	repo           *repo.NotificationRepo
	rdb            *redis.Client
	sender         *webpush.Sender
	defaultSubject string

	mu     sync.RWMutex
	cached *webpush.KeyPair
}

func NewService(r *repo.NotificationRepo, rdb *redis.Client, adminEmail, appURL string) *Service {
	sub := deriveDefaultSubject(adminEmail, appURL)
	return &Service{
		repo:           r,
		rdb:            rdb,
		sender:         webpush.NewSender(),
		defaultSubject: sub,
	}
}

func deriveDefaultSubject(adminEmail, appURL string) string {
	if e := strings.TrimSpace(adminEmail); strings.Contains(e, "@") {
		return "mailto:" + e
	}
	if u, err := url.Parse(strings.TrimSpace(appURL)); err == nil && u.Hostname() != "" && u.Hostname() != "localhost" {
		return "mailto:notificacoes@" + u.Hostname()
	}
	return "mailto:notificacoes@sistema.local"
}

func (s *Service) EnsureKeys(ctx context.Context) (*webpush.KeyPair, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("notification service unavailable")
	}
	s.mu.RLock()
	if s.cached != nil {
		kp := s.cached
		s.mu.RUnlock()
		return kp, nil
	}
	s.mu.RUnlock()

	rec, err := s.repo.GetOrCreateVAPID(ctx, func() (string, string, string, error) {
		kp, gerr := webpush.GenerateVAPIDKeys(s.defaultSubject)
		if gerr != nil {
			return "", "", "", gerr
		}
		return kp.PublicKey, kp.PrivateKeyB64, kp.Subject, nil
	})
	if err != nil {
		return nil, err
	}
	kp, err := webpush.ParseKeyPair(rec.PublicKey, rec.PrivateKey, rec.Subject)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cached = kp
	s.mu.Unlock()
	return kp, nil
}

func (s *Service) PublicKey(ctx context.Context) string {
	kp, err := s.EnsureKeys(ctx)
	if err != nil || kp == nil {
		return ""
	}
	return kp.PublicKey
}

func (s *Service) DispatchAsync(ev Event) {
	if s == nil || s.repo == nil || ev.Category == "" || ev.Title == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		s.Dispatch(ctx, ev)
	}()
}

func (s *Service) Dispatch(ctx context.Context, ev Event) {
	if s == nil || s.repo == nil || ev.Category == "" || ev.Title == "" {
		return
	}
	if ev.CooldownKey != "" && s.rdb != nil {
		ttl := ev.CooldownTTL
		if ttl <= 0 {
			ttl = 10 * time.Minute
		}
		ok, err := s.rdb.SetNX(ctx, "tf:notif:cd:"+ev.CooldownKey, "1", ttl).Result()
		if err == nil && !ok {
			return
		}
	}

	recipients, err := s.repo.ListRecipientsForAccount(ctx, ev.AccountID, ev.OwnersOnly)
	if err != nil || len(recipients) == 0 {
		return
	}

	kp, _ := s.EnsureKeys(ctx)
	for _, rec := range recipients {
		if !rec.Prefs.AllowsCategory(ev.Category) {
			continue
		}
		if !rec.Prefs.AllowsAccount(ev.AccountID, ev.Category) {
			continue
		}
		accName := strings.TrimSpace(ev.AccountName)
		if accName == "" {
			accName = rec.AccountName
		}
		s.deliverToUser(ctx, kp, rec.UserID, ev.AccountID, accName, ev.Category, ev.Title, ev.Body, ev.ActionURL)
	}
}

func (s *Service) deliverToUser(
	ctx context.Context,
	kp *webpush.KeyPair,
	userID, accountID int64,
	accountName, category, title, body, actionURL string,
) *domain.UserNotification {
	if actionURL == "" {
		actionURL = "/"
	}
	var accPtr *int64
	if accountID > 0 {
		idCopy := accountID
		accPtr = &idCopy
	}
	fullTitle := title
	if accountName != "" && !strings.Contains(title, accountName) {
		fullTitle = "[" + accountName + "] " + title
	}
	item := &domain.UserNotification{
		UserID:      userID,
		AccountID:   accPtr,
		AccountName: accountName,
		Category:    category,
		Title:       fullTitle,
		Body:        body,
		ActionURL:   actionURL,
	}
	if err := s.repo.InsertWithMax10(ctx, item); err != nil {
		slog.Warn("notify: insert user notification failed", "user_id", userID, "category", category, "error", err)
		return nil
	}
	wshub.PublishToUser(ctx, s.rdb, userID, "notification_created", item)

	if kp == nil {
		return item
	}
	subs, err := s.repo.ListSubscriptionsByUser(ctx, userID)
	if err != nil || len(subs) == 0 {
		return item
	}
	wireBytes, _ := json.Marshal(pushWirePayload{
		Title:       fullTitle,
		Body:        body,
		URL:         actionURL,
		Category:    category,
		AccountName: accountName,
		Tag:         fmt.Sprintf("tf-%s-%d", category, item.ID),
	})
	for _, sub := range subs {
		gone, sendErr := s.sender.Send(ctx, kp, webpush.SubscriptionTarget{
			Endpoint:   sub.Endpoint,
			P256dh:     sub.P256dh,
			AuthSecret: sub.AuthSecret,
		}, wireBytes)
		if gone {
			s.repo.DeleteSubscriptionByEndpoint(ctx, sub.Endpoint)
		} else if sendErr != nil {
			slog.Warn("notify: webpush send error", "user_id", userID, "sub_id", sub.ID, "error", sendErr)
		}
	}
	return item
}
