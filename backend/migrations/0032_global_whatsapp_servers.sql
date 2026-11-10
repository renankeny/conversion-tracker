-- =====================================================================
-- Migration 0032 — Global WhatsApp Servers
-- Transforma os servidores de WhatsApp em infraestrutura global.
-- =====================================================================

-- Remove a dependência de account_id do índice ativo
DROP INDEX IF EXISTS idx_whatsapp_servers_active;

-- Remove a coluna account_id de whatsapp_servers
ALTER TABLE whatsapp_servers DROP COLUMN IF EXISTS account_id;

-- Desativa todos os servidores apenas na primeira execução da migração para evitar conflito de index único global
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'idx_whatsapp_servers_active_global') THEN
        UPDATE whatsapp_servers SET is_active = FALSE;
    END IF;
END
$$;

-- Cria um índice que garante que apenas 1 servidor seja o ativo na plataforma inteira
CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_servers_active_global ON whatsapp_servers(is_active) WHERE is_active = TRUE;
