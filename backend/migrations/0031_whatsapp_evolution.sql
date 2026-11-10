-- =====================================================================
-- Migration 0031 — WhatsApp Evolution GO & Chat Mirroring
-- Adiciona suporte a multi-servidores, chats (conversas) e mensagens
-- =====================================================================

-- 1. Servidores WhatsApp (Evolution GO)
CREATE TABLE IF NOT EXISTS whatsapp_servers (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL DEFAULT 1 REFERENCES accounts(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    api_key TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Garante que apenas um servidor possa ser o "Ativo" por conta
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='whatsapp_servers' AND column_name='account_id') THEN
        EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_servers_active ON whatsapp_servers(account_id) WHERE is_active = TRUE;';
    END IF;
END
$$;

-- 2. Recriar whatsapp_instances para a nova arquitetura
CREATE TABLE IF NOT EXISTS whatsapp_instances (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL DEFAULT 1 REFERENCES accounts(id) ON DELETE RESTRICT,
    pixel_id BIGINT NOT NULL REFERENCES pixels(id) ON DELETE CASCADE,
    server_id BIGINT NOT NULL REFERENCES whatsapp_servers(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    instance_name TEXT NOT NULL UNIQUE, -- Nome do dispositivo interno gerado
    share_token TEXT UNIQUE, -- Token para o link publico
    connection_status TEXT NOT NULL DEFAULT 'CLOSED',
    
    -- Gatilhos NLP / Regex
    purchase_trigger TEXT NOT NULL DEFAULT '',
    lead_trigger TEXT NOT NULL DEFAULT '',
    custom_event_name TEXT NOT NULL DEFAULT '',
    custom_event_trigger TEXT NOT NULL DEFAULT '',
    
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_instances_pixel ON whatsapp_instances(pixel_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_instances_account ON whatsapp_instances(account_id);

-- 3. Espelhamento de Conversas (Chats)
CREATE TABLE IF NOT EXISTS whatsapp_chats (
    id BIGSERIAL PRIMARY KEY,
    instance_id BIGINT NOT NULL REFERENCES whatsapp_instances(id) ON DELETE CASCADE,
    remote_jid TEXT NOT NULL,
    tracked_user_id BIGINT REFERENCES tracked_users(id) ON DELETE SET NULL,
    push_name TEXT NOT NULL DEFAULT '',
    last_message_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unread_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(instance_id, remote_jid)
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_instance ON whatsapp_chats(instance_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_last_msg ON whatsapp_chats(last_message_time DESC);

-- 4. Histórico de Mensagens
CREATE TABLE IF NOT EXISTS whatsapp_messages (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES whatsapp_chats(id) ON DELETE CASCADE,
    whatsapp_message_id TEXT NOT NULL,
    from_me BOOLEAN NOT NULL DEFAULT FALSE,
    message_type TEXT NOT NULL DEFAULT 'text',
    content TEXT NOT NULL DEFAULT '',
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(chat_id, whatsapp_message_id)
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat ON whatsapp_messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp ASC);
