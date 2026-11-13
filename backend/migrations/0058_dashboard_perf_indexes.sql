-- Migração 0058: Índices compostos para performance das agregações da Dashboard

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat_inbound_ts
    ON whatsapp_messages(chat_id, timestamp DESC)
    WHERE is_from_me = FALSE;

CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_acc_created
    ON whatsapp_chats(account_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_crm_leads_acc_created
    ON crm_leads(account_id, created_at DESC);

