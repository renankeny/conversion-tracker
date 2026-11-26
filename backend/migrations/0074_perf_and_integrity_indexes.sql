-- =====================================================================
-- 0074_perf_and_integrity_indexes.sql
-- Correção de constraints multi-tenant, índices compostos/funcionais de alta
-- escala, índices de Foreign Keys e remoção de índices prefixo-redundantes.
-- =====================================================================

-- 1. Corrigir unicidade global indevida de id_wpp para escopo por empresa
DROP INDEX IF EXISTS uq_tracked_users_id_wpp;
CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_id_wpp
    ON tracked_users(account_id, id_wpp) WHERE id_wpp <> '';

-- 2. Índice funcional canônico de grupo para eliminar Full Scans e Disk Sorts
-- em /users, /events (LTV/Buyer) e Auditoria da Dashboard
CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_gkey_updated
    ON tracked_users (account_id, (COALESCE(group_id, -id)), updated_at DESC, id DESC);

-- 3. Tie-breaker composto em tracked_users (corrige colisão de nome entre 0045 e 0063)
CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_updated_id
    ON tracked_users (account_id, updated_at DESC, id DESC);

-- 4. Índices compostos críticos em whatsapp_messages (histórico paginado, SLA e IA)
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat_ts_id
    ON whatsapp_messages (chat_id, timestamp DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat_id_desc
    ON whatsapp_messages (chat_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_media_exp_eff
    ON whatsapp_messages ((COALESCE(media_expires_at, timestamp)))
    WHERE media_status = 'ready';

-- 5. Garantir account_id populado em whatsapp_chats para ativar os índices de conta
UPDATE whatsapp_chats c
SET account_id = i.account_id
FROM whatsapp_instances i
WHERE c.instance_id = i.id AND c.account_id IS NULL;

-- 6. Índices líderes de Foreign Keys (elimina Seq Scans em DELETE CASCADE / SET NULL e merges)
CREATE INDEX IF NOT EXISTS idx_groups_account_id
    ON groups(account_id);
CREATE INDEX IF NOT EXISTS idx_crm_leads_tracked_user_id
    ON crm_leads(tracked_user_id);
CREATE INDEX IF NOT EXISTS idx_crm_leads_group_id
    ON crm_leads(group_id) WHERE group_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_crm_lead_tags_tag_id
    ON crm_lead_tags(crm_tag_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_crm_lead_id
    ON whatsapp_chats(crm_lead_id) WHERE crm_lead_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_ai_events_crm_lead
    ON whatsapp_ai_events(crm_lead_id) WHERE crm_lead_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_ai_events_tracked_user
    ON whatsapp_ai_events(tracked_user_id) WHERE tracked_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_ai_events_evidence_msg
    ON whatsapp_ai_events(evidence_message_id) WHERE evidence_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_account_all
    ON whatsapp_link_clicks(account_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_pixel_all
    ON whatsapp_link_clicks(pixel_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_link_all
    ON whatsapp_link_clicks(link_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_tracked_user
    ON whatsapp_link_clicks(tracked_user_id) WHERE tracked_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_matched_chat
    ON whatsapp_link_clicks(matched_chat_id) WHERE matched_chat_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_link_clicks_resolved_inst
    ON whatsapp_link_clicks(resolved_instance_id) WHERE resolved_instance_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_instances_server_id
    ON whatsapp_instances(server_id) WHERE server_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_trackable_links_instance
    ON whatsapp_trackable_links(whatsapp_instance_id) WHERE whatsapp_instance_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_notifications_account_id
    ON user_notifications(account_id) WHERE account_id IS NOT NULL;

-- 7. Índices parciais para eliminar Seq Scan nas rotinas de manutenção (Prune)
CREATE INDEX IF NOT EXISTS idx_webhook_logs_prune_candidates
    ON webhook_logs(received_at ASC)
    WHERE raw_payload IS NOT NULL OR input_json IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sent_events_prune_global
    ON sent_events(created_at ASC)
    WHERE payload IS NOT NULL OR response_body <> '' OR input_json IS NOT NULL;

-- 8. Remoção segura de índices 100% prefixo-duplicados (reduz write amplification em Upserts)
DROP INDEX IF EXISTS idx_accounts_slug;
DROP INDEX IF EXISTS idx_tracked_users_account_id;
DROP INDEX IF EXISTS idx_tracked_users_account_pixel;
DROP INDEX IF EXISTS idx_tracked_users_phone;
DROP INDEX IF EXISTS idx_tracked_users_email;
DROP INDEX IF EXISTS idx_tracked_users_pixel;
DROP INDEX IF EXISTS idx_sent_events_account_id;
DROP INDEX IF EXISTS idx_sent_events_channel;
