-- Migration 0042: Agrupamento de Conversoes (Order Bump e Upsell)

-- Colunas de configuracao em pixels
ALTER TABLE pixels ADD COLUMN IF NOT EXISTS aggregation_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE pixels ADD COLUMN IF NOT EXISTS aggregation_window_minutes INT NOT NULL DEFAULT 10;

-- Colunas para historico e transparencia em sent_events
ALTER TABLE sent_events ADD COLUMN IF NOT EXISTS aggregated_count INT NOT NULL DEFAULT 1;
ALTER TABLE sent_events ADD COLUMN IF NOT EXISTS aggregated_items JSONB;
