-- =====================================================================
-- Migration 0036 — WhatsApp LID (Linked ID) em tracked_users
-- Alguns JIDs do WhatsApp expõem um "LID" no lugar do telefone real
-- (formato "<id>@lid"). Guardamos separado de `phone` — nunca inferimos
-- um pelo outro.
-- =====================================================================

ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS wa_lid TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_tracked_users_wa_lid
    ON tracked_users(wa_lid) WHERE wa_lid <> '';
