-- Migração 0065: expande platform_tokens.token_secret para TEXT (suporte a AES-256-GCM)
-- e cria índice composto para resumo rápido por plataforma em webhook_logs.

ALTER TABLE platform_tokens
    ALTER COLUMN token_secret TYPE TEXT;

CREATE INDEX IF NOT EXISTS idx_webhook_logs_account_source_received
    ON webhook_logs(account_id, source, received_at DESC);
