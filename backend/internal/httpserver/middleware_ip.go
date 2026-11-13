package httpserver

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/cors"

	"github.com/renankeny/conversion-tracker/backend/internal/config"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

var trustedProxyCIDRs = mustParseCIDRs([]string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
})

func mustParseCIDRs(raw []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(raw))
	for _, s := range raw {
		_, cidr, err := net.ParseCIDR(s)
		if err == nil {
			out = append(out, cidr)
		}
	}
	return out
}

func isTrustedProxyIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, cidr := range trustedProxyCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// TrustedRealIP extrai X-Forwarded-For / X-Real-IP somente quando a conexão
// TCP imediata vem de um proxy reverso confiável (loopback / rede privada Docker).
// Requisições externas diretas não conseguem forjar o IP real.
func TrustedRealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteHost := r.RemoteAddr
		if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			remoteHost = h
		}
		remoteIP := net.ParseIP(strings.TrimSpace(remoteHost))
		if isTrustedProxyIP(remoteIP) {
			if realIP := extractForwardedClientIP(r); realIP != "" {
				r.RemoteAddr = realIP
			}
		}
		next.ServeHTTP(w, r)
	})
}

func extractForwardedClientIP(r *http.Request) string {
	// Cloudflare na frente do proxy: IP real do visitante vem em CF-Connecting-IP.
	if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" {
		if ip := net.ParseIP(cf); ip != nil {
			return ip.String()
		}
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		// Percorre da direita para a esquerda e escolhe o primeiro IP público válido
		// adicionado pelo proxy confiável mais externo.
		var firstValid string
		for i := len(parts) - 1; i >= 0; i-- {
			cand := strings.TrimSpace(parts[i])
			ip := net.ParseIP(cand)
			if ip == nil {
				continue
			}
			if firstValid == "" {
				firstValid = ip.String()
			}
			if !isTrustedProxyIP(ip) {
				return ip.String()
			}
		}
		if firstValid != "" {
			return firstValid
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		if ip := net.ParseIP(xrip); ip != nil {
			return ip.String()
		}
	}
	return ""
}

func buildCORS(cfg *config.Config, set *util.DomainSet) func(http.Handler) http.Handler {
	adminOrigin := normalizeOrigin(cfg.AppURL)
	if !cfg.StrictCORS {
		return cors.Handler(cors.Options{
			AllowOriginFunc:  func(_ *http.Request, _ string) bool { return true },
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Account-ID"},
			AllowCredentials: true,
			MaxAge:           300,
		})
	}
	return cors.Handler(cors.Options{
		AllowOriginFunc: func(_ *http.Request, origin string) bool {
			if adminOrigin != "" && strings.EqualFold(origin, adminOrigin) {
				return true
			}
			return set != nil && set.AllowOrigin(origin)
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Account-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}

func normalizeOrigin(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

