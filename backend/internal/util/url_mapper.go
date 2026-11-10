package util

import (
	"net/url"
	"strings"
	"sync"
)

// URLMapper aplica o mapeamento "black -> white" em URLs antes de enviá-las
// pra Meta CAPI. Match em ordem de especificidade:
//  1. <scheme>://<host><path>   -> usa target inteiro (substitui tudo, mantém só query e fragment da black)
//  2. <host><path>              -> idem
//  3. <host>                    -> usa só host (e scheme) da target, PRESERVA path/query/fragment da black
//
// É thread-safe: pode ser atualizada via Set() enquanto requisições rodam.
type URLMapper struct {
	mu        sync.RWMutex
	m         map[string]string
	byAccount map[int64]map[string]string
}

// NewURLMapper cria um URLMapper vazio.
func NewURLMapper() *URLMapper {
	return &URLMapper{
		m:         map[string]string{},
		byAccount: map[int64]map[string]string{},
	}
}

// Set substitui o conjunto global de mapeamentos (retrocompatibilidade).
func (u *URLMapper) Set(mappings map[string]string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.m = make(map[string]string, len(mappings))
	for k, v := range mappings {
		u.m[k] = v
	}
}

// SetByAccount substitui o conjunto de mapeamentos isolado por empresa (account_id).
func (u *URLMapper) SetByAccount(byAcc map[int64]map[string]string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	next := make(map[int64]map[string]string, len(byAcc))
	flat := make(map[string]string)
	for accID, mp := range byAcc {
		cp := make(map[string]string, len(mp))
		for k, v := range mp {
			cp[k] = v
			flat[k] = v
		}
		next[accID] = cp
	}
	u.byAccount = next
	u.m = flat
}

// Apply traduz uma URL black em white usando o mapa global.
func (u *URLMapper) Apply(incoming string) string {
	return u.ApplyForAccount(0, incoming)
}

// ApplyForAccount traduz uma URL black em white usando apenas as regras da empresa informada
// (quando accountID > 0 e SetByAccount estiver carregado), evitando vazamento entre tenants.
func (u *URLMapper) ApplyForAccount(accountID int64, incoming string) string {
	if incoming == "" {
		return incoming
	}
	black, err := url.Parse(incoming)
	if err != nil {
		return incoming
	}
	host := strings.ToLower(black.Host)
	if host == "" {
		return incoming
	}
	scheme := black.Scheme
	if scheme == "" {
		scheme = "https"
	}
	path := strings.TrimRight(black.Path, "/")

	// Tenta os matches do mais específico pro menos específico.
	type match struct {
		key      string
		hostOnly bool // true quando o match foi só pelo host (preservar path da black)
	}
	candidates := []match{
		{scheme + "://" + host + path, false},
		{host + path, false},
		{host, true},
	}

	u.mu.RLock()
	var rules map[string]string
	if accountID > 0 && len(u.byAccount) > 0 {
		rules = u.byAccount[accountID]
	} else {
		rules = u.m
	}
	var (
		target   string
		hostOnly bool
		found    bool
	)
	if len(rules) > 0 {
		for _, c := range candidates {
			if v, ok := rules[c.key]; ok {
				target = v
				hostOnly = c.hostOnly
				found = true
				break
			}
		}
	}
	u.mu.RUnlock()
	if !found {
		return incoming
	}

	white, err := url.Parse(target)
	if err != nil {
		return incoming
	}
	if white.Scheme == "" {
		white.Scheme = "https"
	}
	if hostOnly {
		// Match foi por host. Preserva path da black na URL final.
		// O target deve ser o "novo host" (com scheme opcional).
		white.Path = black.Path
	}
	mergeQueryFromBlack(white, black)
	if black.Fragment != "" {
		white.Fragment = black.Fragment
	}
	return white.String()
}

// mergeQueryFromBlack mescla a query da URL black sobre a white. Os valores
// da black sobrescrevem os da white quando houver conflito.
func mergeQueryFromBlack(white, black *url.URL) {
	if black.RawQuery == "" {
		return
	}
	wq := white.Query()
	for k, vs := range black.Query() {
		// substitui completamente o slice da white pra esse param
		wq[k] = vs
	}
	white.RawQuery = wq.Encode()
}
