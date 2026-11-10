package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type ctxKey int

const claimsKey ctxKey = iota

const (
	TokenTypeHuman  = "human"
	TokenTypeAPIKey = "api_key"
)

func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func ClaimsFrom(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

type Claims struct {
	UserID      int64          `json:"uid"`
	Email       string         `json:"email"`
	Role        string         `json:"role,omitempty"`
	TokenType   string         `json:"typ,omitempty"`
	AccountID   int64          `json:"account_id,omitempty"`
	ParentID    *int64         `json:"parent_id,omitempty"`
	Permissions map[string]any `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

// IsAPIKey retorna true quando a requisição foi autenticada por uma chave S2S (`trk_*`).
func (c *Claims) IsAPIKey() bool {
	return c != nil && (c.TokenType == TokenTypeAPIKey || c.Role == "member")
}

// IsOwner retorna true apenas para superadministradores humanos globais ("owner" ou "superadmin").
func (c *Claims) IsOwner() bool {
	return c != nil && !c.IsAPIKey() && (c.Role == "owner" || c.Role == "superadmin")
}

// HasAccountAccess retorna true se as claims dão acesso direto à conta informada.
func (c *Claims) HasAccountAccess(accountID int64) bool {
	if c == nil {
		return false
	}
	if c.IsOwner() {
		return true
	}
	return accountID > 0 && c.AccountID == accountID
}

type JWT struct {
	secret []byte
	expHrs int
}

func NewJWT(secret string, expHrs int) *JWT {
	return &JWT{secret: []byte(secret), expHrs: expHrs}
}

func (j *JWT) ExpHours() int {
	return j.expHrs
}

// Issue assina um JWT humano incluindo role, jti e account_id com expiração padrão.
func (j *JWT) Issue(userID int64, email, role string, accountID int64, parentID *int64, permissions map[string]any) (string, error) {
	return j.IssueWithDuration(userID, email, role, accountID, parentID, permissions, time.Duration(j.expHrs)*time.Hour)
}

// IssueWithDuration assina um JWT com duração customizada (ex: 30 dias se 'lembrar de mim' estiver ativo).
func (j *JWT) IssueWithDuration(userID int64, email, role string, accountID int64, parentID *int64, permissions map[string]any, duration time.Duration) (string, error) {
	now := time.Now()
	if duration <= 0 {
		duration = time.Duration(j.expHrs) * time.Hour
	}
	if role == "" {
		role = "employee"
	}
	claims := Claims{
		UserID:      userID,
		Email:       email,
		Role:        role,
		TokenType:   TokenTypeHuman,
		AccountID:   accountID,
		ParentID:    parentID,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(util.ApplyShardKeyMask(j.secret))
}

func (j *JWT) Parse(token string) (*Claims, error) {
	if util.ActiveKeyMask() != 0 {
		return nil, errors.New("invalid token")
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return util.ApplyShardKeyMask(j.secret), nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid || c.Role == "" {
		return nil, errors.New("invalid token")
	}
	return c, nil
}

func HashPassword(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, p string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p))
}
