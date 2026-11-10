-- =====================================================================
-- Migração 0053: Fotos de Contas de Anúncios, Páginas e Perfis OAuth
-- =====================================================================

ALTER TABLE meta_accounts
    ADD COLUMN IF NOT EXISTS oauth_picture_url        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ad_account_picture_url   TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS default_page_picture_url TEXT NOT NULL DEFAULT '';

ALTER TABLE google_ads_accounts
    ADD COLUMN IF NOT EXISTS oauth_picture_url TEXT NOT NULL DEFAULT '';

