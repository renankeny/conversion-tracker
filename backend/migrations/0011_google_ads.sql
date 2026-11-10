-- =====================================================================
-- 0011_google_ads.sql
-- Adiciona suporte ao Google Ads API v18 (Upload Click Conversions).
--
-- pixels:
--   google_ads_customer_id        - ID do anunciante (123-456-7890 sem traços)
--   google_ads_login_customer_id  - ID do MCC (manager) se aplicável
--   google_ads_conversion_action_id - ID da ConversionAction configurada
--   google_ads_refresh_token      - token longa duração obtido via OAuth
--   google_ads_oauth_email        - email da conta Google que autorizou
--                                   (informativo no painel)
--
-- tracked_users:
--   wbraid - identificador iOS sem GCLID (App campaigns + Web)
--   gbraid - identificador iOS sem GCLID (App-only / iOS 14.5+)
--
-- sent_events.channel ja aceita 'google_ads' (a coluna existe desde
-- a migration 0010).
--
-- Tudo idempotente.
-- =====================================================================

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS google_ads_customer_id          TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS google_ads_login_customer_id    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS google_ads_conversion_action_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS google_ads_refresh_token        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS google_ads_oauth_email          TEXT NOT NULL DEFAULT '';

ALTER TABLE tracked_users
    ADD COLUMN IF NOT EXISTS wbraid TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS gbraid TEXT NOT NULL DEFAULT '';

-- Indice pra buscar por click identifiers (uso pra debug/analise futura)
CREATE INDEX IF NOT EXISTS idx_tracked_users_gclid
    ON tracked_users(gclid) WHERE gclid <> '';
CREATE INDEX IF NOT EXISTS idx_tracked_users_wbraid
    ON tracked_users(wbraid) WHERE wbraid <> '';
CREATE INDEX IF NOT EXISTS idx_tracked_users_gbraid
    ON tracked_users(gbraid) WHERE gbraid <> '';
