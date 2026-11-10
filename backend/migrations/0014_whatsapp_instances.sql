-- =====================================================================
-- Migration 0014 — WhatsApp Instances
-- Tabela para gerenciar instâncias da uazAPI Premium, permitindo cada
-- conta TrackFind cadastrar suas próprias instâncias WhatsApp e vincular
-- a um pixel para envio de conversões.
-- =====================================================================

CREATE TABLE IF NOT EXISTS whatsapp_instances (
    id                   BIGSERIAL PRIMARY KEY,
    account_id           BIGINT NOT NULL DEFAULT 1
                          REFERENCES accounts(id) ON DELETE RESTRICT,
    pixel_id             BIGINT NOT NULL
                          REFERENCES pixels(id) ON DELETE CASCADE,
    name                 TEXT NOT NULL,                  -- "WhatsApp Cliente X"
    uaz_url              TEXT NOT NULL DEFAULT '',       -- https://host.uazapi.com
    uaz_token            TEXT NOT NULL DEFAULT '',       -- token API Premium
    webhook_secret       TEXT NOT NULL,                  -- segredo p/ validar webhooks
    sync_to_uazapi       BOOLEAN NOT NULL DEFAULT TRUE,  -- chama /chat/editLead

    -- Triggers de mensagem do dono (admin), iguais ao GTM
    purchase_trigger     TEXT NOT NULL DEFAULT '',
    lead_trigger         TEXT NOT NULL DEFAULT '',
    custom_event_name    TEXT NOT NULL DEFAULT '',
    custom_event_trigger TEXT NOT NULL DEFAULT '',

    enabled              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_instances_pixel
    ON whatsapp_instances(pixel_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_instances_account
    ON whatsapp_instances(account_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_instances_enabled
    ON whatsapp_instances(enabled) WHERE enabled = TRUE;
