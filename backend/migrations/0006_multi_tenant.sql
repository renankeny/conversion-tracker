-- 0006_multi_tenant.sql
--
-- Adiciona suporte multi-tenant ao TrackFind. Cada conta (account) tem
-- seus próprios pixels, tracked_users e sent_events. Admins podem ser:
--   - role='owner'   → enxerga TUDO (admin master da plataforma)
--   - role='member'  → enxerga apenas account_id da própria conta
--
-- A migração é ADITIVA: cria tabela accounts, cria conta default,
-- adiciona account_id em todas as tabelas relevantes apontando pro default.
-- Nada é apagado.

-- ---------------------------------------------------------------------
-- 1. Tabela de contas
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS accounts (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_accounts_slug ON accounts(slug);

-- Garante que existe a conta default (id=1) pra ancorar dados existentes.
INSERT INTO accounts (id, name, slug)
VALUES (1, 'Conta Principal', 'default')
ON CONFLICT (id) DO NOTHING;

-- Reset da sequência pra evitar conflito quando criar novas contas.
SELECT setval('accounts_id_seq', GREATEST((SELECT MAX(id) FROM accounts), 1));

-- ---------------------------------------------------------------------
-- 2. Adiciona role e account_id em admin_users
-- ---------------------------------------------------------------------
ALTER TABLE admin_users
    ADD COLUMN IF NOT EXISTS account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'owner';

-- Owners enxergam todas as contas. Members enxergam só a sua.
-- Os admins existentes viram 'owner' (não quebra nada).
UPDATE admin_users SET role = 'owner' WHERE role IS NULL OR role = '';

CREATE INDEX IF NOT EXISTS idx_admin_users_account_id
    ON admin_users(account_id) WHERE account_id IS NOT NULL;

-- ---------------------------------------------------------------------
-- 3. Adiciona account_id em pixels
-- ---------------------------------------------------------------------
ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS account_id BIGINT NOT NULL DEFAULT 1
        REFERENCES accounts(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_pixels_account_id ON pixels(account_id);

-- ---------------------------------------------------------------------
-- 4. account_id em tracked_users (denormalizado pra consultas rápidas)
-- ---------------------------------------------------------------------
ALTER TABLE tracked_users
    ADD COLUMN IF NOT EXISTS account_id BIGINT NOT NULL DEFAULT 1
        REFERENCES accounts(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_tracked_users_account_id ON tracked_users(account_id);

-- ---------------------------------------------------------------------
-- 5. account_id em sent_events
-- ---------------------------------------------------------------------
ALTER TABLE sent_events
    ADD COLUMN IF NOT EXISTS account_id BIGINT NOT NULL DEFAULT 1
        REFERENCES accounts(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_sent_events_account_id ON sent_events(account_id);

-- ---------------------------------------------------------------------
-- 6. account_id em event_counters
-- ---------------------------------------------------------------------
ALTER TABLE event_counters
    ADD COLUMN IF NOT EXISTS account_id BIGINT NOT NULL DEFAULT 1
        REFERENCES accounts(id) ON DELETE RESTRICT;

-- Atualiza unique constraint para incluir account_id (idempotente: drop+create).
ALTER TABLE event_counters DROP CONSTRAINT IF EXISTS uq_event_counters_bucket;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_event_counters_account_bucket'
    ) THEN
        ALTER TABLE event_counters
            ADD CONSTRAINT uq_event_counters_account_bucket
            UNIQUE (account_id, pixel_name, event_name, bucket_at);
    END IF;
END$$;

-- ---------------------------------------------------------------------
-- 7. account_id em url_mappings
-- ---------------------------------------------------------------------
ALTER TABLE url_mappings
    ADD COLUMN IF NOT EXISTS account_id BIGINT NOT NULL DEFAULT 1
        REFERENCES accounts(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_url_mappings_account_id ON url_mappings(account_id);

-- ---------------------------------------------------------------------
-- 8. Comentários para documentação
-- ---------------------------------------------------------------------
COMMENT ON TABLE accounts IS
    'Contas (clientes do TrackFind). Cada uma tem seus próprios pixels e dados isolados.';

COMMENT ON COLUMN admin_users.role IS
    '"owner" enxerga todas as contas. "member" enxerga apenas a própria account_id.';

COMMENT ON COLUMN admin_users.account_id IS
    'NULL para owners (acesso global). Obrigatório para members.';
