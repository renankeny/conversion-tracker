-- =====================================================================
-- Migration 0037 — Nomes customizados de eventos por conta
-- Cada empresa (account) pode renomear os eventos que aparecem nos
-- cards/funil da dashboard (ex: InitiateCheckout -> "Iniciou Compra").
-- =====================================================================

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS event_label_overrides JSONB NOT NULL DEFAULT '{}'::jsonb;
