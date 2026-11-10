package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

var safeTraefikHostRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$`)

type TraefikConfigHandler struct {
	pixelRepo  *repo.PixelRepo
	secret     string
	backendURL string
	isProd     bool
}

func NewTraefikConfig(pixelRepo *repo.PixelRepo, secret string) *TraefikConfigHandler {
	backendURL := strings.TrimSpace(os.Getenv("TRAEFIK_BACKEND_URL"))
	if backendURL == "" {
		backendURL = "http://tasks.trackfind_trackfind-api:8080"
	}
	return &TraefikConfigHandler{
		pixelRepo:  pixelRepo,
		secret:     strings.TrimSpace(secret),
		backendURL: backendURL,
		isProd:     strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production"),
	}
}

// TraefikDynamicConfig representa o JSON dinâmico do Traefik v3 HTTP Provider.
type TraefikDynamicConfig struct {
	HTTP struct {
		Routers  map[string]TraefikRouter  `json:"routers,omitempty"`
		Services map[string]TraefikService `json:"services,omitempty"`
	} `json:"http"`
}

type TraefikRouter struct {
	Rule        string      `json:"rule"`
	Service     string      `json:"service"`
	EntryPoints []string    `json:"entryPoints,omitempty"`
	TLS         *TraefikTLS `json:"tls,omitempty"`
}

type TraefikTLS struct {
	CertResolver string `json:"certResolver,omitempty"`
}

type TraefikService struct {
	LoadBalancer struct {
		Servers []TraefikServer `json:"servers"`
	} `json:"loadBalancer"`
}

type TraefikServer struct {
	URL string `json:"url"`
}

func (h *TraefikConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.secret == "" {
		if h.isProd {
			http.Error(w, "traefik secret not configured", http.StatusForbidden)
			return
		}
	} else {
		cand := strings.TrimSpace(r.URL.Query().Get("secret"))
		if cand == "" {
			cand = stripAuthPrefix(r.Header.Get("Authorization"))
		}
		if subtle.ConstantTimeCompare([]byte(cand), []byte(h.secret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	hosts, err := h.pixelRepo.ActiveTrackingHosts(ctx)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	cfg := TraefikDynamicConfig{}
	validHosts := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" && len(host) <= 253 && strings.Contains(host, ".") && safeTraefikHostRe.MatchString(host) {
			validHosts = append(validHosts, host)
		}
	}

	cfg.HTTP.Routers = make(map[string]TraefikRouter)
	if len(validHosts) > 0 {
		cfg.HTTP.Services = make(map[string]TraefikService)
		rules := make([]string, 0, len(validHosts))
		for _, host := range validHosts {
			rules = append(rules, "Host(`"+host+"`)")
		}

		cfg.HTTP.Routers["dynamic-api-router"] = TraefikRouter{
			Rule:        strings.Join(rules, " || "),
			Service:     "dynamic-api-service",
			EntryPoints: []string{"websecure"},
			TLS:         &TraefikTLS{CertResolver: "letsencryptresolver"},
		}
		cfg.HTTP.Services["dynamic-api-service"] = TraefikService{
			LoadBalancer: struct {
				Servers []TraefikServer `json:"servers"`
			}{
				Servers: []TraefikServer{{URL: h.backendURL}},
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}
