-- 0014_add_ga4_fields.sql
-- Adiciona suporte ao Google Analytics 4 (GA4) Measurement Protocol

-- Tabela de pixels
ALTER TABLE pixels ADD COLUMN IF NOT EXISTS ga4_measurement_id VARCHAR(50) DEFAULT '';
ALTER TABLE pixels ADD COLUMN IF NOT EXISTS ga4_api_secret VARCHAR(100) DEFAULT '';

-- Tabela de usuários rastreados
ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS ga4_client_id VARCHAR(100) DEFAULT '';
ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS ga4_session_id VARCHAR(100) DEFAULT '';
