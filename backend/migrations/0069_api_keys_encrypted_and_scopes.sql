-- 0069_api_keys_encrypted_and_scopes.sql
-- Permite visualizar a chave completa (armazenada com AES-256-GCM em repouso)
-- e configurar escopos granulares de permissão + modo somente leitura (API REST + MCP).

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS key_encrypted TEXT NOT NULL DEFAULT '';

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS scopes TEXT[] NOT NULL DEFAULT ARRAY['*']::TEXT[];

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS read_only BOOLEAN NOT NULL DEFAULT FALSE;
