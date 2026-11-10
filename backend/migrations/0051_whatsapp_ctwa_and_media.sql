-- =====================================================================
-- 0051_whatsapp_ctwa_and_media.sql
-- Evolui as tabelas de WhatsApp para multi-tenant real por conta,
-- mapeamento determinístico de @lid, atribuição de portais (CTWA / LP /
-- Google Ads / Bio / Orgânico), vínculo direto com CRM e retenção
-- seletiva de mídias no MinIO (14 dias sem apagar texto).
-- =====================================================================

-- 1. whatsapp_servers: de global único para multi-tenant por conta (account_id)
DROP INDEX IF EXISTS idx_whatsapp_servers_active_global;

ALTER TABLE whatsapp_servers
    ADD COLUMN IF NOT EXISTS account_id BIGINT NOT NULL DEFAULT 1
        REFERENCES accounts(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_whatsapp_servers_account
    ON whatsapp_servers(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_whatsapp_servers_account_active
    ON whatsapp_servers(account_id) WHERE is_active = TRUE;

-- 2. whatsapp_instances: UUID e token da Evolution Go, segredo de webhook e vínculo CRM
ALTER TABLE whatsapp_instances
    ADD COLUMN IF NOT EXISTS evolution_uuid           TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS instance_token_encrypted TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS webhook_secret           TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS owner_jid                TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS profile_name             TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS profile_pic_url          TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS auto_create_crm_lead     BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS default_crm_stage_id     BIGINT REFERENCES crm_stages(id) ON DELETE SET NULL;

UPDATE whatsapp_instances
SET webhook_secret = md5(random()::text || clock_timestamp()::text || id::text)
WHERE webhook_secret = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_instances_webhook_secret
    ON whatsapp_instances(webhook_secret) WHERE webhook_secret <> '';

-- 3. whatsapp_lid_mappings: resolução bidirecional @lid <-> @s.whatsapp.net por conta
CREATE TABLE IF NOT EXISTS whatsapp_lid_mappings (
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    lid_jid    TEXT NOT NULL,
    phone_jid  TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (account_id, lid_jid)
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_lid_mappings_phone
    ON whatsapp_lid_mappings(account_id, phone_jid);

-- 4. whatsapp_chats: atribuição completa de portais/anúncios e vínculo CRM
ALTER TABLE whatsapp_chats
    ADD COLUMN IF NOT EXISTS account_id       BIGINT REFERENCES accounts(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS crm_lead_id      BIGINT REFERENCES crm_leads(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS phone            TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lid_jid          TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS origin_type      VARCHAR(30) NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS ctwa_clid        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_id        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS page_id          TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_app       VARCHAR(30) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ad_name          TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS adset_id         TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS adset_name       TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS campaign_id      TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS campaign_name    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ad_thumbnail_url TEXT NOT NULL DEFAULT '';

UPDATE whatsapp_chats c
SET account_id = i.account_id
FROM whatsapp_instances i
WHERE c.instance_id = i.id AND c.account_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_account_tracked
    ON whatsapp_chats(account_id, tracked_user_id) WHERE tracked_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_account_crm_lead
    ON whatsapp_chats(account_id, crm_lead_id) WHERE crm_lead_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_account_phone
    ON whatsapp_chats(account_id, phone) WHERE phone <> '';
CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_account_lid
    ON whatsapp_chats(account_id, lid_jid) WHERE lid_jid <> '';

-- 5. whatsapp_messages: armazenamento S3/MinIO e expiração seletiva de mídia (14 dias)
ALTER TABLE whatsapp_messages
    ADD COLUMN IF NOT EXISTS media_object_key   TEXT,
    ADD COLUMN IF NOT EXISTS media_status       VARCHAR(20) NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS media_size_bytes   BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS media_duration_sec INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS media_filename     TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS media_expires_at   TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_media_expiry
    ON whatsapp_messages(media_expires_at) WHERE media_status = 'ready';

