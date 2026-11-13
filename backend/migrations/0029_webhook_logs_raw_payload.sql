-- =====================================================================
-- 0029_webhook_logs_raw_payload.sql
-- Adiciona raw_payload e custom_webhook_id ao webhook_logs para
-- auditoria dos checkouts universais.
--
-- NOTA DE COMPATIBILIDADE: O prefixo numérico "0029_" é intencionalmente
-- compartilhado com 0029_platform_tokens.sql por razões históricas.
-- NÃO renomeie este arquivo: o runner (db.Migrate) ordena por nome completo
-- do arquivo (filename PRIMARY KEY em schema_migrations), e renomeá-lo faria
-- instâncias já em produção reexecutarem a migração desnecessariamente.
-- =====================================================================

ALTER TABLE webhook_logs
    ADD COLUMN IF NOT EXISTS raw_payload   JSONB,
    ADD COLUMN IF NOT EXISTS custom_webhook_id TEXT;

CREATE INDEX IF NOT EXISTS idx_webhook_logs_custom_webhook_id
    ON webhook_logs(custom_webhook_id) WHERE custom_webhook_id IS NOT NULL;
