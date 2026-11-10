package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// publicResolver usa 1.1.1.1 (Cloudflare) como nameserver, evitando o resolver
// interno do Docker que às vezes não consegue resolver CNAMEs de domínios
// externos a partir de containers em redes overlay.
var publicResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		d := net.Dialer{Timeout: 3 * time.Second}
		// Tenta Cloudflare primeiro, depois Google.
		if c, err := d.DialContext(ctx, "udp", "1.1.1.1:53"); err == nil {
			return c, nil
		}
		return d.DialContext(ctx, "udp", "8.8.8.8:53")
	},
}

// VerifyDNS verifica se o tracking_host do pixel está apontando corretamente
// para o servidor TrackFind (via CNAME ou A record).
//
// Usa resolver externo (1.1.1.1 / 8.8.8.8) em vez do resolver do sistema,
// que dentro de containers Docker às vezes não consegue resolver CNAMEs
// de domínios externos.
func (h *AdminPixels) VerifyDNS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && p.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}

	host := strings.ToLower(strings.TrimSpace(p.TrackingHost))
	if host == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "no_host",
			"hint":   "Pixel sem tracking_host configurado",
		})
		return
	}

	target := h.dnsTarget
	if target == "" {
		target = "(não configurado: defina TRACKING_TARGET_HOST no servidor)"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	cname, _ := publicResolver.LookupCNAME(ctx, host)
	cname = strings.TrimSuffix(cname, ".")
	cname = strings.ToLower(cname)

	ips, _ := publicResolver.LookupHost(ctx, host)

	// Resolve o target esperado pra IPs também (compara A records).
	var targetIPs []string
	if h.dnsTarget != "" {
		targetIPs, _ = publicResolver.LookupHost(ctx, h.dnsTarget)
	}

	expectedTarget := strings.ToLower(strings.TrimSuffix(h.dnsTarget, "."))

	status := "not_found"
	hint := "Não foi possível resolver o domínio. Verifique se o registro DNS foi criado e aguarde a propagação (até 30 minutos)."

	switch {
	case h.dnsTarget != "" && cname == expectedTarget:
		status = "ok"
		hint = "DNS configurado corretamente via CNAME."
	case h.dnsTarget != "" && strings.HasSuffix(cname, "."+expectedTarget):
		// Caso o CNAME aponte para um subdomínio do target (raro mas possível).
		status = "ok"
		hint = "DNS configurado corretamente via CNAME (subdomínio)."
	case len(ips) > 0 && len(targetIPs) > 0 && hasOverlap(ips, targetIPs):
		status = "ok"
		hint = "DNS configurado corretamente via A record."
	case cname != "" || len(ips) > 0:
		status = "wrong_target"
		hint = "O domínio está resolvendo, mas não para o servidor da plataforma. Confira o valor do CNAME."
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": status,
		"host":   host,
		"cname":  cname,
		"a":      ips,
		"target": target,
		"hint":   hint,
	})
}

func hasOverlap(a, b []string) bool {
	set := map[string]struct{}{}
	for _, x := range a {
		set[x] = struct{}{}
	}
	for _, y := range b {
		if _, ok := set[y]; ok {
			return true
		}
	}
	return false
}

// RegenerateToken POST /admin/pixels/{id}/regenerate-token
// Gera um novo public_token para o pixel e invalida imediatamente o antigo no Redis.
func (h *AdminPixels) RegenerateToken(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar pixels")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	var body struct {
		Confirm bool `json:"confirm"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if !body.Confirm {
		writeError(w, http.StatusConflict, "requires {\"confirm\": true}")
		return
	}
	newToken, err := h.repo.RegenerateToken(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "regenerate error: "+err.Error())
		return
	}
	h.repo.InvalidateCache(r.Context(), existing.Name, existing.PublicToken, newToken)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           id,
		"public_token": newToken,
	})
}
