package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

const maxDataURLLength = 1024 * 1024 // 1 MB por imagem (as imagens otimizadas têm ~15-60KB)

// BrandingHandler serve a configuração pública de Whitelabel e o endpoint admin de atualização.
type BrandingHandler struct {
	repo *repo.BrandingRepo
	rdb  *redis.Client
}

func NewBrandingHandler(r *repo.BrandingRepo) *BrandingHandler {
	return &BrandingHandler{repo: r}
}

func (h *BrandingHandler) SetRedis(rdb *redis.Client) {
	h.rdb = rdb
}

// GetPublic GET /api/v1/branding
func (h *BrandingHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	b, err := h.repo.Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, domain.DefaultPlatformBranding())
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, http.StatusOK, b)
}

// UpdateAdmin PUT /api/v1/admin/branding
func (h *BrandingHandler) UpdateAdmin(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok || claims == nil || !claims.IsOwner() {
		writeError(w, http.StatusForbidden, "apenas o proprietário da plataforma pode alterar a personalização Whitelabel")
		return
	}

	var in domain.PlatformBranding
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}

	in.PlatformName = strings.TrimSpace(in.PlatformName)
	in.Tagline = strings.TrimSpace(in.Tagline)
	in.FooterText = strings.TrimSpace(in.FooterText)

	if len(in.PlatformName) > 60 {
		writeError(w, http.StatusBadRequest, "nome da plataforma deve ter no máximo 60 caracteres")
		return
	}
	if len(in.LogoDarkURL) > maxDataURLLength || len(in.LogoLightURL) > maxDataURLLength || len(in.FaviconURL) > maxDataURLLength {
		writeError(w, http.StatusBadRequest, "imagem excede o tamanho máximo permitido (1MB)")
		return
	}
	if !SafeLogoOrFaviconURL(in.LogoDarkURL) || !SafeLogoOrFaviconURL(in.LogoLightURL) || !SafeLogoOrFaviconURL(in.FaviconURL) {
		writeError(w, http.StatusBadRequest, "formato de imagem ou URL não permitido (use PNG, JPG, WEBP, GIF, ICO ou HTTPS)")
		return
	}

	if !validatePalette(&in.ThemeConfig.Dark) || !validatePalette(&in.ThemeConfig.Light) {
		writeError(w, http.StatusBadRequest, "cores devem estar no formato hexadecimal (#RRGGBB)")
		return
	}

	saved, err := h.repo.Upsert(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar personalização: "+err.Error())
		return
	}
	cache.PublishInvalidate(r.Context(), h.rdb, cache.InvalidateEvent{Kind: "branding"})
	writeJSON(w, http.StatusOK, saved)
}

func validatePalette(p *domain.ThemePalette) bool {
	for _, c := range []*string{&p.Brand, &p.Surface1, &p.Surface2, &p.SurfaceDeep} {
		*c = strings.TrimSpace(*c)
		if *c != "" && !hexColorRegex.MatchString(*c) {
			return false
		}
	}
	return true
}
