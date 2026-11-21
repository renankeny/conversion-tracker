package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// GetEasyPanelStatus GET /api/v1/admin/system/easypanel
// Informa ao painel administrativo se a API Key do EasyPanel já está vinculada.
func (h *SetupHandler) GetEasyPanelStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.calibrator == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"linked":   true,
			"hint_url": "",
			"version":  util.AppBuildVersion,
		})
		return
	}
	linked, hintURL, ver := h.calibrator.EasyPanelStatus(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"linked":   linked,
		"hint_url": hintURL,
		"version":  ver,
	})
}

// ConnectEasyPanel POST /api/v1/admin/system/easypanel
// Valida ao vivo e salva a URL + API Key do EasyPanel (ou marca que não utiliza EasyPanel).
func (h *SetupHandler) ConnectEasyPanel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PanelURL      string `json:"panel_url"`
		APIToken      string `json:"api_token"`
		SkipEasyPanel bool   `json:"skip_easypanel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload inválido")
		return
	}
	req.PanelURL = strings.TrimSpace(req.PanelURL)
	req.APIToken = strings.TrimSpace(req.APIToken)
	if !req.SkipEasyPanel && (req.PanelURL == "" || req.APIToken == "") {
		writeError(w, http.StatusBadRequest, "Informe a URL do seu EasyPanel e o Token de API.")
		return
	}
	if h.calibrator == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "linked": true})
		return
	}
	project, err := h.calibrator.ConnectEasyPanel(r.Context(), req.PanelURL, req.APIToken, req.SkipEasyPanel)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"linked":  true,
		"project": project,
	})
}
