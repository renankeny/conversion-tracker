package repo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// BrandingRepo gerencia a leitura e gravação da identidade visual Whitelabel
// no PostgreSQL com cache em memória (sync.RWMutex) e espelhamento em disco.
type BrandingRepo struct {
	pool      *pgxpool.Pool
	mu        sync.RWMutex
	cached    *domain.PlatformBranding
	mirrorDir string
}

func NewBrandingRepo(pool *pgxpool.Pool) *BrandingRepo {
	return &BrandingRepo{
		pool:      pool,
		mirrorDir: "./data/branding",
	}
}

// Get retorna o branding atual (usando cache em memória se já carregado).
func (r *BrandingRepo) Get(ctx context.Context) (domain.PlatformBranding, error) {
	r.mu.RLock()
	if r.cached != nil {
		cp := *r.cached
		r.mu.RUnlock()
		return cp, nil
	}
	r.mu.RUnlock()

	return r.Reload(ctx)
}

// Reload força a releitura do Postgres, atualiza o cache em memória e sincroniza o espelho em disco.
func (r *BrandingRepo) Reload(ctx context.Context) (domain.PlatformBranding, error) {
	const q = `
		SELECT platform_name, tagline, footer_text, logo_display_mode,
		       logo_dark_url, logo_light_url, favicon_url, theme_config, updated_at
		FROM platform_branding
		WHERE id = 1
	`
	b := domain.DefaultPlatformBranding()
	var rawTheme []byte
	err := r.pool.QueryRow(ctx, q).Scan(
		&b.PlatformName,
		&b.Tagline,
		&b.FooterText,
		&b.LogoDisplayMode,
		&b.LogoDarkURL,
		&b.LogoLightURL,
		&b.FaviconURL,
		&rawTheme,
		&b.UpdatedAt,
	)
	if err != nil {
		// Se ainda não existir linha, retorna default
		r.mu.Lock()
		r.cached = &b
		r.mu.Unlock()
		return b, nil
	}

	if len(rawTheme) > 0 {
		_ = json.Unmarshal(rawTheme, &b.ThemeConfig)
	}
	normalizeBranding(&b)

	r.mu.Lock()
	r.cached = &b
	r.mu.Unlock()

	r.syncDiskMirror(b)
	return b, nil
}

// Upsert salva a configuração Whitelabel no Postgres, atualiza o cache em memória e o espelho em disco.
func (r *BrandingRepo) Upsert(ctx context.Context, in domain.PlatformBranding) (domain.PlatformBranding, error) {
	normalizeBranding(&in)
	rawTheme, err := json.Marshal(in.ThemeConfig)
	if err != nil {
		return in, fmt.Errorf("marshal theme_config: %w", err)
	}

	const q = `
		INSERT INTO platform_branding (
			id, platform_name, tagline, footer_text, logo_display_mode,
			logo_dark_url, logo_light_url, favicon_url, theme_config, updated_at
		) VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (id) DO UPDATE SET
			platform_name     = EXCLUDED.platform_name,
			tagline           = EXCLUDED.tagline,
			footer_text       = EXCLUDED.footer_text,
			logo_display_mode = EXCLUDED.logo_display_mode,
			logo_dark_url     = EXCLUDED.logo_dark_url,
			logo_light_url    = EXCLUDED.logo_light_url,
			favicon_url       = EXCLUDED.favicon_url,
			theme_config      = EXCLUDED.theme_config,
			updated_at        = NOW()
		RETURNING updated_at
	`
	if err := r.pool.QueryRow(
		ctx, q,
		in.PlatformName,
		in.Tagline,
		in.FooterText,
		in.LogoDisplayMode,
		in.LogoDarkURL,
		in.LogoLightURL,
		in.FaviconURL,
		rawTheme,
	).Scan(&in.UpdatedAt); err != nil {
		return in, fmt.Errorf("upsert platform_branding: %w", err)
	}

	r.mu.Lock()
	r.cached = &in
	r.mu.Unlock()

	r.syncDiskMirror(in)
	return in, nil
}

func normalizeBranding(b *domain.PlatformBranding) {
	def := domain.DefaultPlatformBranding()
	if strings.TrimSpace(b.PlatformName) == "" || b.PlatformName == "TrackFind" {
		b.PlatformName = def.PlatformName
	}
	switch b.LogoDisplayMode {
	case "icon_text", "logo_only", "logo_and_text":
	default:
		b.LogoDisplayMode = "icon_text"
	}
	if b.ThemeConfig.Dark.Brand == "" {
		b.ThemeConfig.Dark = def.ThemeConfig.Dark
	}
	if b.ThemeConfig.Light.Brand == "" {
		b.ThemeConfig.Light = def.ThemeConfig.Light
	}
}

// syncDiskMirror grava cópias físicas das imagens DataURL em ./data/branding (best-effort).
func (r *BrandingRepo) syncDiskMirror(b domain.PlatformBranding) {
	if r.mirrorDir == "" {
		return
	}
	if err := os.MkdirAll(r.mirrorDir, 0o755); err != nil {
		return
	}
	writeDataURLFile(filepath.Join(r.mirrorDir, "logo-dark"), b.LogoDarkURL)
	writeDataURLFile(filepath.Join(r.mirrorDir, "logo-light"), b.LogoLightURL)
	writeDataURLFile(filepath.Join(r.mirrorDir, "favicon"), b.FaviconURL)
}

func writeDataURLFile(basePath, dataURL string) {
	if !strings.HasPrefix(dataURL, "data:image/") {
		return
	}
	comma := strings.IndexByte(dataURL, ',')
	if comma < 0 {
		return
	}
	meta := dataURL[:comma]
	payload := dataURL[comma+1:]
	ext := ".bin"
	switch {
	case strings.Contains(meta, "image/svg+xml"):
		ext = ".svg"
	case strings.Contains(meta, "image/webp"):
		ext = ".webp"
	case strings.Contains(meta, "image/png"):
		ext = ".png"
	case strings.Contains(meta, "image/jpeg"):
		ext = ".jpg"
	case strings.Contains(meta, "image/x-icon"), strings.Contains(meta, "image/vnd.microsoft.icon"):
		ext = ".ico"
	}
	var raw []byte
	var err error
	if strings.Contains(meta, ";base64") {
		raw, err = base64.StdEncoding.DecodeString(payload)
	} else {
		raw = []byte(payload)
	}
	if err != nil || len(raw) == 0 {
		return
	}
	if werr := os.WriteFile(basePath+ext, raw, 0o644); werr != nil {
		slog.Debug("branding disk mirror write skipped", "err", werr)
	}
}

