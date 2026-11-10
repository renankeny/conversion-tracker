package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type HealthAlert struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`      // "pixel_inactive", "google_ads_disconnected", "webhook_errors"
	Severity  string    `json:"severity"`  // "warning", "danger", "info"
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	ActionURL string    `json:"action_url,omitempty"`
	ActionTxt string    `json:"action_text,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminAlerts struct {
	pixels       *repo.PixelRepo
	googleAdsAcc *repo.GoogleAdsAccountRepo
	webhookLogs  *repo.WebhookLogRepo
	counters     *repo.EventCounterRepo
}

func NewAdminAlerts(
	pixels *repo.PixelRepo,
	googleAdsAcc *repo.GoogleAdsAccountRepo,
	webhookLogs *repo.WebhookLogRepo,
	counters *repo.EventCounterRepo,
) *AdminAlerts {
	return &AdminAlerts{
		pixels:       pixels,
		googleAdsAcc: googleAdsAcc,
		webhookLogs:  webhookLogs,
		counters:     counters,
	}
}

// GetHealthAlerts executa diagnósticos do sistema e devolve anomalias ativas.
func (h *AdminAlerts) GetHealthAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := TenantFromContext(ctx)

	var alerts []HealthAlert
	now := time.Now()

	// 1) Diagnóstico de Google Ads Desconectado / Sem Token
	if h.googleAdsAcc != nil {
		accounts, err := h.googleAdsAcc.ListByAccount(ctx, scope.AccountID)
		if err == nil {
			for _, acc := range accounts {
				if acc.Enabled && !acc.IsConnected() {
					alerts = append(alerts, HealthAlert{
						ID:        "gads_" + strconv.FormatInt(acc.ID, 10),
						Type:      "google_ads_disconnected",
						Severity:  "warning",
						Title:     "Google Ads Desconectado",
						Message:   "A conta de Google Ads (" + acc.CustomerID + ") está sem autorização ativa. Reconecte para manter o envio de conversões.",
						ActionURL: "/google-ads",
						ActionTxt: "Reconectar Conta",
						CreatedAt: now,
					})
				}
			}
		}
	}

	// 2) Diagnóstico de Webhooks com Erro Recente
	if h.webhookLogs != nil {
		// Busca falhas nas últimas 6 horas
		from := now.Add(-6 * time.Hour)
		_, failCount, err := h.webhookLogs.List(ctx, repo.WebhookLogFilter{
			AccountID: scope.AccountID,
			Status:    "failed",
			From:      from,
			Limit:     1,
		})
		if err == nil && failCount > 0 {
			alerts = append(alerts, HealthAlert{
				ID:        "webhook_recent_failures",
				Type:      "webhook_errors",
				Severity:  "danger",
				Title:     "Erros em Webhooks Recentes",
				Message:   "Foram detectadas falhas em webhooks de pagamento nas últimas horas. Verifique os logs para garantir que nenhuma compra foi perdida.",
				ActionURL: "/webhooks/logs",
				ActionTxt: "Ver Logs de Webhook",
				CreatedAt: now,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"alerts": alerts,
		"count":  len(alerts),
		"status": func() string {
			if len(alerts) == 0 {
				return "healthy"
			}
			return "degraded"
		}(),
	})
}
