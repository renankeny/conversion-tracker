package util

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

// SafeGo executa uma goroutine protegida contra panic para nunca derrubar o processo inteiro.
func SafeGo(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("background goroutine panic recovered",
					"worker", name,
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}

// ValidateServiceEndpointURL valida URLs de serviços externos/containers (Evolution Go, UAZAPI),
// bloqueando esquemas inválidos, endpoints de cloud metadata (169.254.169.254) e portas internas críticas.
func ValidateServiceEndpointURL(rawURL string) (string, error) {
	s := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if s == "" {
		return "", errors.New("URL obrigatória")
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("URL inválida (use http:// ou https://)")
	}
	host := strings.ToLower(u.Hostname())
	if host == "metadata.google.internal" || host == "metadata" || host == "169.254.169.254" {
		return "", errors.New("host de metadados bloqueado por segurança")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return "", errors.New("endereço IP reservado bloqueado")
		}
	}
	switch u.Port() {
	case "5432", "6379", "9000", "9001", "2375", "2376":
		return "", errors.New("porta interna de infraestrutura bloqueada")
	}
	return s, nil
}

// IsPrivateOrReservedIP identifica IPs não roteáveis na internet pública (SSRF protection).
func IsPrivateOrReservedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

// NewSSRFSafeHTTPClient retorna um http.Client cujo DialContext rejeita conexões para IPs privados/internos.
func NewSSRFSafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 8 * time.Second, KeepAlive: 30 * time.Second}
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          32,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil || len(ips) == 0 {
				return nil, errors.New("falha ao resolver host externo")
			}
			for _, ipa := range ips {
				if IsPrivateOrReservedIP(ipa.IP) {
					return nil, errors.New("conexão para IP privado ou interno bloqueada")
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}
	return &http.Client{Timeout: timeout, Transport: tr}
}
