-- =====================================================================
-- 0045_performance_indexes.sql
--
-- Otimizações de alto impacto para suportar escala massiva de usuários
-- e requisições simultâneas sem sobrecarregar o PostgreSQL:
--
-- 1. sent_events (tabela mais volumosa do sistema):
--    - idx_sent_events_account_external_created: elimina seq scan ao abrir modal/jornada
--    - idx_sent_events_dedup_sent: index-only scan na checagem de deduplicação (ExistsSent / GetSuccessfulChannels)
--    - idx_sent_events_recent_dedup: busca instantânea (0ms) na prevenção de duplo clique (ExistsRecentEvent)
--    - idx_sent_events_ext_purchase: cálculo instantâneo de LTV e 1ª compra por external_id
--    - idx_sent_events_account_revenue: agregação de receita total no Dashboard em tempo recorde
--
-- 2. tracked_users (visitantes e sessões):
--    - idx_tracked_users_phone_updated / email_updated: busca por telefone/email ordenado por updated_at DESC
--    - idx_tracked_users_acc_phone / acc_email: resolução rápida de identidade em multi-tenant (ResolveAndLink)
--    - idx_tracked_users_acc_updated: acelera paginação da listagem de visitantes
--
-- 3. event_counters:
--    - idx_event_counters_acc_bucket: acelera agregados no Dashboard quando "Todos os Pixels" está selecionado
--
-- 4. crm_leads:
--    - idx_crm_leads_acc_stage_updated: carregamento ultra-rápido dos cards do Kanban por estágio
--
-- 5. webhook_logs:
--    - idx_webhook_logs_account_ext: correlação direta de webhook com external_id
-- =====================================================================

-- ---------------------------------------------------------------------
-- 1. sent_events
-- ---------------------------------------------------------------------

-- Acelera histórico, jornada e linha do tempo de usuários específicos
CREATE INDEX IF NOT EXISTS idx_sent_events_account_external_created
    ON sent_events(account_id, external_id, created_at DESC);

-- Acelera checagem de deduplicação antes do envio (Index-Only Scan)
CREATE INDEX IF NOT EXISTS idx_sent_events_dedup_sent
    ON sent_events(event_id, pixel_name, channel)
    WHERE status = 'sent';

-- Acelera prevenção de duplo-clique / cliques simultâneos no ingestion (ExistsRecentEvent)
CREATE INDEX IF NOT EXISTS idx_sent_events_recent_dedup
    ON sent_events(pixel_name, external_id, event_name, created_at DESC);

-- Acelera cálculo de LTV e primeira compra no modal do usuário (Index-Only Scan)
CREATE INDEX IF NOT EXISTS idx_sent_events_ext_purchase
    ON sent_events(external_id, created_at ASC)
    INCLUDE (purchase_value, event_id)
    WHERE event_name = 'Purchase' AND status = 'sent';

-- Acelera agregação de faturamento e vendas no Dashboard
CREATE INDEX IF NOT EXISTS idx_sent_events_account_revenue
    ON sent_events(account_id, created_at DESC)
    INCLUDE (purchase_value, event_id, pixel_name)
    WHERE event_name = 'Purchase' AND status = 'sent';

-- ---------------------------------------------------------------------
-- 2. tracked_users
-- ---------------------------------------------------------------------

-- Acelera busca de leads por telefone e email com ordenação por updated_at DESC
CREATE INDEX IF NOT EXISTS idx_tracked_users_phone_updated
    ON tracked_users(phone, updated_at DESC)
    WHERE phone <> '';

CREATE INDEX IF NOT EXISTS idx_tracked_users_email_updated
    ON tracked_users(email, updated_at DESC)
    WHERE email <> '';

-- Acelera resolução e vinculação de grupos multi-tenant (ResolveAndLink)
CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_phone
    ON tracked_users(account_id, phone)
    WHERE phone <> '';

CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_email
    ON tracked_users(account_id, email)
    WHERE email <> '';

-- Acelera listagem e paginação geral por conta
CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_updated
    ON tracked_users(account_id, updated_at DESC);

-- ---------------------------------------------------------------------
-- 3. event_counters
-- ---------------------------------------------------------------------

-- Acelera série temporal e agregados de métricas no Dashboard
CREATE INDEX IF NOT EXISTS idx_event_counters_acc_bucket
    ON event_counters(account_id, bucket_at DESC);

-- ---------------------------------------------------------------------
-- 4. crm_leads
-- ---------------------------------------------------------------------

-- Acelera carregamento das colunas do Kanban CRM ordenadas por atualização
CREATE INDEX IF NOT EXISTS idx_crm_leads_acc_stage_updated
    ON crm_leads(account_id, stage_id, updated_at DESC);

-- ---------------------------------------------------------------------
-- 5. webhook_logs
-- ---------------------------------------------------------------------

-- Acelera busca e replay de webhooks por external_id do visitante
CREATE INDEX IF NOT EXISTS idx_webhook_logs_account_ext
    ON webhook_logs(account_id, external_id)
    WHERE external_id <> '';
