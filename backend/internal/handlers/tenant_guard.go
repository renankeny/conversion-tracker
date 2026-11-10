package handlers

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

var errForbiddenAccount = errors.New("sem permissão para acessar ou vincular recursos a esta empresa")

var safeImageDataPrefixes = []string{
	"data:image/png;base64,",
	"data:image/jpeg;base64,",
	"data:image/jpg;base64,",
	"data:image/webp;base64,",
	"data:image/gif;base64,",
	"data:image/x-icon;base64,",
	"data:image/vnd.microsoft.icon;base64,",
}

// CanAccessAccount valida se as claims possuem posse/acesso sobre targetAccountID.
func CanAccessAccount(ctx context.Context, pool *pgxpool.Pool, c *auth.Claims, targetAccountID int64) bool {
	if c == nil || targetAccountID <= 0 {
		return false
	}
	if c.IsAPIKey() {
		return c.AccountID == targetAccountID
	}
	if pool == nil {
		return c.IsOwner() || c.AccountID == targetAccountID
	}
	if c.IsOwner() {
		var exists bool
		_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1)`, targetAccountID).Scan(&exists)
		return exists
	}
	if c.Role != "admin" && c.Role != "manager" {
		return c.AccountID == targetAccountID
	}
	var allowed bool
	const q = `SELECT EXISTS(
		SELECT 1 FROM accounts
		WHERE id = $1 AND (
			owner_id = $2
			OR owner_id IN (SELECT id FROM admin_users WHERE parent_id = $2)
			OR id = (SELECT COALESCE(account_id, 0) FROM admin_users WHERE id = $2)
		)
	)`
	_ = pool.QueryRow(ctx, q, targetAccountID, c.UserID).Scan(&allowed)
	return allowed
}

// ResolveTargetAccountID determina o account_id efetivo em criações/transferências,
// validando posse estrita quando override é informado.
func ResolveTargetAccountID(ctx context.Context, pool *pgxpool.Pool, override *int64) (int64, error) {
	c, ok := auth.ClaimsFrom(ctx)
	if !ok || c == nil {
		return 0, errForbiddenAccount
	}
	if override != nil && *override > 0 {
		if !CanAccessAccount(ctx, pool, c, *override) {
			return 0, errForbiddenAccount
		}
		return *override, nil
	}
	scope := TenantFromContext(ctx)
	if scope.AccountID > 0 {
		return scope.AccountID, nil
	}
	if scope.IsOwner {
		return 1, nil
	}
	return 0, errForbiddenAccount
}

// RequireAccountPixel garante que o pixel informado existe e pertence à conta alvo.
func RequireAccountPixel(ctx context.Context, pixels *repo.PixelRepo, accountID int64, pixelName string) (*domain.Pixel, error) {
	if pixels == nil || strings.TrimSpace(pixelName) == "" {
		return nil, errors.New("pixel inválido")
	}
	p, err := pixels.FindByName(ctx, strings.TrimSpace(pixelName))
	if err != nil {
		return nil, errors.New("pixel não encontrado")
	}
	if accountID > 0 && p.AccountID != accountID {
		return nil, errors.New("o pixel informado não pertence a esta empresa")
	}
	return p, nil
}

// RequireAccountGoogleAds garante que a conta Google Ads vinculada pertence à mesma empresa.
func RequireAccountGoogleAds(ctx context.Context, gads *repo.GoogleAdsAccountRepo, accountID int64, gadsID *int64) error {
	if gads == nil || gadsID == nil || *gadsID <= 0 {
		return nil
	}
	acc, err := gads.FindByID(ctx, *gadsID)
	if err != nil {
		return errors.New("conta Google Ads não encontrada")
	}
	if accountID > 0 && acc.AccountID != accountID {
		return errors.New("a conta Google Ads selecionada pertence a outra empresa")
	}
	return nil
}

// RequireAccountMetaAds garante que a conta Meta Ads OAuth vinculada pertence à mesma empresa.
func RequireAccountMetaAds(ctx context.Context, metaRepo *repo.MetaAccountRepo, accountID int64, metaID *int64) error {
	if metaRepo == nil || metaID == nil || *metaID <= 0 {
		return nil
	}
	acc, err := metaRepo.FindByID(ctx, *metaID)
	if err != nil {
		return errors.New("conta Meta Ads não encontrada")
	}
	if accountID > 0 && acc.AccountID != accountID {
		return errors.New("a conta Meta Ads selecionada pertence a outra empresa")
	}
	return nil
}

// SafeLogoOrFaviconURL bloqueia javascript:, data:text/html e SVGs com script em logos/favicons.
func SafeLogoOrFaviconURL(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return true
	}
	lower := strings.ToLower(s)
	for _, prefix := range safeImageDataPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Scheme, "https") {
		return true
	}
	if strings.EqualFold(u.Scheme, "http") {
		h := strings.ToLower(u.Hostname())
		return h == "localhost" || h == "127.0.0.1"
	}
	return false
}

