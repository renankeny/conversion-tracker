-- =====================================================================
-- Migration 0064: Nome e Customer ID especifico da Conta Google Ads no Pixel
-- Permite escolher no Pixel tanto a Conta Google (OAuth) quanto a Conta
-- de Anuncios Google Ads especifica (Customer ID + MCC Login ID) para
-- exibicao de resultados na Dashboard e conversoes offline.
-- =====================================================================

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS google_ads_customer_name TEXT NOT NULL DEFAULT '';

UPDATE pixels p
SET
    google_ads_customer_id = CASE
        WHEN p.google_ads_customer_id = '' THEN ga.customer_id
        ELSE p.google_ads_customer_id
    END,
    google_ads_customer_name = CASE
        WHEN p.google_ads_customer_name = '' THEN COALESCE(NULLIF(ga.customer_name, ''), ga.name)
        ELSE p.google_ads_customer_name
    END,
    google_ads_login_customer_id = CASE
        WHEN p.google_ads_customer_id = '' AND p.google_ads_login_customer_id = '' THEN ga.login_customer_id
        ELSE p.google_ads_login_customer_id
    END
FROM google_ads_accounts ga
WHERE p.google_ads_account_id = ga.id
  AND ga.customer_id <> '';
