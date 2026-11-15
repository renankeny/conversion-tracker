-- =====================================================================
-- Migração 0060: Nome descritivo da conta selecionada no Google Ads
-- =====================================================================

ALTER TABLE google_ads_accounts
    ADD COLUMN IF NOT EXISTS customer_name TEXT NOT NULL DEFAULT '';
