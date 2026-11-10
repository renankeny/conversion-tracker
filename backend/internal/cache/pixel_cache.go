package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const (
	pixelCacheTTL   = 10 * time.Minute
	trackerCacheTTL = 10 * time.Minute
)

type CachedTrackerScript struct {
	Body string `json:"body"`
	ETag string `json:"etag"`
}

// PixelCache armazena configurações de Pixel (criptografadas em repouso com AES-GCM)
// e scripts renderizados de tracker.js com ETag determinístico no Redis.
type PixelCache struct {
	rdb *redis.Client
	sf  singleflight.Group
}

func NewPixelCache(rdb *redis.Client) *PixelCache {
	if rdb == nil {
		return nil
	}
	return &PixelCache{rdb: rdb}
}

func pixelCfgKey(nameOrToken string) string {
	return "pixel:cfg:" + strings.TrimSpace(nameOrToken)
}

func pixelShareKey(shareToken string) string {
	return "pixel:share:" + strings.TrimSpace(shareToken)
}

func trackerJSKey(slugOrToken string) string {
	return "tracker:js:" + strings.TrimSpace(slugOrToken)
}

func (c *PixelCache) GetOrLoadPixel(ctx context.Context, key string, loader func(context.Context) (*domain.Pixel, error)) (*domain.Pixel, error) {
	if c == nil || c.rdb == nil || strings.TrimSpace(key) == "" {
		return loader(ctx)
	}
	rKey := pixelCfgKey(key)
	if p, ok := c.readPixel(ctx, rKey); ok {
		return p, nil
	}
	return c.loadSingleflight(ctx, rKey, pixelCacheTTL, loader)
}

func (c *PixelCache) GetOrLoadSharePixel(ctx context.Context, shareToken string, loader func(context.Context) (*domain.Pixel, error)) (*domain.Pixel, error) {
	if c == nil || c.rdb == nil || strings.TrimSpace(shareToken) == "" {
		return loader(ctx)
	}
	rKey := pixelShareKey(shareToken)
	if p, ok := c.readPixel(ctx, rKey); ok {
		return p, nil
	}
	return c.loadSingleflight(ctx, rKey, 5*time.Minute, loader)
}

func (c *PixelCache) loadSingleflight(ctx context.Context, rKey string, ttl time.Duration, loader func(context.Context) (*domain.Pixel, error)) (*domain.Pixel, error) {
	val, err, _ := c.sf.Do(rKey, func() (any, error) {
		loadCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p, lerr := loader(loadCtx)
		if lerr != nil {
			return nil, lerr
		}
		c.writePixel(loadCtx, rKey, p, ttl)
		return p, nil
	})
	if err != nil {
		return nil, err
	}
	orig, _ := val.(*domain.Pixel)
	if orig == nil {
		return loader(ctx)
	}
	clone := *orig
	return &clone, nil
}

func (c *PixelCache) readPixel(ctx context.Context, rKey string) (*domain.Pixel, bool) {
	rawEnc, err := c.rdb.Get(ctx, rKey).Result()
	if err != nil || rawEnc == "" {
		return nil, false
	}
	plain := util.Decrypt(rawEnc)
	var p domain.Pixel
	if err := json.Unmarshal([]byte(plain), &p); err != nil {
		return nil, false
	}
	return &p, true
}

func (c *PixelCache) writePixel(ctx context.Context, rKey string, p *domain.Pixel, ttl time.Duration) {
	if p == nil {
		return
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return
	}
	enc := util.Encrypt(string(raw))
	_ = c.rdb.Set(ctx, rKey, enc, ttl).Err()
}

func (c *PixelCache) GetTrackerScript(ctx context.Context, slugOrToken string) (CachedTrackerScript, bool) {
	if c == nil || c.rdb == nil || strings.TrimSpace(slugOrToken) == "" {
		return CachedTrackerScript{}, false
	}
	raw, err := c.rdb.Get(ctx, trackerJSKey(slugOrToken)).Bytes()
	if err != nil || len(raw) == 0 {
		return CachedTrackerScript{}, false
	}
	var out CachedTrackerScript
	if err := json.Unmarshal(raw, &out); err != nil || out.Body == "" {
		return CachedTrackerScript{}, false
	}
	return out, true
}

func (c *PixelCache) SetTrackerScript(ctx context.Context, body string, keys ...string) CachedTrackerScript {
	sum := sha256.Sum256([]byte(body))
	script := CachedTrackerScript{
		Body: body,
		ETag: `"` + hex.EncodeToString(sum[:16]) + `"`,
	}
	if c == nil || c.rdb == nil {
		return script
	}
	raw, err := json.Marshal(script)
	if err != nil {
		return script
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			_ = c.rdb.Set(ctx, trackerJSKey(k), raw, trackerCacheTTL).Err()
		}
	}
	return script
}

// InvalidateKeys remove imediatamente chaves de config, share link e tracker.js associadas ao pixel.
func (c *PixelCache) InvalidateKeys(ctx context.Context, tokensOrSlugs ...string) {
	if c == nil || c.rdb == nil {
		return
	}
	delKeys := make([]string, 0, len(tokensOrSlugs)*3)
	for _, tok := range tokensOrSlugs {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		delKeys = append(delKeys, pixelCfgKey(tok), pixelShareKey(tok), trackerJSKey(tok))
	}
	if len(delKeys) > 0 {
		_ = c.rdb.Del(ctx, delKeys...).Err()
	}
}

