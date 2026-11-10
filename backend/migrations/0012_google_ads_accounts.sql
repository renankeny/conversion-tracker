-- =====================================================================
-- 0012_google_ads_accounts.sql
-- Separa Google Ads em entidade propria. Antes a configuracao ficava
-- nos campos google_ads_* da tabela pixels (migration 0011), agora
-- vira uma tabela independente porque conceitualmente Google Ads
-- trabalha por GCLID (cada conversao tem um click_id diferente) e nao
-- por pixel (Meta/TikTok). Uma conta Google Ads pode atender varios
-- pixels.
--
-- Os campos google_ads_* em pixels seguem existindo (additive only —
-- backend para de gravar/ler, mas nao dropamos pra nao ser destrutivo).
-- =====================================================================

CREATE TABLE IF NOT EXISTS google_ads_accounts (
    id                       BIGSERIAL PRIMARY KEY,
    account_id               BIGINT NOT NULL DEFAULT 1,
    name                     TEXT NOT NULL,
    customer_id              TEXT NOT NULL DEFAULT '',
    login_customer_id        TEXT NOT NULL DEFAULT '',
    conversion_action_id     TEXT NOT NULL DEFAULT '',
    refresh_token            TEXT NOT NULL DEFAULT '',
    oauth_email              TEXT NOT NULL DEFAULT '',
    enabled                  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_google_ads_accounts_account_id
    ON google_ads_accounts(account_id);

-- Pixel pode opcionalmente apontar pra uma conta Google Ads.
-- ON DELETE SET NULL pra nao perder o pixel se a conta for removida.
ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS google_ads_account_id BIGINT
        REFERENCES google_ads_accounts(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_pixels_google_ads_account_id
    ON pixels(google_ads_account_id) WHERE google_ads_account_id IS NOT NULL;
