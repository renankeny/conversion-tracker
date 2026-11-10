-- Alinhamento com o sistema legado:
-- - id_wpp: id curto (10 chars [A-Z0-9]) gerado no client e usado no protocolo do WhatsApp
-- - gclid: Google Click ID (capturado pelo pixel.js mesmo sem Google Ads ativo)
-- - índice em email pra fallback de webhook (RkCursos, etc.)
-- - url_mappings: tabela de mapeamento black -> white de URLs

ALTER TABLE tracked_users
    ADD COLUMN IF NOT EXISTS id_wpp TEXT NOT NULL DEFAULT '';

ALTER TABLE tracked_users
    ADD COLUMN IF NOT EXISTS gclid TEXT NOT NULL DEFAULT '';

-- id_wpp é único quando preenchido (vazio é permitido pra múltiplas linhas)
CREATE UNIQUE INDEX IF NOT EXISTS uq_tracked_users_id_wpp
    ON tracked_users(id_wpp)
    WHERE id_wpp <> '';

CREATE INDEX IF NOT EXISTS idx_tracked_users_email_lookup
    ON tracked_users(email)
    WHERE email <> '';

-- ---------------------------------------------------------------------
-- url_mappings
-- Mapeamento de URLs "black" (origem) para "white" (destino), aplicado
-- antes de enviar event_source_url pra Meta. Match em ordem de
-- especificidade: <scheme>://<host><path> > <host><path> > <host>.
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS url_mappings (
    id          BIGSERIAL PRIMARY KEY,
    source      TEXT NOT NULL UNIQUE,           -- chave do match (URL completa, host+path ou host)
    target      TEXT NOT NULL,                  -- URL "white" final
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_url_mappings_enabled ON url_mappings(enabled);
