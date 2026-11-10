-- =====================================================================
-- Migration 0038 — Dashboard por pixel + link público de visualização
-- Cada pixel passa a ter sua própria config de dashboard (cards, ordem,
-- funil, nomes customizados) e pode gerar um link público view-only.
-- =====================================================================

ALTER TABLE pixels ADD COLUMN IF NOT EXISTS dashboard_config JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE pixels ADD COLUMN IF NOT EXISTS dashboard_share_token TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_pixels_dashboard_share_token
    ON pixels(dashboard_share_token) WHERE dashboard_share_token <> '';
