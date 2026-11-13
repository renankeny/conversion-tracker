package handlers

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// setFirstPartyTrackingCookies emite Set-Cookie 1st-party (imune ao cap de 7 dias
// do Safari ITP em document.cookie) quando a requisição chega via subdomínio CNAME
// próprio do cliente (ex: track.seudominio.com.br != TRACKING_TARGET_HOST).
func setFirstPartyTrackingCookies(
	w http.ResponseWriter,
	r *http.Request,
	trackingTargetHost, pixelTrackingHost string,
	req sendRequest,
) {
	reqHost := stripHostPort(r.Host)
	if reqHost == "" || reqHost == "localhost" || net.ParseIP(reqHost) != nil {
		return
	}

	targetHost := stripHostPort(trackingTargetHost)
	pxHost := stripHostPort(pixelTrackingHost)

	if targetHost != "" && strings.EqualFold(reqHost, targetHost) {
		return
	}
	if targetHost == "" && pxHost == "" {
		return
	}
	if targetHost == "" && !strings.EqualFold(reqHost, pxHost) {
		return
	}

	rootDomain, err := publicsuffix.EffectiveTLDPlusOne(strings.ToLower(reqHost))
	if err != nil || rootDomain == "" {
		return
	}

	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		originHost := extractOriginHost(origin)
		if originHost == "" || !sameRegistrableDomain(originHost, rootDomain) {
			return
		}
	}

	secure := isHTTPSRequest(r)

	if validCookieValue(req.UserID, 64) {
		writeTrackingCookie(w, "tf_uid", req.UserID, rootDomain, 365*86400, secure)
	}
	if strings.HasPrefix(req.FBP, "fb.") && validCookieValue(req.FBP, 128) {
		writeTrackingCookie(w, "_fbp", req.FBP, rootDomain, 90*86400, secure)
	}
	if strings.HasPrefix(req.FBC, "fb.") && validCookieValue(req.FBC, 256) {
		writeTrackingCookie(w, "_fbc", req.FBC, rootDomain, 90*86400, secure)
	}
	if validCookieValue(req.TTP, 128) {
		writeTrackingCookie(w, "_ttp", req.TTP, rootDomain, 390*86400, secure)
	}
}

func writeTrackingCookie(w http.ResponseWriter, name, val, domain string, maxAge int, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    val,
		Path:     "/",
		Domain:   domain,
		MaxAge:   maxAge,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		HttpOnly: false,
	})
}

func stripHostPort(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		return strings.Trim(h, "[]")
	}
	return strings.Trim(s, "[]")
}

func extractOriginHost(origin string) string {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return ""
	}
	return stripHostPort(u.Host)
}

func sameRegistrableDomain(host, rootDomain string) bool {
	return host == rootDomain || strings.HasSuffix(host, "."+rootDomain)
}

func isHTTPSRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func validCookieValue(s string, maxLen int) bool {
	if s == "" || len(s) > maxLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c <= 0x20 || c >= 0x7f || c == ';' || c == ',' || c == '"' || c == '\\' {
			return false
		}
	}
	return true
}
