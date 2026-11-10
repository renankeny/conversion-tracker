package util

import (
	"net/url"
	"strings"
	"sync"
)

// DomainSet é um conjunto thread-safe de hostnames autorizados, alimentado
// a partir do banco e usado pelo handler CORS pra validar Origin dinamicamente.
//
// Suporta wildcard simples no formato "*.dominio.com" — qualquer subdomínio
// de dominio.com é aceito.
type DomainSet struct {
	mu        sync.RWMutex
	exact     map[string]struct{}
	wildcards []string // ex: ["dominio.com"] (sem o "*.") significa "qualquer *.dominio.com"
}

func NewDomainSet() *DomainSet {
	return &DomainSet{exact: map[string]struct{}{}}
}

// Set substitui a lista atual.
func (d *DomainSet) Set(domains []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.exact = make(map[string]struct{}, len(domains))
	d.wildcards = d.wildcards[:0]
	for _, raw := range domains {
		host := strings.ToLower(strings.TrimSpace(raw))
		if host == "" {
			continue
		}
		if strings.HasPrefix(host, "*.") {
			d.wildcards = append(d.wildcards, strings.TrimPrefix(host, "*."))
			continue
		}
		d.exact[host] = struct{}{}
	}
}

// AllowOrigin verifica se o Origin (URL completa) é permitido.
func (d *DomainSet) AllowOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return d.AllowHost(u.Hostname())
}

// AllowHost verifica se um hostname (sem scheme) é permitido.
func (d *DomainSet) AllowHost(host string) bool {
	host = strings.ToLower(host)
	if host == "" {
		return false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if _, ok := d.exact[host]; ok {
		return true
	}
	for _, base := range d.wildcards {
		if host == base || strings.HasSuffix(host, "."+base) {
			return true
		}
	}
	return false
}

// Size devolve o total de entradas (exatas + wildcards).
func (d *DomainSet) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.exact) + len(d.wildcards)
}
