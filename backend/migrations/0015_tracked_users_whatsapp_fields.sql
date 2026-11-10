-- =====================================================================
-- Migration 0015 — Tracked Users WhatsApp Fields
-- Adiciona campos WhatsApp à tabela tracked_users para armazenar
-- informações de Click to WhatsApp (CTWA) e identificação do chat.
-- =====================================================================

ALTER TABLE tracked_users
    ADD COLUMN IF NOT EXISTS ctwa_clid     TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS wa_chat_id    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS wa_name       TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS wa_ad_id      TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS wa_source_app TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_tracked_users_wa_chat_id
    ON tracked_users(wa_chat_id) WHERE wa_chat_id <> '';
CREATE INDEX IF NOT EXISTS idx_tracked_users_ctwa_clid
    ON tracked_users(ctwa_clid) WHERE ctwa_clid <> '';
