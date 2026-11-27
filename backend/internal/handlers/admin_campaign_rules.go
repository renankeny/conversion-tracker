package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/campaignrules"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminCampaignRules struct {
	rules  *repo.CampaignRulesRepo
	logs   *repo.CampaignRuleLogsRepo
	runner *campaignrules.Runner
}

func NewAdminCampaignRules(rules *repo.CampaignRulesRepo, logs *repo.CampaignRuleLogsRepo, runner *campaignrules.Runner) *AdminCampaignRules {
	return &AdminCampaignRules{rules: rules, logs: logs, runner: runner}
}

func (h *AdminCampaignRules) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	items, err := h.rules.ListByAccount(r.Context(), scope.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar regras de campanhas")
		return
	}
	nowSP := time.Now().In(campaignrules.SaoPauloLocation())
	startOfDaySP := time.Date(nowSP.Year(), nowSP.Month(), nowSP.Day(), 0, 0, 0, 0, nowSP.Location())
	execToday, simToday, _ := h.logs.CountTodayExecutions(r.Context(), scope.AccountID, startOfDaySP)

	activeCount, dryRunCount := 0, 0
	for _, it := range items {
		if it.Enabled {
			activeCount++
		}
		if it.DryRun || it.ActionType == domain.RuleActionNotifyOnly {
			dryRunCount++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"stats": map[string]int{
			"total":           len(items),
			"active":          activeCount,
			"dry_run":         dryRunCount,
			"executed_today":  execToday,
			"simulated_today": simToday,
		},
	})
}

func (h *AdminCampaignRules) Create(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para criar regras de campanhas")
		return
	}
	var rule domain.CampaignRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	scope := TenantFromContext(r.Context())
	rule.AccountID = scope.AccountID

	if err := h.rules.Create(r.Context(), &rule); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": rule})
}

func (h *AdminCampaignRules) Update(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar regras de campanhas")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID de regra inválido")
		return
	}
	var rule domain.CampaignRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	scope := TenantFromContext(r.Context())
	rule.ID = id
	rule.AccountID = scope.AccountID

	if err := h.rules.Update(r.Context(), &rule); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "regra não encontrada")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rule})
}

func (h *AdminCampaignRules) Toggle(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para alterar regras de campanhas")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID de regra inválido")
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	scope := TenantFromContext(r.Context())
	if err := h.rules.ToggleEnabled(r.Context(), scope.AccountID, id, body.Enabled); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "regra não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao atualizar status da regra")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "enabled": body.Enabled})
}

func (h *AdminCampaignRules) Delete(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para excluir regras de campanhas")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID de regra inválido")
		return
	}
	scope := TenantFromContext(r.Context())
	if err := h.rules.Delete(r.Context(), scope.AccountID, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "regra não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao excluir regra")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *AdminCampaignRules) RunNow(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para executar regras de campanhas")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID de regra inválido")
		return
	}
	scope := TenantFromContext(r.Context())
	rule, err := h.rules.GetByID(r.Context(), scope.AccountID, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "regra não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar regra")
		return
	}
	dryQ := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("dry_run")))
	overrideDryRun := dryQ == "1" || dryQ == "true"

	summary, err := h.runner.EvaluateRule(r.Context(), *rule, true, overrideDryRun)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": summary})
}

func (h *AdminCampaignRules) ListLogs(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	ruleID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("rule_id")), 10, 64)
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	items, err := h.logs.ListLogs(r.Context(), scope.AccountID, ruleID, status, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar histórico de execuções")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *AdminCampaignRules) UndoLog(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para desfazer ações de campanhas")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID do log inválido")
		return
	}
	scope := TenantFromContext(r.Context())
	updated, err := h.runner.UndoLog(r.Context(), scope.AccountID, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "registro de execução não encontrado")
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": updated})
}
