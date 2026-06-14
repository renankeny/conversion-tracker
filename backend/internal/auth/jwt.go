package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey int

const claimsKey ctxKey = iota

func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func ClaimsFrom(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

type Claims struct {
	UserID    int64  `json:"uid"`
	Email     string `json:"email"`
	Role      string `json:"role,omitempty"`        // "owner" ou "member"
	AccountID int64  `json:"account_id,omitempty"`  // 0 para owners (acesso global)
	jwt.RegisteredClaims
}

// IsOwner retorna true se o admin é "owner" (acesso global a todas as contas).
func (c *Claims) IsOwner() bool {
	return c != nil && (c.Role == "" || c.Role == "owner")
}

// HasAccountAccess retorna true se o admin pode acessar a conta especificada.
// Owners podem acessar qualquer conta. Members só a sua.
func (c *Claims) HasAccountAccess(accountID int64) bool {
	if c == nil {
		return false
	}
	if c.IsOwner() {
		return true
	}
	return c.AccountID == accountID
}

type JWT struct {
	secret []byte
	expHrs int
}

func NewJWT(secret string, expHrs int) *JWT {
	return &JWT{secret: []byte(secret), expHrs: expHrs}
}

// Issue assina um JWT incluindo role e account_id (multi-tenant).
// role="" e accountID=0 mantém compatibilidade com tokens antigos (tratados como owner global).
func (j *JWT) Issue(userID int64, email, role string, accountID int64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		AccountID: accountID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(j.expHrs) * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(j.secret)
}

func (j *JWT) Parse(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
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
