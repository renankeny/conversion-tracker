// Package handlers — endpoints públicos /loader.js e /tracker.js.
//
// O loader é um stub micro (~1KB), cacheável longa duração, que apenas
// detecta o pixel slug e injeta o tracker principal. O tracker é gerado
// dinamicamente a partir de tracker.js.tmpl + a config persistida em
// pixels.tracker_config.
package handlers

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"html"
	"log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

//go:embed assets/loader.js.tmpl assets/tracker.js.tmpl
var assetsFS embed.FS

// TrackerHandler serve /loader.js e /tracker.js
// a partir dos templates embedados, injetando a configuração do pixel inline.
type TrackerHandler struct {
	pixels         *repo.PixelRepo
	branding       *repo.BrandingRepo
	pixelCache     *cache.PixelCache
	loaderTpl      *template.Template
	trackerTpl     *template.Template
	fallbackAPIURL string
}

func NewTracker(p *repo.PixelRepo, fallbackAPIURL string) (*TrackerHandler, error) {
	loaderTpl, err := template.ParseFS(assetsFS, "assets/loader.js.tmpl")
	if err != nil {
		return nil, err
	}
	trackerTpl, err := template.ParseFS(assetsFS, "assets/tracker.js.tmpl")
	if err != nil {
		return nil, err
	}

	return &TrackerHandler{
		pixels:         p,
		pixelCache:     p.PixelCache(),
		loaderTpl:      loaderTpl,
		trackerTpl:     trackerTpl,
		fallbackAPIURL: fallbackAPIURL,
	}, nil
}

// SetBranding injeta o repositório Whitelabel para assinatura dinâmica do loader.
func (h *TrackerHandler) SetBranding(b *repo.BrandingRepo) {
	h.branding = b
}

func (h *TrackerHandler) SetPixelCache(pc *cache.PixelCache) {
	h.pixelCache = pc
}

// Loader serve /api/v1/loader.js. Cache: 1h (é estático).
func (h *TrackerHandler) Loader(w http.ResponseWriter, r *http.Request) {
	// Quando acessado diretamente pelo browser (Accept: text/html),
	// retorna uma tela preta com o nome da plataforma — não expõe o código fonte.
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		brandName := "Conversion Tracker"
		if h.branding != nil {
			if b, err := h.branding.Get(r.Context()); err == nil && b.PlatformName != "" {
				brandName = b.PlatformName
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><style>*{margin:0;padding:0}html,body{background:#000;height:100%;display:flex;align-items:center;justify-content:center}p{color:#fff;font-family:monospace;font-size:1rem}</style></head><body><p>` + html.EscapeString(brandName) + `</p></body></html>`))
		return
	}

	pixel := r.URL.Query().Get("pixel")
	// loader em si não exige pixel válido — quem valida é o tracker.js.
	// Renderiza só pra deixar o slug visível em comentário.

	var buf bytes.Buffer
	err := h.loaderTpl.Execute(&buf, map[string]any{
		"PixelName":   safeSlug(pixel),
		"APIURL":      h.fallbackAPIURL,
		"GeneratedAt": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		http.Error(w, "loader render error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(buf.Bytes())
}

// Tracker serve /api/v1/tracker.js?pixel=<slug>. Renderiza o template com a
// config persistida pra esse pixel. Cache curto (60s) pra refletir mudanças
// do painel rapidamente.
func (h *TrackerHandler) Tracker(w http.ResponseWriter, r *http.Request) {
	pixelName := strings.TrimSpace(r.URL.Query().Get("pixel"))
	if pixelName == "" {
		http.Error(w, "pixel param required", http.StatusBadRequest)
		return
	}

	pc := h.pixelCache
	if pc == nil && h.pixels != nil {
		pc = h.pixels.PixelCache()
	}
	if cached, ok := pc.GetTrackerScript(r.Context(), pixelName); ok {
		writeTrackerResponse(w, r, cached.Body, cached.ETag)
		return
	}

	pixel, err := h.pixels.FindByNameOrToken(r.Context(), pixelName)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			http.Error(w, "unknown pixel", http.StatusNotFound)
			return
		}
		slog.Error("tracker find pixel", "err", err, "pixel_name", pixelName)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !pixel.Enabled {
		http.Error(w, "pixel disabled", http.StatusForbidden)
		return
	}

	configJSON, err := json.Marshal(pixel.TrackerConfig)
	if err != nil {
		slog.Error("tracker marshal config", "err", err, "pixel_name", pixelName)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	pixelNameJSON, _ := json.Marshal(pixel.Name)

	metaInitID := strings.TrimSpace(pixel.TrackerConfig.MetaInitPixelID)
	if metaInitID == "" && len(pixel.PixelIDs) > 0 {
		metaInitID = strings.TrimSpace(pixel.PixelIDs[0])
	}
	metaInitIDJSON, _ := json.Marshal(metaInitID)
	metaPixelIDs := uniqueNonEmptyStrings(append([]string{metaInitID}, pixel.PixelIDs...))
	metaPixelIDsJSON, _ := json.Marshal(metaPixelIDs)

	tiktokInitID := ""
	if len(pixel.TikTokPixelIDs) > 0 {
		tiktokInitID = strings.TrimSpace(pixel.TikTokPixelIDs[0])
	}
	tiktokInitIDJSON, _ := json.Marshal(tiktokInitID)
	tiktokPixelIDs := uniqueNonEmptyStrings(pixel.TikTokPixelIDs)
	tiktokPixelIDsJSON, _ := json.Marshal(tiktokPixelIDs)
	apiURLJSON, _ := json.Marshal(h.fallbackAPIURL)
	ga4MeasurementIDJSON, _ := json.Marshal(pixel.GA4MeasurementID)

	genAt := pixel.UpdatedAt.UTC().Format(time.RFC3339)
	if pixel.UpdatedAt.IsZero() {
		genAt = "v2"
	}

	var buf bytes.Buffer
	err = h.trackerTpl.Execute(&buf, map[string]any{
		"PixelName":             safeSlug(pixel.Name),
		"PixelNameJSON":         string(pixelNameJSON),
		"ConfigJSON":            string(configJSON),
		"MetaInitPixelIDJSON":   string(metaInitIDJSON),
		"MetaPixelIDsJSON":      string(metaPixelIDsJSON),
		"TiktokInitPixelIDJSON": string(tiktokInitIDJSON),
		"TiktokPixelIDsJSON":    string(tiktokPixelIDsJSON),
		"GA4MeasurementIDJSON":  string(ga4MeasurementIDJSON),
		"APIURLJSON":            string(apiURLJSON),
		"APIURL":                h.fallbackAPIURL,
		"GeneratedAt":           genAt,
	})
	if err != nil {
		slog.Error("tracker render", "err", err, "pixel_name", pixelName)
		http.Error(w, "tracker render error", http.StatusInternalServerError)
		return
	}

	script := pc.SetTrackerScript(r.Context(), buf.String(), pixel.Name, pixel.PublicToken)
	writeTrackerResponse(w, r, script.Body, script.ETag)
}

func writeTrackerResponse(w http.ResponseWriter, r *http.Request, body, etag string) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if etag != "" {
		w.Header().Set("ETag", etag)
		if inm := strings.TrimSpace(r.Header.Get("If-None-Match")); inm != "" && inm == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	_, _ = w.Write([]byte(body))
}

// safeSlug evita injetar caracteres estranhos no comentário do loader.
// Strip simples de tudo fora de [a-zA-Z0-9_-].
func safeSlug(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			out = append(out, c)
		}
	}
	return string(out)
}

func uniqueNonEmptyStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
