-- =====================================================================
-- 0077_campaign_rules.sql
-- Regras Automáticas de Campanhas ("Piloto Automático" baseado em venda
-- real) para Meta Ads v26.0 e Google Ads v26, com modo simulação
-- (dry_run), cooldown por objeto, agendamento/dayparting, reativação
-- por venda atrasada e histórico auditável com suporte a Desfazer.
-- =====================================================================

CREATE TABLE IF NOT EXISTS campaign_rules (
    id                BIGSERIAL PRIMARY KEY,
    account_id        BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    template_key      TEXT NOT NULL DEFAULT '',
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    dry_run           BOOLEAN NOT NULL DEFAULT FALSE,
    platform          TEXT NOT NULL DEFAULT 'all',
    pixel_name        TEXT NOT NULL DEFAULT '',
    level             TEXT NOT NULL DEFAULT 'adset',
    attribution_mode  TEXT NOT NULL DEFAULT 'last_click',
    rule_kind         TEXT NOT NULL DEFAULT 'metric',
    period_preset     TEXT NOT NULL DEFAULT 'today',
    conditions        JSONB NOT NULL DEFAULT '[]'::jsonb,
    action_type       TEXT NOT NULL DEFAULT 'pause',
    action_value      DOUBLE PRECISION NOT NULL DEFAULT 0,
    min_budget        DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_budget        DOUBLE PRECISION NOT NULL DEFAULT 0,
    name_contains     TEXT NOT NULL DEFAULT '',
    name_excludes     TEXT NOT NULL DEFAULT '',
    target_status     TEXT NOT NULL DEFAULT 'ACTIVE',
    frequency_minutes INT NOT NULL DEFAULT 30,
    min_hour_sp       INT NOT NULL DEFAULT 0,
    schedule_time     TEXT NOT NULL DEFAULT '',
    schedule_days     JSONB NOT NULL DEFAULT '[]'::jsonb,
    cooldown_minutes  INT NOT NULL DEFAULT 360,
    notify_push       BOOLEAN NOT NULL DEFAULT TRUE,
    notify_whatsapp   BOOLEAN NOT NULL DEFAULT FALSE,
    whatsapp_phone    TEXT NOT NULL DEFAULT '',
    last_run_at       TIMESTAMPTZ,
    last_triggered_at TIMESTAMPTZ,
    last_error        TEXT NOT NULL DEFAULT '',
    trigger_count     INT NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_rules_account
    ON campaign_rules(account_id, enabled, id DESC);
CREATE INDEX IF NOT EXISTS idx_campaign_rules_enabled
    ON campaign_rules(enabled)
    WHERE enabled = TRUE;

CREATE TABLE IF NOT EXISTS campaign_rule_logs (
    id                BIGSERIAL PRIMARY KEY,
    account_id        BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    rule_id           BIGINT REFERENCES campaign_rules(id) ON DELETE SET NULL,
    rule_name         TEXT NOT NULL,
    platform          TEXT NOT NULL,
    level             TEXT NOT NULL,
    entity_subtype    TEXT NOT NULL DEFAULT '',
    object_id         TEXT NOT NULL,
    parent_id         TEXT NOT NULL DEFAULT '',
    object_name       TEXT NOT NULL,
    pixel_name        TEXT NOT NULL DEFAULT '',
    account_ref_id    BIGINT NOT NULL DEFAULT 0,
    ad_account_id     TEXT NOT NULL DEFAULT '',
    login_customer_id TEXT NOT NULL DEFAULT '',
    budget_id         TEXT NOT NULL DEFAULT '',
    currency          TEXT NOT NULL DEFAULT 'BRL',
    action_type       TEXT NOT NULL,
    dry_run           BOOLEAN NOT NULL DEFAULT FALSE,
    status            TEXT NOT NULL,
    previous_status   TEXT NOT NULL DEFAULT '',
    new_status        TEXT NOT NULL DEFAULT '',
    previous_budget   DOUBLE PRECISION NOT NULL DEFAULT 0,
    new_budget        DOUBLE PRECISION NOT NULL DEFAULT 0,
    metrics_snapshot  JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason            TEXT NOT NULL DEFAULT '',
    error_message     TEXT NOT NULL DEFAULT '',
    undone_at         TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_rule_logs_account_created
    ON campaign_rule_logs(account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_campaign_rule_logs_cooldown
    ON campaign_rule_logs(rule_id, object_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_campaign_rule_logs_reactivate
    ON campaign_rule_logs(account_id, platform, object_id, action_type, status, created_at DESC);
