package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/geoip"
)

const (
	geoipEditionID = "GeoLite2-City"
	geoipDownload  = "https://download.maxmind.com/app/geoip_download"
)

// AdminGeoIP atualiza a base GeoLite2-City baixando da MaxMind via license key.
// Endpoint público apenas pra admins autenticados.
type AdminGeoIP struct {
	licenseKey string
	dbPath     string
	lookup     *geoip.Lookup
	refreshing atomic.Bool
}

func NewAdminGeoIP(licenseKey, dbPath string, lookup *geoip.Lookup) *AdminGeoIP {
	return &AdminGeoIP{
		licenseKey: licenseKey,
		dbPath:     dbPath,
		lookup:     lookup,
	}
}

// Status retorna informações sobre a base atualmente carregada
// (path, exists, size, modified). Útil pro painel mostrar estado.
func (h *AdminGeoIP) Status(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{
		"path":            h.dbPath,
		"license_key_set": h.licenseKey != "",
		"loaded":          h.lookup != nil && h.lookup.Available(),
	}
	if h.dbPath != "" {
		if info, err := os.Stat(h.dbPath); err == nil {
			out["exists"] = true
			out["size_bytes"] = info.Size()
			out["modified_at"] = info.ModTime().Format(time.RFC3339)
		} else {
			out["exists"] = false
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// Refresh baixa a base GeoLite2-City da MaxMind, valida SHA256, extrai o
// .mmdb e substitui o arquivo em h.dbPath. Recarrega o reader em memória.
func (h *AdminGeoIP) Refresh(w http.ResponseWriter, r *http.Request) {
	if h.licenseKey == "" {
		writeError(w, http.StatusBadRequest, "MAXMIND_LICENSE_KEY não configurada no servidor")
		return
	}
	if h.dbPath == "" {
		writeError(w, http.StatusBadRequest, "GEOIP_DB_PATH não configurado no servidor")
		return
	}

	if !h.refreshing.CompareAndSwap(false, true) {
		writeError(w, http.StatusConflict, "uma atualização já está em andamento")
		return
	}
	defer h.refreshing.Store(false)

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	result, err := h.downloadAndInstall(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao atualizar GeoIP: "+err.Error())
		return
	}

	if h.lookup != nil {
		if reloadErr := h.lookup.Reload(h.dbPath); reloadErr != nil {
			result["reload_error"] = reloadErr.Error()
		} else {
			result["reloaded"] = true
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// StartScheduler inicia uma goroutine que atualiza a base todo dia à 00:00.
func (h *AdminGeoIP) StartScheduler(ctx context.Context) {
	if h.licenseKey == "" || h.dbPath == "" {
		return
	}
	go func() {
		for {
			now := time.Now()
			loc, err := time.LoadLocation("America/Sao_Paulo")
			if err != nil {
				loc = time.Local
			}
			nowSP := now.In(loc)
			next := time.Date(nowSP.Year(), nowSP.Month(), nowSP.Day()+1, 0, 0, 0, 0, loc)
			duration := next.Sub(nowSP)

			select {
			case <-ctx.Done():
				return
			case <-time.After(duration):
				slog.Info("Running scheduled GeoIP update")
				if h.refreshing.CompareAndSwap(false, true) {
					refreshCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
					result, err := h.downloadAndInstall(refreshCtx)
					if err != nil {
						slog.Error("Scheduled GeoIP update failed", "err", err)
					} else {
						if h.lookup != nil {
							if reloadErr := h.lookup.Reload(h.dbPath); reloadErr != nil {
								slog.Error("Scheduled GeoIP reload failed", "err", reloadErr)
							}
						}
						slog.Info("Scheduled GeoIP update succeeded", "result", result)
					}
					cancel()
					h.refreshing.Store(false)
				}
			}
		}
	}()
}
