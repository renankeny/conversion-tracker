-- =====================================================================
-- Migration 0050 — Schema Sync, Multi-Tenant Uniqueness & Scale Indexes
-- Reconcilia divergências históricas entre migrações 0014..0049, ajusta
-- constraints multi-tenant e cria índices de escala p/ worker, dashboard e prune.
-- =====================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS schema_migrations (
    filename   TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

SELECT setval('accounts_id_seq', GREATEST((SELECT COALESCE(MAX(id), 1) FROM accounts), 1));

-- 1. Reconciliação de pixels (GA4 e tokens Meta NOT NULL DEFAULT '')
ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS ga4_measurement_id VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ga4_api_secret     VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS meta_permanent_token TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS meta_ad_account_id   TEXT NOT NULL DEFAULT '';

UPDATE pixels SET
    ga4_measurement_id   = COALESCE(ga4_measurement_id, ''),
    ga4_api_secret       = COALESCE(ga4_api_secret, ''),
    meta_permanent_token = COALESCE(meta_permanent_token, ''),
    meta_ad_account_id   = COALESCE(meta_ad_account_id, '')
WHERE ga4_measurement_id IS NULL
   OR ga4_api_secret IS NULL
   OR meta_permanent_token IS NULL
   OR meta_ad_account_id IS NULL;

ALTER TABLE pixels
    ALTER COLUMN ga4_measurement_id   SET DEFAULT '',
    ALTER COLUMN ga4_measurement_id   SET NOT NULL,
    ALTER COLUMN ga4_api_secret       SET DEFAULT '',
    ALTER COLUMN ga4_api_secret       SET NOT NULL,
    ALTER COLUMN meta_permanent_token SET DEFAULT '',
    ALTER COLUMN meta_permanent_token SET NOT NULL,
    ALTER COLUMN meta_ad_account_id   SET DEFAULT '',
    ALTER COLUMN meta_ad_account_id   SET NOT NULL;

-- 2. Reconciliação de tracked_users (TikTok e GA4 NOT NULL DEFAULT '')
ALTER TABLE tracked_users
    ADD COLUMN IF NOT EXISTS ttclid         TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ttp            TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ga4_client_id  VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ga4_session_id VARCHAR(100) NOT NULL DEFAULT '';

UPDATE tracked_users SET
    ttclid         = COALESCE(ttclid, ''),
    ttp            = COALESCE(ttp, ''),
    ga4_client_id  = COALESCE(ga4_client_id, ''),
    ga4_session_id = COALESCE(ga4_session_id, '')
WHERE ttclid IS NULL
   OR ttp IS NULL
   OR ga4_client_id IS NULL
   OR ga4_session_id IS NULL;

ALTER TABLE tracked_users
    ALTER COLUMN ttclid         SET DEFAULT '',
    ALTER COLUMN ttclid         SET NOT NULL,
    ALTER COLUMN ttp            SET DEFAULT '',
    ALTER COLUMN ttp            SET NOT NULL,
    ALTER COLUMN ga4_client_id  SET DEFAULT '',
    ALTER COLUMN ga4_client_id  SET NOT NULL,
    ALTER COLUMN ga4_session_id SET DEFAULT '',
    ALTER COLUMN ga4_session_id SET NOT NULL;

-- 3. Reconciliação de crm_stages, meta_ad_accounts e campaign_insights
ALTER TABLE crm_stages
    ADD COLUMN IF NOT EXISTS google_ads_account_id BIGINT REFERENCES google_ads_accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS google_ads_action_id  TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS meta_ad_accounts (
    id            BIGSERIAL PRIMARY KEY,
    account_id    BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    label         VARCHAR(255) NOT NULL,
    ad_account_id VARCHAR(50) NOT NULL,
    ads_token     TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_meta_ad_accounts_account_id ON meta_ad_accounts(account_id);

CREATE TABLE IF NOT EXISTS campaign_insights (
    id            BIGSERIAL PRIMARY KEY,
    date          DATE NOT NULL,
    ad_account_id VARCHAR(50) NOT NULL,
    campaign_id   VARCHAR(50) NOT NULL,
    campaign_name VARCHAR(255) NOT NULL,
    spend         NUMERIC(15, 2) NOT NULL DEFAULT 0,
    clicks        INT NOT NULL DEFAULT 0,
    impressions   INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(date, campaign_id)
);

CREATE INDEX IF NOT EXISTS idx_campaign_insights_campaign_id ON campaign_insights(campaign_id);

-- 4. Reconciliação de whatsapp_instances (caso criada originalmente pela 0014)
ALTER TABLE whatsapp_instances
    ADD COLUMN IF NOT EXISTS server_id         BIGINT REFERENCES whatsapp_servers(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS instance_name     TEXT,
    ADD COLUMN IF NOT EXISTS share_token       TEXT,
    ADD COLUMN IF NOT EXISTS connection_status TEXT NOT NULL DEFAULT 'CLOSED';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'whatsapp_instances' AND column_name = 'webhook_secret'
    ) THEN
        ALTER TABLE whatsapp_instances ALTER COLUMN webhook_secret SET DEFAULT '';
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_whatsapp_instances_instance_name
    ON whatsapp_instances(instance_name)
    WHERE instance_name IS NOT NULL AND instance_name <> '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_whatsapp_instances_share_token
    ON whatsapp_instances(share_token)
    WHERE share_token IS NOT NULL AND share_token <> '';

-- 5. Garantir ON DELETE CASCADE em google_ads_accounts.account_id
ALTER TABLE google_ads_accounts DROP CONSTRAINT IF EXISTS google_ads_accounts_account_id_fkey;
ALTER TABLE google_ads_accounts
    ADD CONSTRAINT google_ads_accounts_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;

-- 6. Unicidade Multi-Tenant em tracked_users: UNIQUE (account_id, external_id)
ALTER TABLE tracked_users DROP CONSTRAINT IF EXISTS tracked_users_external_id_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_tracked_users_account_external'
    ) THEN
        ALTER TABLE tracked_users
            ADD CONSTRAINT uq_tracked_users_account_external UNIQUE (account_id, external_id);
    END IF;
END $$;

-- 7. Unicidade Multi-Tenant em url_mappings: UNIQUE (account_id, source)
DELETE FROM url_mappings a
USING (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY account_id, source ORDER BY id DESC) AS rn
    FROM url_mappings
) b
WHERE a.id = b.id AND b.rn > 1;

