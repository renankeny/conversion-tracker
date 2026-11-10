-- 0008_account_retention.sql
--
-- Permite que cada conta defina sua própria janela de retenção, sobrescrevendo
-- as envs globais SENT_EVENTS_RETENTION_DAYS / EVENT_COUNTERS_RETENTION_DAYS.
--
-- Convenção:
--   NULL  → usa o default global da env
--   0     → desativa a retenção dessa categoria pra essa conta (mantém pra sempre)
--   N>0   → mantém apenas os últimos N dias

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS sent_events_retention_days    INT,
    ADD COLUMN IF NOT EXISTS event_counters_retention_days INT;

COMMENT ON COLUMN accounts.sent_events_retention_days IS
    'NULL=usa env global, 0=desativa, N>0=mantém últimos N dias';
COMMENT ON COLUMN accounts.event_counters_retention_days IS
    'NULL=usa env global, 0=desativa, N>0=mantém últimos N dias';
