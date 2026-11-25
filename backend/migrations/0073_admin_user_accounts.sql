-- =====================================================================
-- 0073_admin_user_accounts.sql
-- Permite vincular um usuário (admin, manager ou employee) a quantas
-- empresas (accounts) desejar via tabela associativa N:N.
-- Mantém admin_users.account_id sincronizado como empresa padrão inicial.
-- =====================================================================

CREATE TABLE IF NOT EXISTS admin_user_accounts (
    user_id    BIGINT NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_admin_user_accounts_account_id
    ON admin_user_accounts(account_id);

INSERT INTO admin_user_accounts (user_id, account_id)
SELECT id, account_id
FROM admin_users
WHERE account_id IS NOT NULL
ON CONFLICT (user_id, account_id) DO NOTHING;