ALTER TABLE url_mappings DROP CONSTRAINT IF EXISTS url_mappings_source_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_url_mappings_account_source'
    ) THEN
        ALTER TABLE url_mappings
            ADD CONSTRAINT uq_url_mappings_account_source UNIQUE (account_id, source);
    END IF;
END $$;

-- 8. Deduplicação de retries em sent_events + índice único para UPSERT do Worker
DELETE FROM sent_events a
USING (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY account_id, pixel_name, channel, event_id
               ORDER BY CASE WHEN status = 'sent' THEN 0 ELSE 1 END, id DESC
           ) AS rn
    FROM sent_events
) b
WHERE a.id = b.id AND b.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_sent_events_acc_pixel_chan_ev
    ON sent_events(account_id, pixel_name, channel, event_id);

-- 9. Índices de escala (Dashboard, Prune, Falhas, TikTok, Webhooks e WhatsApp)
CREATE INDEX IF NOT EXISTS idx_sent_events_acc_created
    ON sent_events(account_id, created_at DESC)
    INCLUDE (event_name, status, event_id, purchase_value, pixel_name);

CREATE INDEX IF NOT EXISTS idx_sent_events_prune_candidates
    ON sent_events(account_id, created_at ASC)
    WHERE payload IS NOT NULL OR response_body <> '' OR input_json IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sent_events_acc_failed
    ON sent_events(account_id, created_at DESC)
    WHERE status = 'failed';

CREATE INDEX IF NOT EXISTS idx_tracked_users_ttclid
    ON tracked_users(ttclid)
    WHERE ttclid <> '';

CREATE INDEX IF NOT EXISTS idx_webhook_logs_acc_failed_rec
    ON webhook_logs(account_id, received_at DESC)
    WHERE status = 'failed';

CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_tracked_user
    ON whatsapp_chats(tracked_user_id)
    WHERE tracked_user_id IS NOT NULL;

COMMIT;

