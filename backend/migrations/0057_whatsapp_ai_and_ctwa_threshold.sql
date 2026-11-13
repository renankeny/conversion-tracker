-- Migração 0057: Analista de Conversas WhatsApp com IA (OpenAI) + Regra de disparo LeadSubmitted CTWA (1ª vs 3ª mensagem)

ALTER TABLE whatsapp_instances
    ADD COLUMN IF NOT EXISTS ctwa_min_inbound_messages SMALLINT NOT NULL DEFAULT 1;

ALTER TABLE whatsapp_chats
    ADD COLUMN IF NOT EXISTS ai_qualified            BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS ai_last_analyzed_msg_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS ai_last_analyzed_at     TIMESTAMPTZ;

ALTER TABLE whatsapp_messages
    ADD COLUMN IF NOT EXISTS ai_transcript TEXT;

CREATE TABLE IF NOT EXISTS account_ai_settings (
    account_id               BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    enabled                  BOOLEAN NOT NULL DEFAULT FALSE,
    model                    TEXT NOT NULL DEFAULT 'gpt-4o-mini',
    qualified_stage_id       BIGINT REFERENCES crm_stages(id) ON DELETE SET NULL,
    sale_stage_id            BIGINT REFERENCES crm_stages(id) ON DELETE SET NULL,
    base_prompt              TEXT NOT NULL DEFAULT '',
    learned_rules            JSONB NOT NULL DEFAULT '[]'::jsonb,
    last_run_at              TIMESTAMPTZ,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS whatsapp_ai_events (
    id                   BIGSERIAL PRIMARY KEY,
    account_id           BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    chat_id              BIGINT NOT NULL REFERENCES whatsapp_chats(id) ON DELETE CASCADE,
    crm_lead_id          BIGINT REFERENCES crm_leads(id) ON DELETE SET NULL,
    tracked_user_id      BIGINT REFERENCES tracked_users(id) ON DELETE SET NULL,
    event_type           VARCHAR(20) NOT NULL,
    evidence_message_id  BIGINT REFERENCES whatsapp_messages(id) ON DELETE SET NULL,
    amount               NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency             VARCHAR(10) NOT NULL DEFAULT 'BRL',
    ai_reason            TEXT NOT NULL DEFAULT '',
    crm_prev_stage_id    BIGINT REFERENCES crm_stages(id) ON DELETE SET NULL,
    crm_new_stage_id     BIGINT REFERENCES crm_stages(id) ON DELETE SET NULL,
    crm_prev_value       NUMERIC(12,2) NOT NULL DEFAULT 0,
    dispatched_event_id  TEXT NOT NULL DEFAULT '',
    deduplicated         BOOLEAN NOT NULL DEFAULT FALSE,
    status               VARCHAR(20) NOT NULL DEFAULT 'active',
    contest_reason       TEXT NOT NULL DEFAULT '',
    contested_by_user_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,
    contested_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_ai_events_chat
    ON whatsapp_ai_events(chat_id, id ASC);
CREATE INDEX IF NOT EXISTS idx_whatsapp_ai_events_account
    ON whatsapp_ai_events(account_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS uq_whatsapp_ai_events_active_msg
    ON whatsapp_ai_events(chat_id, event_type, evidence_message_id)
    WHERE status = 'active' AND evidence_message_id IS NOT NULL;

