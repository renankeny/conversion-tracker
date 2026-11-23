-- =====================================================================
-- 0068_pwa_push_notifications.sql
-- PWA Whitelabel & Notificações Web Push (VAPID automático, preferências,
-- múltiplos dispositivos e histórico enxuto limitado às 10 mais recentes)
-- =====================================================================

CREATE TABLE IF NOT EXISTS push_vapid_config (
    id                    SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    public_key            TEXT NOT NULL,
    private_key_encrypted TEXT NOT NULL,
    subject               TEXT NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_notification_preferences (
    user_id                      BIGINT PRIMARY KEY REFERENCES admin_users(id) ON DELETE CASCADE,
    account_scope_mode           VARCHAR(16) NOT NULL DEFAULT 'all',
    enabled_account_ids          BIGINT[] NOT NULL DEFAULT '{}',
    whatsapp_connection          BOOLEAN NOT NULL DEFAULT TRUE,
    lead_created                 BOOLEAN NOT NULL DEFAULT TRUE,
    ai_lead_qualified            BOOLEAN NOT NULL DEFAULT TRUE,
    sale_approved                BOOLEAN NOT NULL DEFAULT TRUE,
    meta_account_lifecycle       BOOLEAN NOT NULL DEFAULT TRUE,
    google_ads_account_lifecycle BOOLEAN NOT NULL DEFAULT TRUE,
    pixel_lifecycle              BOOLEAN NOT NULL DEFAULT TRUE,
    event_dispatch_error         BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_push_subscriptions (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    endpoint     TEXT NOT NULL UNIQUE,
    p256dh       TEXT NOT NULL,
    auth_secret  TEXT NOT NULL,
    device_label TEXT NOT NULL DEFAULT '',
    user_agent   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_push_subscriptions_user
    ON user_push_subscriptions(user_id, id DESC);

CREATE TABLE IF NOT EXISTS user_notifications (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    account_id   BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    account_name TEXT NOT NULL DEFAULT '',
    category     VARCHAR(40) NOT NULL,
    title        TEXT NOT NULL,
    body         TEXT NOT NULL,
    action_url   TEXT NOT NULL DEFAULT '/',
    is_read      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_notifications_user_created
    ON user_notifications(user_id, id DESC);
