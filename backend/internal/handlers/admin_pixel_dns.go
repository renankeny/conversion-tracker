package handlers

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

// VerifyDNS verifica se o tracking_host do pixel está apontando corretamente
// para o servidor TrackFind (via CNAME ou A record).
//
// Retorna:
//   - status: "ok" | "wrong_target" | "not_found" | "no_host"
//   - cname:  o CNAME atual (se houver)
//   - a:      lista de IPs A apontados (se houver)
//   - target: o host alvo esperado (config TRACKING_TARGET_HOST)
//   - hint:   mensagem amigável explicando o estado
func (h *AdminPixels) VerifyDNS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == repo.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resolver := &net.Resolver{}

	cname, _ := resolver.LookupCNAME(ctx, host)
	cname = strings.TrimSuffix(cname, ".")
	cname = strings.ToLower(cname)

	ips, _ := resolver.LookupHost(ctx, host)

	// Resolve o target esperado pra IPs também (compara A records).
	var targetIPs []string
	if h.dnsTarget != "" {
		targetIPs, _ = resolver.LookupHost(ctx, h.dnsTarget)
	}

	status := "not_found"
	hint := "Não foi possível resolver o domínio. Verifique se o registro DNS foi criado e aguarde a propagação (até 30 minutos)."

	if cname == strings.ToLower(strings.TrimSuffix(h.dnsTarget, ".")) && h.dnsTarget != "" {
		status = "ok"
		hint = "DNS configurado corretamente via CNAME."
	} else if len(ips) > 0 && len(targetIPs) > 0 && hasOverlap(ips, targetIPs) {
		status = "ok"
		hint = "DNS configurado corretamente via A record."
	} else if cname != "" || len(ips) > 0 {
		status = "wrong_target"
		hint = "O domínio está resolvendo, mas não para o servidor TrackFind. Confira o valor do CNAME."
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
