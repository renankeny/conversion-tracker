-- =====================================================================
-- 0013_google_ads_conversion_actions.sql
-- Cada conta Google Ads pode ter MULTIPLAS Conversion Actions (uma por
-- tipo de evento: Purchase/Vendas, Lead, CompleteRegistration, etc).
-- Cada ConversionAction tem seu proprio ID gerado no painel do Google
-- Ads e e referenciada no payload da API quando subimos a conversao
-- offline (cada conversao precisa apontar pra UMA action especifica).
--
-- Modelado como JSONB array dentro de google_ads_accounts pra evitar
-- tabela auxiliar (relacao 1:N pequena, sem queries cross-conta).
--
-- Formato de cada item:
--   {
--     "event_type": "Purchase",        -- nome do evento TrackFind
--     "conversion_action_id": "12345", -- ID interno do Google Ads
--     "name": "Vendas",                 -- label pro painel
--     "enabled": true
--   }
--
-- O campo legado conversion_action_id (string unica) continua existindo
-- pra compat reversa, mas o backend nao usa mais.
-- =====================================================================

ALTER TABLE google_ads_accounts
    ADD COLUMN IF NOT EXISTS conversion_actions JSONB NOT NULL DEFAULT '[]'::jsonb;
