-- =====================================================================
-- 0029_webhook_logs_raw_payload.sql
-- Adiciona raw_payload e custom_webhook_id ao webhook_logs para
-- auditoria dos checkouts universais.
-- =====================================================================

ALTER TABLE webhook_logs
    ADD COLUMN IF NOT EXISTS raw_payload   JSONB,
    ADD COLUMN IF NOT EXISTS custom_webhook_id TEXT;

CREATE INDEX IF NOT EXISTS idx_webhook_logs_custom_webhook_id
    ON webhook_logs(custom_webhook_id) WHERE custom_webhook_id IS NOT NULL;
