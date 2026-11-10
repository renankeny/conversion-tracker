-- =====================================================================
-- 0020_webhook_logs.sql
-- Tabela de log de webhooks recebidos dos gateways (Hotmart, Cakto,
-- Kirvano, RkCursos). Armazena o purchaseInput normalizado para
-- permitir replay manual pelo painel admin.
-- =====================================================================

CREATE TABLE IF NOT EXISTS webhook_logs (
    id          BIGSERIAL PRIMARY KEY,
    account_id  BIGINT NOT NULL DEFAULT 1 REFERENCES accounts(id) ON DELETE CASCADE,
    source      TEXT NOT NULL,                          -- 'hotmart' | 'cakto' | 'kirvano' | 'rkcursos'
    pixel_name  TEXT NOT NULL DEFAULT '',               -- pixel_name do tracked_user ('' se não encontrado)
    external_id TEXT NOT NULL DEFAULT '',               -- external_id extraído (correlação com tracked_users)
    event_name  TEXT NOT NULL DEFAULT 'Purchase',       -- evento enfileirado
    event_id    TEXT NOT NULL DEFAULT '',               -- event_id enfileirado (correlação com sent_events)
    input_json  JSONB,                                  -- domain.WebhookPurchaseInput para replay
    status      TEXT NOT NULL DEFAULT 'queued',         -- 'queued' | 'failed' | 'replayed'
    error       TEXT NOT NULL DEFAULT '',               -- mensagem de erro quando status='failed'
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_logs_account_received
    ON webhook_logs(account_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_status
    ON webhook_logs(status);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_source
    ON webhook_logs(source);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_event_id
    ON webhook_logs(event_id) WHERE event_id <> '';
