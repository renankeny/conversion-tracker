-- 0028_custom_webhooks.sql

-- Tabela para armazenar os mapeamentos customizados de webhooks.
-- Pode ser global (account_id = NULL) para o superadmin fornecer para todos,
-- ou por tenant (account_id != NULL).
CREATE TABLE IF NOT EXISTS custom_webhook_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id BIGINT REFERENCES accounts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    mapping_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_webhook_account_id ON custom_webhook_templates(account_id);
