package repo

import (
	"context"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
)

func (r *PixelRepo) BumpDashboardCache(ctx context.Context, accountID int64) {
	if r == nil || accountID <= 0 {
		return
	}
	cache.BumpDashboardVersion(ctx, r.rdb, accountID)
}

func (r *PixelRepo) InvalidateByMetaAccount(ctx context.Context, metaAccountID int64) {
	if r == nil || r.pool == nil || metaAccountID <= 0 {
		return
	}
	r.invalidatePixelsWhere(ctx, `SELECT name, public_token, dashboard_share_token FROM pixels WHERE meta_account_id = $1`, metaAccountID)
}

func (r *PixelRepo) InvalidateByGoogleAdsAccount(ctx context.Context, googleAdsAccountID int64) {
	if r == nil || r.pool == nil || googleAdsAccountID <= 0 {
		return
	}
	r.invalidatePixelsWhere(ctx, `SELECT name, public_token, dashboard_share_token FROM pixels WHERE google_ads_account_id = $1`, googleAdsAccountID)
}

func (r *PixelRepo) invalidatePixelsWhere(ctx context.Context, query string, arg int64) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var name, pubTok, shareTok string
		if err := rows.Scan(&name, &pubTok, &shareTok); err != nil {
			continue
		}
		if s := strings.TrimSpace(name); s != "" {
			keys = append(keys, s)
		}
		if s := strings.TrimSpace(pubTok); s != "" {
			keys = append(keys, s)
		}
		if s := strings.TrimSpace(shareTok); s != "" {
			keys = append(keys, s)
		}
	}
	if len(keys) > 0 {
		r.InvalidateCache(ctx, keys...)
	}
}
