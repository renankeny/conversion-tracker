package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type HealthAlert struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`     // "pixel_inactive", "google_ads_disconnected", "webhook_errors"
	Severity  string    `json:"severity"` // "warning", "danger", "info"
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
	qcache       *cache.QueryCache
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

func (h *AdminAlerts) SetQueryCache(qc *cache.QueryCache) {
	h.qcache = qc
}

// GetHealthAlerts executa diagnósticos do sistema e devolve anomalias ativas.
func (h *AdminAlerts) GetHealthAlerts(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	writeDashboardCachedJSON(w, r, h.qcache, scope.AccountID, "health_alerts", 30*time.Second, "health alerts error", func(ctx context.Context) (any, error) {
		var alerts []HealthAlert
		now := time.Now()

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

		if h.webhookLogs != nil {
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

		status := "healthy"
		if len(alerts) > 0 {
			status = "degraded"
		}
		return map[string]any{
			"alerts": alerts,
			"count":  len(alerts),
			"status": status,
		}, nil
	})
}
