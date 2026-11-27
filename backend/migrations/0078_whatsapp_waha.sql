-- =====================================================================
-- 0078_whatsapp_waha.sql
-- Suporte ao servidor global WAHA API lado a lado com Evolution Go
-- na tabela whatsapp_servers (Conexão Direta gerenciada).
-- =====================================================================

ALTER TABLE whatsapp_servers
    ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'evolution_go';

UPDATE whatsapp_servers
SET provider = 'evolution_go'
WHERE provider IS NULL OR BTRIM(provider) = '';

DROP INDEX IF EXISTS uq_whatsapp_servers_account_active;

CREATE UNIQUE INDEX IF NOT EXISTS uq_whatsapp_servers_account_provider_active
    ON whatsapp_servers(account_id, provider) WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_whatsapp_servers_provider_active
    ON whatsapp_servers(provider, is_active);
