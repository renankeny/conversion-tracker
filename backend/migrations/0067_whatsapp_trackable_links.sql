-- =====================================================================
-- 0067_whatsapp_trackable_links.sql
-- Links Rastreáveis de WhatsApp por Pixel (Bio, Stories, YouTube, Ads)
-- =====================================================================

CREATE TABLE IF NOT EXISTS whatsapp_trackable_links (
    id                   BIGSERIAL PRIMARY KEY,
    account_id           BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    pixel_id             BIGINT NOT NULL REFERENCES pixels(id) ON DELETE CASCADE,
    slug                 VARCHAR(80) NOT NULL UNIQUE,
    name                 TEXT NOT NULL,
    routing_mode         VARCHAR(24) NOT NULL DEFAULT 'auto_pixel',
    whatsapp_instance_id BIGINT REFERENCES whatsapp_instances(id) ON DELETE SET NULL,
    fallback_phone       TEXT NOT NULL DEFAULT '',
    prefill_message      TEXT NOT NULL DEFAULT 'Olá! Gostaria de mais informações.',
    utm_source           TEXT NOT NULL DEFAULT '',
    utm_medium           TEXT NOT NULL DEFAULT '',
    utm_campaign         TEXT NOT NULL DEFAULT '',
    utm_content          TEXT NOT NULL DEFAULT '',
    utm_term             TEXT NOT NULL DEFAULT '',
    send_events_enabled  BOOLEAN NOT NULL DEFAULT TRUE,
    event_trigger_mode   VARCHAR(24) NOT NULL DEFAULT 'on_first_message',
    event_name           VARCHAR(40) NOT NULL DEFAULT 'Lead',
    send_meta            BOOLEAN NOT NULL DEFAULT TRUE,
    send_tiktok          BOOLEAN NOT NULL DEFAULT TRUE,
    send_google_ads      BOOLEAN NOT NULL DEFAULT TRUE,
    send_ga4             BOOLEAN NOT NULL DEFAULT TRUE,
    enabled              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_trackable_links_pixel
    ON whatsapp_trackable_links(pixel_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_whatsapp_trackable_links_account
    ON whatsapp_trackable_links(account_id, id DESC);

CREATE TABLE IF NOT EXISTS whatsapp_link_clicks (
    id                   BIGSERIAL PRIMARY KEY,
    account_id           BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    pixel_id             BIGINT NOT NULL REFERENCES pixels(id) ON DELETE CASCADE,
    link_id              BIGINT NOT NULL REFERENCES whatsapp_trackable_links(id) ON DELETE CASCADE,
    tracked_user_id      BIGINT REFERENCES tracked_users(id) ON DELETE SET NULL,
    resolved_instance_id BIGINT REFERENCES whatsapp_instances(id) ON DELETE SET NULL,
    protocol_code        VARCHAR(36) NOT NULL DEFAULT '',
    client_ip            TEXT NOT NULL DEFAULT '',
    is_bot               BOOLEAN NOT NULL DEFAULT FALSE,
    matched_chat_id      BIGINT REFERENCES whatsapp_chats(id) ON DELETE SET NULL,
    click_event_sent     BOOLEAN NOT NULL DEFAULT FALSE,
    message_event_sent   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_link_created
    ON whatsapp_link_clicks(link_id, created_at DESC)
    WHERE is_bot = FALSE;

CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_acc_created
    ON whatsapp_link_clicks(account_id, created_at DESC)
    WHERE is_bot = FALSE;

CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_pending_inst
    ON whatsapp_link_clicks(resolved_instance_id, created_at DESC)
    WHERE matched_chat_id IS NULL AND is_bot = FALSE;

CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_proto
    ON whatsapp_link_clicks(protocol_code)
    WHERE protocol_code <> '';

ALTER TABLE tracked_users
    ADD COLUMN IF NOT EXISTS trackable_link_id BIGINT REFERENCES whatsapp_trackable_links(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_tracked_users_trackable_link
    ON tracked_users(trackable_link_id)
    WHERE trackable_link_id IS NOT NULL;

ALTER TABLE whatsapp_chats
    ADD COLUMN IF NOT EXISTS trackable_link_id BIGINT REFERENCES whatsapp_trackable_links(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_trackable_link
    ON whatsapp_chats(trackable_link_id, created_at DESC)
    WHERE trackable_link_id IS NOT NULL;
