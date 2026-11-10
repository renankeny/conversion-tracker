-- =====================================================================
-- Migration 0040 — Agrupamento de identidade acima do external_id
--
-- Múltiplos external_id (navegadores/dispositivos diferentes) da mesma
-- pessoa passam a poder ser ligados via group_id, quando telefone ou email
-- batem. Nada é apagado ou reescrito — external_id continua sendo a chave
-- de cada evento/sessão, group_id é só uma camada de agregação por cima.
--
-- group_id nunca é serializado em resposta de API — uso interno apenas
-- (queries de agregação de LTV, listagem, CRM).
-- =====================================================================

CREATE TABLE IF NOT EXISTS groups (
    id         BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_tracked_users_group_id ON tracked_users(group_id) WHERE group_id IS NOT NULL;

ALTER TABLE crm_leads ADD COLUMN IF NOT EXISTS group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_crm_leads_account_group
    ON crm_leads(account_id, group_id) WHERE group_id IS NOT NULL;
