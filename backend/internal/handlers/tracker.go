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
	"log/slog"
	"net/http"
	"text/template"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

//go:embed assets/loader.js.tmpl assets/tracker.js.tmpl
var assetsFS embed.FS

// TrackerHandler serve /loader.js e /tracker.js a partir dos templates
// embedados, injetando a configuração do pixel inline.
type TrackerHandler struct {
	pixels         *repo.PixelRepo
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
		loaderTpl:      loaderTpl,
		trackerTpl:     trackerTpl,
		fallbackAPIURL: fallbackAPIURL,
	}, nil
}

// Loader serve /api/v1/loader.js. Cache: 1h (é estático).
func (h *TrackerHandler) Loader(w http.ResponseWriter, r *http.Request) {
	pixel := r.URL.Query().Get("pixel")
	// loader em si não exige pixel válido — quem valida é o tracker.js.
	// Renderiza só pra deixar o slug visível em comentário.

	var buf bytes.Buffer
	err := h.loaderTpl.Execute(&buf, map[string]any{
		"PixelName":   safeSlug(pixel),
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
	pixelName := r.URL.Query().Get("pixel")
	if pixelName == "" {
		http.Error(w, "pixel param required", http.StatusBadRequest)
		return
	}

	pixel, err := h.pixels.FindByName(r.Context(), pixelName)
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
	metaInitIDJSON, _ := json.Marshal(pixel.TrackerConfig.MetaInitPixelID)
	apiURLJSON, _ := json.Marshal(h.fallbackAPIURL)

	var buf bytes.Buffer
	err = h.trackerTpl.Execute(&buf, map[string]any{
		"PixelName":           pixel.Name,
		"PixelNameJSON":       string(pixelNameJSON),
		"ConfigJSON":          string(configJSON),
		"MetaInitPixelIDJSON": string(metaInitIDJSON),
		"APIURLJSON":          string(apiURLJSON),
		"APIURL":              h.fallbackAPIURL,
		"GeneratedAt":         time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		slog.Error("tracker render", "err", err, "pixel_name", pixelName)
		http.Error(w, "tracker render error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(buf.Bytes())
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
