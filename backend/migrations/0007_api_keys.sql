-- 0007_api_keys.sql
--
-- API keys de servidor: alternativa ao JWT admin para integrações
-- programáticas (CI, scripts internos, ingestão server-to-server).
--
-- Cada API key é vinculada a uma conta (account_id) e atua como um "member"
-- daquela conta — pode ler/escrever dados da conta mas não consegue gerenciar
-- outras contas nem criar admins.
--
-- Armazenamos apenas o SHA-256 do segredo. O plaintext (`trk_<base64>`) é
-- mostrado uma única vez, na criação. `key_prefix` é o início legível (8
-- chars) usado pra identificar a key no painel sem revelar o segredo.

CREATE TABLE IF NOT EXISTS api_keys (
    id          BIGSERIAL PRIMARY KEY,
    account_id  BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    key_prefix  TEXT NOT NULL,                  -- "trk_xxxx" (8 chars) só pra display
    key_hash    TEXT NOT NULL UNIQUE,           -- sha256(plaintext) hex
    last_used_at TIMESTAMPTZ,                   -- atualizado on-use (nullable)
    revoked_at  TIMESTAMPTZ,                    -- NULL = ativa
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_account_id ON api_keys(account_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(account_id) WHERE revoked_at IS NULL;

COMMENT ON TABLE api_keys IS
    'Tokens server-to-server vinculados a uma conta. Atuam como member da conta.';
COMMENT ON COLUMN api_keys.key_hash IS
    'SHA-256 hex do plaintext. O segredo nunca é armazenado em claro.';
COMMENT ON COLUMN api_keys.key_prefix IS
    'Primeiros chars do segredo (formato trk_xxxx) para exibição no painel.';
