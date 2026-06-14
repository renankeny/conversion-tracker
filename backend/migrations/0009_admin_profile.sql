-- 0009_admin_profile.sql
--
-- Perfil do admin (dono da agência): nome + telefone para exibição no
-- painel e contato. Senha continua em password_hash; e-mail continua
-- sendo o identificador único de login.

ALTER TABLE admin_users
    ADD COLUMN IF NOT EXISTS name  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN admin_users.name IS 'Nome de exibição no painel (sidebar, perfil).';
COMMENT ON COLUMN admin_users.phone IS 'Telefone de contato — uso administrativo apenas.';
