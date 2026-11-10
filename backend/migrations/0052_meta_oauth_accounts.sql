-- =====================================================================
-- Migração 0052: Contas Meta Ads OAuth Blindadas (Vault Proxy)
-- =====================================================================

CREATE TABLE IF NOT EXISTS meta_accounts (
    id                    BIGSERIAL PRIMARY KEY,
    account_id            BIGINT NOT NULL DEFAULT 1 REFERENCES accounts(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    vault_token_encrypted TEXT NOT NULL DEFAULT '',
    oauth_user_id         TEXT NOT NULL DEFAULT '',
    oauth_name            TEXT NOT NULL DEFAULT '',
    oauth_email           TEXT NOT NULL DEFAULT '',
    token_expires_at      TIMESTAMPTZ,
    ad_account_id         TEXT NOT NULL DEFAULT '',
    ad_account_name       TEXT NOT NULL DEFAULT '',
    default_page_id       TEXT NOT NULL DEFAULT '',
    default_page_name     TEXT NOT NULL DEFAULT '',
    enabled               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_meta_accounts_account_id
    ON meta_accounts(account_id);

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS meta_account_id BIGINT REFERENCES meta_accounts(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_pixels_meta_account_id
    ON pixels(meta_account_id) WHERE meta_account_id IS NOT NULL;

