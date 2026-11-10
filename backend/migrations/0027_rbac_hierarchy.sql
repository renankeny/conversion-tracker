-- 0027_rbac_hierarchy.sql
-- Adiciona a hierarquia de RBAC: Super Admin -> Admin -> Manager -> Employee
-- Adiciona parent_id e permissions em admin_users
-- Adiciona owner_id em accounts

BEGIN;

-- 1. admin_users
ALTER TABLE admin_users
    ADD COLUMN IF NOT EXISTS parent_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS permissions JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_admin_users_parent_id ON admin_users(parent_id);

-- Promove o primeiro owner para superadmin apenas se ainda não existir um superadmin
UPDATE admin_users
SET role = 'superadmin'
WHERE id = (
    SELECT id FROM admin_users
    WHERE role IN ('superadmin', 'owner')
    ORDER BY CASE WHEN role = 'superadmin' THEN 0 ELSE 1 END, id ASC
    LIMIT 1
)
AND NOT EXISTS (SELECT 1 FROM admin_users WHERE role = 'superadmin');

UPDATE admin_users SET role = 'admin' WHERE role = 'owner';
UPDATE admin_users SET role = 'manager' WHERE role = 'member';

-- 2. accounts
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_owner_id ON accounts(owner_id);

-- Vincular conta 1 ao superadmin inicial caso ainda não possua owner_id
UPDATE accounts
SET owner_id = (SELECT id FROM admin_users WHERE role = 'superadmin' ORDER BY id ASC LIMIT 1)
WHERE id = 1 AND owner_id IS NULL;

COMMIT;
