-- =====================================================================
-- Migration 0054 — Multi-Provedor WhatsApp (Evolution Go, API Premium, Uazapi)
-- Permite conexões sem servidor Evolution Go obrigatório e armazena
-- o provedor selecionado ('evolution_go', 'api_premium', 'uazapi') e URL externa.
-- =====================================================================

ALTER TABLE whatsapp_instances
    ALTER COLUMN server_id DROP NOT NULL;

ALTER TABLE whatsapp_instances
    ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'evolution_go',
    ADD COLUMN IF NOT EXISTS external_api_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS share_token TEXT UNIQUE;

