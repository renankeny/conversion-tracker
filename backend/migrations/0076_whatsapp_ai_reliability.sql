-- =====================================================================
-- 0076_whatsapp_ai_reliability.sql
-- Telemetria real da última execução da WhatsApp IA (para diagnóstico
-- visível no painel Whitelabel) e saneamento de account_id legado em
-- whatsapp_chats.
-- =====================================================================

ALTER TABLE account_ai_settings
    ADD COLUMN IF NOT EXISTS last_error           TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_analyzed_count  INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_qualified_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_sales_count     INT NOT NULL DEFAULT 0;

UPDATE whatsapp_chats c
SET account_id = i.account_id
FROM whatsapp_instances i
WHERE c.instance_id = i.id
  AND c.account_id IS NULL;
